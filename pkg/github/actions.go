package github

import (
	"app/app/database"
	"app/pkg/git"
	"app/pkg/git/model"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gopkg.in/ini.v1"
)

// A Release containing its tag as returned by the GitHub API
type Release struct {
	Tag string `json:"tag_name"`
}

// A Tag with its SHA code as returned by the GitHub API
type Tag struct {
	Object struct {
		Sha string `json:"sha"`
	} `json:"object"`
}

// getTags returns all the version tags present in an Action's repository
func getTags(action string, bearer string) ([]string, error) {
	var releases []string

	writer := uilive.New()
	writer.Start()

	i := 1

	for {
		uri := fmt.Sprintf("repos/%s/releases?page=%d&per_page=100", action, i)
		res, err := PerformApiCall(uri, bearer, nil)

		if err != nil {
			return nil, err
		}

		var releasesRaw []Release
		err = json.NewDecoder(res).Decode(&releasesRaw)

		if err != nil {
			return nil, err
		}

		if len(releasesRaw) == 0 {
			break
		}

		for index, release := range releasesRaw {
			releases = append(releases, release.Tag)

			_, _ = fmt.Fprintf(
				writer,
				"[ACTIONS] Extracting releases from \033[31m%s\033[0m Action [%d/?]\n",
				action, index+1,
			)

			time.Sleep(time.Millisecond * 25)
		}

		i++
	}

	_, _ = fmt.Fprintf(
		writer.Bypass(),
		"[ACTIONS] Extracting releases from \033[31m%s\033[0m Action \u001B[32m✓\u001B[0m\n",
		action,
	)

	writer.Stop()

	return releases, nil
}

// getCommitHashes returns all the commit hashes connected to the version tags of an Action
func getCommitHashes(action string, tags []string, bearer string) (map[string]string, error) {
	hashes := map[string]string{}

	writer := uilive.New()
	writer.Start()

	for index, tag := range tags {
		// Get tag SHA
		uri := fmt.Sprintf("repos/%s/git/ref/tags/%s", action, tag)
		res, err := PerformApiCall(uri, bearer, nil)

		if err != nil {
			return nil, err
		}

		var tagRaw Tag
		err = json.NewDecoder(res).Decode(&tagRaw)

		if err != nil {
			return nil, err
		}

		// Get commit SHA
		uri = fmt.Sprintf("repos/%s/git/tags/%s", action, tagRaw.Object.Sha)
		res, err = PerformApiCall(uri, bearer, nil)

		if err != nil {
			hashes[tag] = tagRaw.Object.Sha
		} else {
			var commitRaw Tag
			err = json.NewDecoder(res).Decode(&commitRaw)

			if err != nil {
				return nil, err
			}

			hashes[tag] = commitRaw.Object.Sha
		}

		_, _ = fmt.Fprintf(
			writer,
			"[ACTIONS] Extracting tags [%d/%d]\n",
			index+1, len(tags),
		)
	}

	_, _ = fmt.Fprintf(
		writer.Bypass(),
		"[ACTIONS] Extracting tags \u001B[32m✓\u001B[0m\n",
	)

	time.Sleep(time.Millisecond * 25)
	writer.Stop()

	return hashes, nil
}

// pullActionRepo clones an Action repo from GitHub and returns its absolute path
func pullActionRepo(action string) (string, error) {
	_, filename, _, _ := runtime.Caller(0)

	reposPath := path.Join(path.Dir(filename), "../../tmp/actions")
	repoName := strings.Split(action, "/")[1]
	repoPath := path.Join(reposPath, repoName)
	repoUrl := fmt.Sprintf("https://github.com/%s", action)

	err := os.MkdirAll(reposPath, 0755)

	if err != nil {
		return "", err
	}

	if _, err = os.Stat(path.Join(repoPath)); err != nil {
		if os.IsNotExist(err) {
			fmt.Print("[ACTIONS] Action not in filesystem, cloning (might take some time)")

			cmd := exec.Command("git", "clone", repoUrl, repoPath)
			err = cmd.Run()

			if err != nil {
				fmt.Println(" \u001B[31m𐄂\u001B[0m")

				return "", err
			}

			fmt.Println(" \u001B[32m✓\u001B[0m")
		}
	}

	return repoPath, nil
}

