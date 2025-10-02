package scanner

import (
	"os"
	"path/filepath"
)

type Scanner struct {
	basePath string
}

func New(basePath string) *Scanner {
	return &Scanner{basePath: basePath}
}

func (s *Scanner) ScanEmails() ([]string, error) {
	var emailFiles []string

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip directories, continue walking
		if info.IsDir() {
			return nil
		}

		// process files in cur/ or new/ folders
		dir := filepath.Base(filepath.Dir(path))
		if dir == "cur" || dir == "new" {
			emailFiles = append(emailFiles, path)
		}

		return nil

	})
	return emailFiles, err
}
