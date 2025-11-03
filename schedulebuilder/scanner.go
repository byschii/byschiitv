package main

import (
	"os"
	"path/filepath"
	"strings"
)

var mediaExtensions = map[string]struct{}{
	".mp4": {}, ".mkv": {}, ".avi": {}, ".mov": {}, ".flv": {}, ".wmv": {},
	".mpg": {}, ".mpeg": {}, ".webm": {}, ".m4v": {}, ".ts": {},
}

// scanMedia walks the provided directory and returns a list of media files (relative paths)
func scanMedia(root string) []string {
	var files []string

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := mediaExtensions[ext]; ok {
			rel := path
			if r, err := filepath.Rel(root, path); err == nil {
				rel = r
			} else {
				rel = strings.Replace(path, root+string(os.PathSeparator), "", 1)
			}
			files = append(files, rel)
		}
		return nil
	})
	return files
}
