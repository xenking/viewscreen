package viewscreen

import (
	"github.com/xenking/viewscreen/viewscreen/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Download struct {
	ID      string
	Created time.Time
}

type Episode struct {
	ID     string
	Number int
}

func (dl Download) Thumbnailfile() string {
	return filepath.Join(dl.Path(), "thumbnail.png")
}

func (dl Download) Thumbnail() bool {
	_, err := os.Stat(dl.Thumbnailfile())
	return err == nil
}

func (dl Download) Uploadingfile() string {
	return dl.Path() + ".uploading"
}

func (dl Download) Uploading() bool {
	_, err := os.Stat(dl.Uploadingfile())
	return err == nil
}

func (dl Download) Downloadingfile() string {
	return dl.Path() + ".downloading"
}

func (dl Download) Downloading() bool {
	_, err := os.Stat(dl.Downloadingfile())
	return err == nil
}

func (dl Download) Sharefile() string {
	return filepath.Join(downloadDir, ".shared", dl.ID)
}

func (dl Download) Shared() bool {
	_, err := os.Stat(dl.Sharefile())
	return err == nil
}

func (dl Download) Share() error {
	if dl.Shared() {
		return nil
	}
	// Ensure the sharing directory exists first.
	path := filepath.Dir(dl.Sharefile())
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	_, err := os.Create(dl.Sharefile())
	return err
}

func (dl Download) Unshare() error {
	if !dl.Shared() {
		return nil
	}
	return os.Remove(dl.Sharefile())
}

func (dl Download) Rename(id string) error {
	// TODO: Recursive rename all files with the same dl.ID to new id
	// dl.Files()
	oldpath := filepath.Join(downloadDir, dl.ID)
	oldpath = filepath.Clean(oldpath)
	if oldpath == downloadDir {
		logger.Debugf("old path %q download %q", oldpath, dl.ID)
	}
	newpath := filepath.Join(downloadDir, id)
	newpath = filepath.Clean(newpath)
	if newpath == downloadDir {
		logger.Debugf("new path %q download %q", newpath, id)
	}
	return utils.RenameDir(oldpath, newpath, true)
}

func (dl Download) Path() string {
	path := filepath.Join(downloadDir, dl.ID)
	path = filepath.Clean(path)
	if path == downloadDir {
		logger.Debugf("path %q download %q", path, dl.ID)
		panic("invalid or missing download ID")
	}
	return path
}

func (dl Download) ModTime() time.Time {
	fi, _ := os.Stat(dl.Path())
	return fi.ModTime()
}

func (dl Download) Size() int64 {
	var size int64
	filepath.Walk(dl.Path(), func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func (dl Download) Files(thumbnails, toTranscode bool) []File {
	var files []File
	filepath.Walk(dl.Path(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !thumbnails {
			if strings.HasSuffix(info.Name(), "thumbnail.png") {
				return nil
			}
			if strings.Contains(path, "/thumb") {
				return filepath.SkipDir
			}
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		if toTranscode {
			ext := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
			switch ext {
			case "mkv", "avi", "flv":
			default:
				return nil
			}
		}

		// The ID is a relative path from the download's path.
		id := path
		id = strings.TrimPrefix(id, dl.Path())
		id = strings.TrimPrefix(id, "/")

		files = append(files, File{
			ID:   id,
			Info: info,
			Path: path,
		})
		return nil
	})
	return files
}

func (dl Download) GetEpisodes() ([]Episode, error) {
	var episodes []Episode
	num := 1
	err := filepath.Walk(dl.Path(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !!info.IsDir() {
			return nil
		}
		if dl.Path() == filepath.Dir(path) {
			ext := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
			switch ext {
			case "mp4", "m4v", "webm":
				episodes = append(episodes, Episode{
					ID:     info.Name(),
					Number: num,
				})
				num += 1
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return episodes, nil
}

func (dl Download) GetCurrentEpisode(current string) int {
	eps, err := dl.GetEpisodes()
	if err != nil {
		return 0
	}

	for i := range eps {
		if eps[i].ID == current {
			return eps[i].Number - 1
		}
	}
	return 0
}

func (dl Download) GetFonts() []string {
	var fonts []string
	fontsdir := filepath.Join(dl.Path(), "subs", "fonts")
	err := filepath.Walk(fontsdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !!info.IsDir() {
			return nil
		}
		ext := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
		if ext == "ttf" {
			fonts = append(fonts, info.Name())
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return fonts
}

func (ep Episode) Name() string {
	return strings.TrimSuffix(ep.ID, filepath.Ext(ep.ID))
}

func (dl Download) FindFile(id string) (File, error) {
	thumbnails := false
	if strings.Contains(id, "thumbnail") || strings.Contains(id, "thumb/") {
		thumbnails = true
	}
	for _, file := range dl.Files(thumbnails, false) {
		if id == file.ID {
			return file, nil
		}
	}
	return File{}, ErrFileNotFound
}
