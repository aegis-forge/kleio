package git

import (
	"app/pkg/git/model"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// getContent returns the content of a file as a string given its commit hash
func getContent(repositoryPath string, filePath string, hash string) (string, error) {
	cmd := exec.Command("git", "-C", repositoryPath, "show", fmt.Sprintf("%s:%s", hash, filePath))
	out, err := cmd.Output()

	if err != nil {
		return "", err
	}

	return string(out), nil
}

// getFileHistory returns a [File] struct containing its history
func getFileHistory(repositoryPath string, path string) (model.File, error) {
	var commits []model.Commit

	filePathSlice := strings.Split(path, "/")
	filename := filePathSlice[len(filePathSlice)-1]

	fmt.Println("   Reading commits from \033[34m" + filename + "\033[0m workflow file")

	cmd := exec.Command("git", "-C", repositoryPath, "log", "--follow", "--", path)
	out, err := cmd.Output()

	if err != nil {
		return model.File{}, err
	}

	r := regexp.MustCompile(`commit\s+(\S{40})\n.*\nDate:\s+(.+)`)
	rawCommits := r.FindAllStringSubmatch(string(out), -1)

	for commit := range rawCommits {
		date, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", rawCommits[commit][2])

		if err != nil {
			return model.File{}, err
		}

		content, err := getContent(repositoryPath, path, rawCommits[commit][1])

		if err != nil {
			continue
		}

		fmt.Print("      Extracting components from \033[34m" +
			rawCommits[commit][1] + "\033[0m \033[37m[" + date.String() + "]\033[0m commit")

		components, err := extractComponents(content)

		if err != nil {
			return model.File{}, err
		}

		if components == nil {
			fmt.Println(" \033[31m𐄂\u001B[0m YAML parsing error")
			continue
		}

		acc := 0

		for _, component := range components {
			acc += component.GetAllUses()
		}

		fmt.Print(" \033[32m✓\033[0m (" + strconv.Itoa(len(components)) +
			" components extracted / " + strconv.Itoa(acc) + " total uses)\n")

		commitStruct := model.Commit{}
		commitStruct.Init(rawCommits[commit][1], date, content, components)

		commits = append(commits, commitStruct)
	}

	fmt.Println("   \033[32m✓\033[0m (" + strconv.Itoa(len(commits)) + " commits extracted)\n")

	fileStruct := model.File{}
	fileStruct.Init(filename, path, commits, nil)

	return fileStruct, nil
}
