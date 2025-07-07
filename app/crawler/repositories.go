package crawler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gosuri/uilive"
	"gopkg.in/ini.v1"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"
)

type Repo struct {
	Url string `json:"html_url"`
}

type Repos struct {
	Repo []Repo `json:"items"`
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
func GetTopRepositories(config *ini.Section) error {
	var urls []string
	client := &http.Client{}

	ghToken := config.Key("TOKEN").String()
	ghPageSize, err := config.Key("SIZE").Int()
	ghPages, err := config.Key("PAGES").Int()

	if err != nil {
		return err
	}

	writer := uilive.New()
	writer.Start()

	i := 1

	ghQuery := "https://api.github.com/search/repositories?q=stars:>1000&sort:stars&per_page=" + strconv.Itoa(ghPageSize)

	for page := range ghPages {
		req, err := http.NewRequest("GET", ghQuery+"&page="+strconv.Itoa(page), nil)

		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+ghToken)
		res, err := client.Do(req)

		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			return errors.New("status code: " + res.Status)
		}

		var repos Repos
		err = json.NewDecoder(res.Body).Decode(&repos)

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
