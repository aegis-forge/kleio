package git

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

// GetContent returns the content of a file as a string given its commit hash
func GetContent(repositoryPath string, filePath string, hash string) (string, error) {
	cmd := exec.Command("git", "-C", repositoryPath, "show", fmt.Sprintf("%s:%s", hash, filePath))
	out, err := cmd.Output()

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(out), nil
}

// GetHistory returns a slice of [Commit] structs given
func GetHistory(repositoryPath string, filePath string) ([]Commit, error) {
	var commits []Commit

	cmd := exec.Command("git", "-C", repositoryPath, "log", "--follow", "--", filePath)
	out, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	r := regexp.MustCompile(`commit\s+(\S{40})\n.*\nDate:\s+(.+)`)
	rawCommits := r.FindAllStringSubmatch(string(out), -1)

	for commit := range rawCommits {
		date, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", rawCommits[commit][2])

		if err != nil {
			return nil, err
		}

		content, err := GetContent(repositoryPath, filePath, rawCommits[commit][1])

		if err != nil {
			return nil, err
		}

		commits = append(commits, Commit{
			Hash:    rawCommits[commit][1],
			Date:    date,
			Content: content,
		})
	}

	return commits, nil
}
