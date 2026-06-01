package discover

import (
	"os"
	"path/filepath"
	"strings"
)

func DetectLanguages(dir string) []string {
	var langs []string
	hasGo := false
	hasPy := false

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == ".git" || base == "node_modules" || base == "__pycache__" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" && !hasGo {
			hasGo = true
			langs = append(langs, "go")
		}
		if ext == ".py" && !hasPy {
			hasPy = true
			langs = append(langs, "python")
		}
		if hasGo && hasPy {
			return filepath.SkipAll
		}
		return nil
	})

	return langs
}
