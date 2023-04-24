package utils

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/xigxog/kubefox-cli/internal/log"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return !info.IsDir()
}

func EnsureDirForFile(path string) {
	EnsureDir(filepath.Dir(path))
}

func EnsureDir(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatal("Error creating directory: %s", err)
	}
}

func IsDirEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal("%v", err)
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}

	return false
}

func CheckAllEmpty(strs ...string) bool {
	for _, s := range strs {
		if s != "" {
			return false
		}
	}

	return true
}
