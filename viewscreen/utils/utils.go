package utils

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type DiskInfo struct {
	free int64
	used int64
}

type StringSet struct {
	_set map[string]bool
}

func NewStringSet() *StringSet {
	return &StringSet{make(map[string]bool, )}
}

func (set *StringSet) Add(s string) bool {
	_, found := set._set[s]
	set._set[s] = true
	return !found
}

func (set *StringSet) Contains(s string) bool {
	_, found := set._set[s]
	return found
}

func (set *StringSet) Slice() []string {
	arr := make([]string, len(set._set))
	for k := range set._set {
		arr = append(arr, k)
	}
	if arr[0] == "" {
		arr = arr[:0]
	}
	return arr
}

func (set *StringSet) remove(s string) {
	delete(set._set, s)
}

func (set *StringSet) Size() int {
	return len(set._set)
}

func (d *DiskInfo) Total() int64   { return d.free + d.used }
func (d *DiskInfo) TotalMB() int64 { return d.Total() / 1024 / 1024 }
func (d *DiskInfo) TotalGB() int64 { return d.TotalMB() / 1024 }

func (d *DiskInfo) Free() int64   { return d.free }
func (d *DiskInfo) FreeMB() int64 { return d.free / 1024 / 1024 }
func (d *DiskInfo) FreeGB() int64 { return d.FreeMB() / 1024 }

func (d *DiskInfo) Used() int64   { return d.used }
func (d *DiskInfo) UsedMB() int64 { return d.used / 1024 / 1024 }
func (d *DiskInfo) UsedGB() int64 { return d.UsedMB() / 1024 }

func (d *DiskInfo) UsedPercent() float64 {
	return (float64(d.used) / float64(d.Total())) * 100
}

func NewDiskInfo(path string) (*DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("diskinfo failed: %s", err)
	}
	free := stat.Bavail * uint64(stat.Bsize)
	used := (stat.Blocks * uint64(stat.Bsize)) - free
	return &DiskInfo{int64(free), int64(used)}, nil
}

func ListDirectory(path string) ([]os.FileInfo, []os.FileInfo, error) {
	list, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, nil, err
	}
	var dirs []os.FileInfo
	var files []os.FileInfo
	for _, f := range list {
		if strings.HasSuffix(f.Name(), "thumbnail.png") { // skip thumbnail files
			continue
		}
		if strings.HasPrefix(f.Name(), ".") { // skip hidden files
			continue
		}
		if f.IsDir() {
			dirs = append(dirs, f)
		} else {
			files = append(files, f)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[j].Name() > dirs[i].Name() })
	sort.Slice(files, func(i, j int) bool { return files[j].Name() > files[i].Name() })
	return dirs, files, nil
}

func RandomNumber() (int, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint32(b)), nil
}

func Overwrite(filename string, data []byte, perm os.FileMode) error {
	f, err := ioutil.TempFile(filepath.Dir(filename), filepath.Base(filename)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(f.Name(), perm); err != nil {
		return err
	}
	return os.Rename(f.Name(), filename)
}

func RenameDir(src string, dst string, force bool) (err error) {
	err = CopyDir(src, dst, force)
	if err != nil {
		return fmt.Errorf("failed to copy source dir %s to %s: %s", src, dst, err)
	}
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("failed to cleanup source dir %s: %s", src, err)
	}
	return nil
}

func RenameFile(src string, dst string) (err error) {
	if src == dst {
		return nil
	}
	err = CopyFile(src, dst)
	if err != nil {
		return fmt.Errorf("failed to copy source file %s to %s: %s", src, dst, err)
	}
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("failed to cleanup source file %s: %s", src, err)
	}
	return nil
}

// CopyFileOrDir copies the source file or directory to the given destination
func CopyFileOrDir(src string, dst string, force bool) (err error) {
	fi, err := os.Stat(src)
	if err != nil {
		return errors.Wrapf(err, "getting details of file '%s'", src)
	}
	if fi.IsDir() {
		return CopyDir(src, dst, force)
	}
	return CopyFile(src, dst)
}

func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDirPreserve copies from the src dir to the dst dir if the file does NOT already exist in dst
func CopyDirPreserve(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return errors.Wrapf(err, "checking %s exists", src)
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "checking %s exists", dst)
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return errors.Wrapf(err, "creating %s", dst)
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return errors.Wrapf(err, "reading files in %s", src)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDirPreserve(srcPath, dstPath)
			if err != nil {
				return errors.Wrapf(err, "recursively copying %s", entry.Name())
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				err = CopyFile(srcPath, dstPath)
				if err != nil {
					return errors.Wrapf(err, "copying %s to %s", srcPath, dstPath)
				}
			} else if err != nil {
				return errors.Wrapf(err, "checking if %s exists", dstPath)
			}
		}
	}
	return nil
}

// CopyDirOverwrite copies from the source dir to the destination dir overwriting files along the way
func CopyDirOverwrite(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDirOverwrite(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}
	return
}

func CopyDir(src string, dst string, force bool) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		if force {
			os.RemoveAll(dst)
		} else {
			return fmt.Errorf("destination already exists")
		}
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath, force)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}
