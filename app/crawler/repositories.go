package crawler

import (
	"app/pkg/github"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/gosuri/uilive"
	"gopkg.in/ini.v1"
)

type Repos struct {
	Repo []struct {
		Url string `json:"html_url"`
	} `json:"items"`
}

// writeToFile takes all the retrieved URLs and saves them in a file
func writeToFile(urls []string) error {
	_, filename, _, _ := runtime.Caller(0)
	file, err := os.Create(path.Join(path.Dir(filename), "../../out/repositories.txt"))

	if err != nil {
		return err
	}

	for _, url := range urls {
		_, err = file.WriteString(url + "\n")

		if err != nil {
			return err
		}
	}

	err = file.Close()

	if err != nil {
		return err
	}

	return nil
}

// GetTopRepositories saves all retrieved top repository URLs to a file
func getTopRepositories(config *ini.Section) error {
	var urls []string

	ghToken := config.Key("TOKEN").String()
	ghPageSize, err := config.Key("SIZE").Int()
	ghPages, err := config.Key("PAGES").Int()

	if err != nil {
		return err
	}

	writer := uilive.New()
	writer.Start()

	i := 1

	for page := range ghPages {
		url := fmt.Sprintf(
			"search/repositories?q=stars:>1000&sort:stars&per_page=%s&page=%s",
			strconv.Itoa(ghPageSize),
			strconv.Itoa(page),
		)

		res, err := github.PerformApiCall(url, ghToken, nil)

		if err != nil {
			return err
		}

		var repos Repos
		err = json.NewDecoder(res).Decode(&repos)

		if err != nil {
			return err
		}

		for _, repo := range repos.Repo {
			urls = append(urls, repo.Url)
			_, _ = fmt.Fprintf(
				writer,
				"\u001B[37m[INIT]\u001B[0m URLs not found, retrieving top %d from GitHub [%d/%d]\n",
				ghPageSize*ghPages, i, ghPageSize*ghPages,
			)

			time.Sleep(time.Millisecond * 25)
			i++
		}
	}

	_, _ = fmt.Fprintf(
		writer.Bypass(),
		"\u001B[37m[INIT]\u001B[0m URLs not found, retrieving top %d from GitHub \u001B[32m✓\u001B[0m\n",
		ghPageSize*ghPages,
	)

	writer.Stop()

	err = writeToFile(urls)

	if err != nil {
		return err
	}

	return nil
}