// getActionVersions saves all the commits, versions, components, and vendors retrieved in the Neo4j database
func getActionVersions(action string, hashes map[string]string, repoPath string, driver neo4j.DriverWithContext, ctx context.Context) {
	versionToCommitMap := map[string][]string{}
	actionSplit := strings.Split(action, "/")

	for tag, hash := range hashes {
		for _, command := range []string{"tag", "branch"} {
			cmd := exec.Command("git", "-C", repoPath, command, "--contains", hash)
			out, err := cmd.Output()

			if err != nil {
				panic(err)
			}

			for _, version := range strings.Split(string(out), "\n") {
				if version == "" {
					break
				}

				version = strings.TrimPrefix(version, "* ")

				versionToCommitMap[version] = append(versionToCommitMap[version], hash)

				if version == tag {
					break
				}
			}
		}
	}

	fmt.Printf("\u001B[37m[NEO4J]\u001B[0m Saving releases")

	for version, hashes := range versionToCommitMap {
		database.ExecuteQueryNeo(
			`MERGE (v:Vendor {name: $vendor})
			MERGE (c:Component {full_name: $component, name: $action, type: "action"})
			MERGE (ve:Version {full_name: $version, name: $semver})
			MERGE (v)-[:PUBLISHES]->(c)
			MERGE (c)-[:DEPLOYS]->(ve)`,
			map[string]any{
				"vendor":    actionSplit[0],
				"component": action,
				"action":    actionSplit[1],
				"version":   action + "/" + version,
				"semver":    version,
			},
			driver, ctx,
		)

		for _, hash := range hashes {
			cmd := exec.Command("git", "-C", repoPath, "show", "-s", "--format=%ci", hash)
			out, err := cmd.Output()

			if err != nil {
				panic(err)
			}

			if strings.HasPrefix(string(out), "fatal:") {
				continue
			}

			dateRaw := string(out)
			outSplit := strings.Split(string(out), "\n")

			if len(outSplit) > 2 {
				dateRaw = outSplit[len(outSplit)-2]
			}

			date, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(dateRaw))

			if err != nil {
				fmt.Println(hash)
				fmt.Println(string(out))
				panic(err)
			}

			database.ExecuteQueryNeo(
				`MATCH (v:Version {full_name: $version})
				MERGE (c:Commit {full_name: $commit, name: $hash, date: $date})
				MERGE (v)-[:PUSHES]->(c)`,
				map[string]any{
					"version": action + "/" + version,
					"commit":  action + "/" + hash,
					"hash":    hash,
					"date":    neo4j.LocalDateTimeOf(date),
				},
				driver, ctx,
			)
		}
	}

	fmt.Print(" \u001B[32m✓\u001B[0m\n\n")
}

// GetActionsCommits retrieves all the versions and commits of all the Actions present in the repositories' workflows
func GetActionsCommits(repo model.Repository, config *ini.Section, driver neo4j.DriverWithContext, ctx context.Context) {
	bearer := config.Key("TOKEN").String()

	for _, workflow := range repo.GetFiles() {
		for _, commit := range workflow.GetHistory() {
			for _, component := range commit.GetComponents() {
				if component.GetCategory() != "action" {
					continue
				}

				action := component.GetName()

				if len(strings.Split(action, "/")) > 2 {
					actionSplit := strings.Split(action, "/")
					action = strings.Join(actionSplit[:len(actionSplit)-1], "/")
				}

				// Check if Action exists in database
				if res := database.ExecuteQueryWithRetNeo(
					`MATCH (c:Component {full_name: $component})
					WITH COUNT(c) > 0 as node_c
					RETURN node_c`,
					map[string]any{
						"component": action,
					},
					driver, ctx,
				); res[0].Values[0] == true || strings.HasPrefix(action, "./") {
					continue
				}

				// Extract the release tags
				tags, err := getTags(action, bearer)

				if err != nil {
					panic(err)
				}

				// Extract the commit hashes from the release tags
				hashes, err := getCommitHashes(action, tags, bearer)

				if err != nil {
					panic(err)
				}

				// Pull Action repo
				repoPath, err := pullActionRepo(action)

				if err != nil {
					panic(err)
				}

				// Extract and save the versions of the Action
				getActionVersions(action, hashes, repoPath, driver, ctx)

				// Delete Action repository
				git.DeleteRepo(repoPath)
			}
		}
	}
}
