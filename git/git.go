package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func GetChangedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseFileList(root, string(out)), nil
}

func GetStagedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--cached")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseFileList(root, string(out)), nil
}

func parseFileList(root, output string) []string {
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, filepath.Join(root, line))
	}
	return files
}
