package git

import (
	"app/pkg/git/model"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
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
func getFileHistory(repositoryPath, path, repo, token string) (model.File, error) {
	var commits []model.Commit

	filePathSlice := strings.Split(path, "/")
	filename := filePathSlice[len(filePathSlice)-1]

	fmt.Println("   Reading commits from \033[34m" + filename + "\033[0m workflow file")

	type commit struct {
		Sha    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}

	commitsRaw := []commit{}
	page := 1

	for {
		uri := fmt.Sprintf("repos/%s/commits?path=%s&page=%d&per_page=100", 
			repo, strings.ReplaceAll(path, "/", "%2F"), page)

		client := &http.Client{}

		req, err := http.NewRequest("GET", "https://api.github.com/"+uri, nil)

		if err != nil {
			break
		}

		req.Header.Set("Authorization", "Bearer "+token)
		res, err := client.Do(req)

		if res.StatusCode != 200 || err != nil {
			fmt.Println(res.StatusCode)
			fmt.Println(err)
			break
		}

		var commits []commit
		err = json.NewDecoder(res.Body).Decode(&commits)
		
		if err != nil || len(commits) == 0 {
			break
		}

		commitsRaw = append(commitsRaw, commits...)
		
		page++
	}

	for _, commit := range commitsRaw {
		date, err := time.Parse("2006-01-02T15:04:05Z", commit.Commit.Committer.Date)

		if err != nil {
			return model.File{}, err
		}

		content, err := getContent(repositoryPath, path, commit.Sha)

		if err != nil {
			continue
		}

		fmt.Print("      Extracting components from \033[34m" +
			commit.Sha + "\033[0m \033[37m[" + date.String() + "]\033[0m commit")

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
		commitStruct.Init(commit.Sha, date, content, components)

		commits = append(commits, commitStruct)
	}

	fmt.Println("   \033[32m✓\033[0m (" + strconv.Itoa(len(commits)) + " commits extracted)\n")

	fileStruct := model.File{}
	fileStruct.Init(filename, path, commits, nil)

	return fileStruct, nil
}
