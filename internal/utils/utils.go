package utils

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xigxog/fox/internal/log"
)

var specChars = regexp.MustCompile(`[^a-z0-9]`)

func Wd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting working dir: %v", err)
	}

	return filepath.Clean(wd)
}

func Find(file, path, stop string) string {
	log.Verbose("looking for %s in %s, stopping at %s", file, path, stop)
	if _, err := os.Stat(filepath.Join(path, file)); err == nil {
		log.Verbose("found %s in %s", file, path)
		return path
	}

	if path == stop || path == string(filepath.Separator) {
		return ""
	}

	return Find(file, filepath.Join(path, ".."), stop)
}

func Subpath(path, root string) string {
	return "" +
		strings.TrimPrefix( // trim separator
			strings.TrimPrefix( // trim repo path
				path,
				root,
			),
			string(filepath.Separator),
		)
}

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

	return err == io.EOF
}

func CheckAllEmpty(strs ...string) bool {
	for _, s := range strs {
		if s != "" {
			return false
		}
	}

	return true
}

func Clean(path string) string {
	b := filepath.Base(path)
	b = strings.ToLower(b)
	b = specChars.ReplaceAllString(b, "-")
	b = strings.TrimPrefix(strings.TrimSuffix(b, "-"), "-")
	return b
}

func YesNoPrompt(prompt string, def bool) bool {
	if def {
		log.Printf(prompt + " [Y/n] ")
	} else {
		log.Printf(prompt + " [y/N] ")
	}

	var input string
	fmt.Scanln(&input)
	if input == "" {
		return def
	}
	input = strings.ToLower(input)
	switch input {
	case "y":
		return true
	case "n":
		return false
	default:
		return YesNoPrompt(prompt, def)
	}
}

func InputPrompt(prompt, def string, required bool) string {
	log.Printf(prompt)
	if def != "" {
		log.Printf(" (default '%s')", def)
	} else if required {
		log.Printf(" (required)")
	}
	log.Printf(": ")

	var input string
	fmt.Scanln(&input)
	if input == "" {
		input = def
	}
	if required && input == "" {
		return InputPrompt(prompt, def, required)
	}
	return input
}

func NamePrompt(what, def string, required bool) string {
	name := InputPrompt(fmt.Sprintf("Enter the %s's name", what), def, required)
	if name != Clean(name) {
		log.Error("The %s's name is invalid.", what)
		if YesNoPrompt(fmt.Sprintf("Would you like to use '%s' instead", Clean(name)), true) {
			return Clean(name)
		} else {
			return NamePrompt(what, def, required)
		}
	}
	return name
}
