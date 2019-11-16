package transcoder

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/xenking/viewscreen/viewscreen/downloader"
	"github.com/xenking/viewscreen/viewscreen/utils"
)

type Transcoder struct {
	sync.RWMutex
	concurrency int
	queue       []string
	running     map[string]*exec.Cmd
}

func NewTranscoder() *Transcoder {
	t := &Transcoder{}
	t.running = make(map[string]*exec.Cmd)
	t.concurrency = runtime.NumCPU()
	go t.manager()
	return t
}

func (t *Transcoder) manager() {
	for {
		t.Lock()
		if len(t.queue) > 0 && len(t.running) < t.concurrency {
			srcname := t.queue[0]
			t.queue = t.queue[1:]
			log.Debugf("job manager adding %q", srcname)
			go t.transcode(srcname)
		}
		t.Unlock()
		time.Sleep(5 * time.Second)
	}
}

func (t *Transcoder) queued(srcname string) bool {
	for _, job := range t.queue {
		if job == srcname {
			return true
		}
	}
	return false
}

func (t *Transcoder) dequeue(srcname string) {
	var keep []string
	for _, job := range t.queue {
		if job == srcname {
			continue
		}
		keep = append(keep, job)
	}
	t.queue = keep
}

func (t *Transcoder) Cancel(srcname string) error {
	t.Lock()
	defer t.Unlock()

	if t.queued(srcname) {
		log.Infof("dequeing %q", srcname)
		t.dequeue(srcname)
		return nil
	}

	// must be an active job now or it doesn't exist.
	cmd, ok := t.running[srcname]
	if !ok {
		return fmt.Errorf("no transcoding job found")
	}
	// it's actually running, so kill it.
	if cmd.Process != nil {
		log.Infof("killing transcode job %q", srcname)
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transcoder) filenames(srcname string) (string, string, string) {
	srcname = filepath.Clean(srcname)
	dir := filepath.Dir(srcname)           // "/some dir"
	ext := filepath.Ext(srcname)           // ".avi"
	base := filepath.Base(srcname)         // "somewhere.avi"
	noext := strings.TrimSuffix(base, ext) // "somewhere"

	tmpname := fmt.Sprintf("%s/.%s.mp4", dir, noext)
	dstname := fmt.Sprintf("%s/%s.mp4", dir, noext)
	return srcname, tmpname, dstname
}

func (t *Transcoder) Busy() bool {
	t.RLock()
	defer t.RUnlock()
	return len(t.queue) > 0 || len(t.running) > 0
}

func (t *Transcoder) QueueCount() int {
	t.RLock()
	defer t.RUnlock()
	return len(t.queue)
}

func (t *Transcoder) RunningCount() int {
	t.RLock()
	defer t.RUnlock()
	return len(t.running)
}

func (t *Transcoder) Active(srcname string) bool {
	t.RLock()
	defer t.RUnlock()

	// check if waiting in queued
	if t.queued(srcname) {
		return true
	}

	// check if it's actually running
	cmd, ok := t.running[srcname]
	if !ok {
		return false
	}
	if cmd.Process == nil {
		return false
	}
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (t *Transcoder) Add(srcname string) error {
	fi, err := os.Stat(srcname)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("must be a file (not a dir)")
	}

	// return if already queued.
	t.RLock()
	if t.queued(srcname) {
		t.RUnlock()
		return nil
	}
	t.RUnlock()

	// return if already running.
	t.RLock()
	if _, ok := t.running[srcname]; ok {
		t.RUnlock()
		return nil
	}
	t.RUnlock()

	t.Lock()
	t.queue = append(t.queue, srcname)
	t.Unlock()
	return nil
}

func (t *Transcoder) transcode(srcname string) {
	srcname, tmpname, dstname := t.filenames(srcname)

	/*	srcfi, err := os.Stat(srcname)
		if err != nil {
			log.Errorf("job %q: %s", srcname, err)
			return
		}
	*/
	// Find ffmpeg
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Error(err)
		return
	}
	cmd, err := exec.Command(ffmpeg,
		"-y",
		"-i", srcname,
		"-codec:v", "libx264",
		"-crf", "17",
		"-bf", "2",
		"-flags", "+cgop",
		"-pix_fmt", "yuv420p",
		"-codec:a", "aac",
		"-strict", "-2",
		"-b:a", "384k",
		"-r:a", "48000",
		"-movflags", "faststart", // make streaming work
		"-max_muxing_queue_size", "500", // handle sparse audio/video frames (see: https://trac.ffmpeg.org/ticket/6375#comment:2)
		tmpname,
	), nil
	if err != nil {
		log.Errorf("ffmpeg failed: %s", err)
		return
	}

	// Add as a running job.
	log.Infof("adding transcode job %q -> %q", srcname, dstname)
	t.Lock()
	t.running[srcname] = cmd
	t.Unlock()

	// Remove on completion.
	defer func() {
		t.Lock()
		delete(t.running, srcname)
		t.Unlock()

		// Remove the temp file if it still exists at this point.
		os.Remove(tmpname)
	}()

	// Transcode
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("job %q: %s", srcname, string(output))
		return
	}

	// Rename temp file to real file.
	if err := utils.RenameFile(tmpname, dstname); err != nil {
		log.Errorf("job %q: %s", srcname, err)
		return
	}

	// Check our new file is the same size.

	srcinfo, err := downloader.Ffprobe(srcname)
	if err != nil {
		log.Errorf("job %q: %s", srcname, err)
		return
	}
	dstinfo, err := downloader.Ffprobe(dstname)
	if err != nil {
		log.Errorf("job %q: %s", dstname, err)
		return
	}

	log.Debugf("transcode complete: src duration = %f, dst duration = %f", srcinfo.Format.Duration, dstinfo.Format.Duration)
	if int(srcinfo.Format.Duration+0.01) > int(dstinfo.Format.Duration) {
		log.Errorf("job %q: transcoded is too small (%f vs %f); deleting.", srcname, srcinfo.Format.Duration, dstinfo.Format.Duration)
		if err := os.Remove(dstname); err != nil {
			log.Error(err)
		}
		return
	}

	// Rename the old thumbnail if it exists.
	oldthumb := srcname + ".thumbnail.png"
	newthumb := dstname + ".thumbnail.png"
	if _, err := os.Stat(oldthumb); err == nil {
		if err := os.Rename(oldthumb, newthumb); err != nil {
			log.Errorf("job %q: %s", srcname, err)
			return
		}
	}

	// Remove the source file.
	if err := os.Remove(srcname); err != nil {
		log.Errorf("job %q: %s", srcname, err)
		return
	}
	if err := GenerateContactSheet(dstname); err != nil {
		log.Errorf("job generate contact sheet failed: %s", dstname, err)
		return
	}
}

