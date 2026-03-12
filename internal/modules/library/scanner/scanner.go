package scanner

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

type FileEntry struct {
	Path    string
	Size    int64
	ModTime time.Time
}

type FilesystemScanner struct{}

func NewFilesystemScanner() *FilesystemScanner {
	return &FilesystemScanner{}
}

var supportedExtensions = map[string]struct{}{
	".mp3":  {},
	".flac": {},
	".ogg":  {},
	".m4a":  {},
}

func (s *FilesystemScanner) Scan(rootPath string) ([]FileEntry, error) {
	files := make([]FileEntry, 0)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := supportedExtensions[ext]; !ok {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		files = append(files, FileEntry{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