func GenerateContactSheet(videofile string) error {
	// Adding contact sheet
	dstdir, videoname := filepath.Split(videofile)
	thumbdir := filepath.Join(dstdir, "thumb", videoname)
	thumbfile := filepath.Join(dstdir, "thumb", videoname, "thumbnail.jpg")
	if err := downloader.EnsureDir(thumbdir); err != nil {
		return fmt.Errorf("mkdir failed: %s (%s)", thumbdir, err)
	}
	if err := contactsheet(videofile, thumbdir); err != nil {
		return fmt.Errorf("contact sheet failed: %s (%s)", videofile, err)
	}
	// Adding big preview thumbnail
	if err := downloader.Ffthumb(videofile, thumbfile, "thumbnail,fps=1/6", true); err != nil {
		return fmt.Errorf("ffthumb failed: %s (%s)", videofile, err)
	}
	return nil
}

func contactsheet(videofile, thumbdir string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return err
	}

	montage, err := exec.LookPath("montage")
	if err != nil {
		return err
	}

	tmpdir, err := ioutil.TempDir(filepath.Dir(thumbdir), "frames")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	// Create temp frames.
	output, err := exec.Command(ffmpeg,
		"-y",
		"-i", videofile,
		"-f", "image2", "-vsync", "cfr", "-an", "-sn", "-vf", "scale=108:60,fps=1/10", "-b:v", "2000", "-bt", "20M",
		filepath.Join(tmpdir, "frame-%03d.jpeg"),
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %s (%s)", string(output), err)
	}

	output, err = exec.Command(montage,
		filepath.Join(tmpdir, "frame-*.jpeg"),
		"-scenes", "1", "-background", "black", "-quality", "80", "-geometry", "+0+0", "-tile", "5x5",
		filepath.Join(thumbdir, "cs-%d.jpg"),
	).CombinedOutput()
	if err != nil {
		if !strings.Contains(err.Error(), "unable to read font") {
			return fmt.Errorf("montage failed: %s (%s)", string(output), err)
		}
	}
	return nil
}
