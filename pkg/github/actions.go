package github

import (
	"app/cmd/database"
	"app/cmd/helpers"
	"app/pkg/git"
	"app/pkg/git/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/aegis-forge/cage"
	"github.com/gosuri/uilive"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	gocvss20 "github.com/pandatix/go-cvss/20"
	gocvss31 "github.com/pandatix/go-cvss/31"
	gocvss40 "github.com/pandatix/go-cvss/40"
	"gopkg.in/ini.v1"
)

// A Release containing its tag as returned by the GitHub API
type release struct {
	Tag string `json:"tag_name"`
}

// A Tag with its SHA code as returned by the GitHub API
type tag struct {
	Object struct {
		Sha string `json:"sha"`
	} `json:"object"`
}

// A Lock is the structure of a package-lock.json file
type lock struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

type yarnLock struct {
	Data struct {
		Trees []struct {
			Name string `json:"name"`
		} `json:"trees"`
	} `json:"data"`
}

// getTags returns all the version tags present in an Action's repository
func getTags(action string, bearer string) ([]string, error) {
	var releases []string

	writer := uilive.New()
	writer.Start()

	i := 1

	for {
		uri := fmt.Sprintf("repos/%s/releases?page=%d&per_page=100", action, i)
		res, _, err := PerformApiCall(uri, bearer, nil)

		if err != nil {
			return nil, err
		}

		var releasesRaw []release
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

	for index, tagz := range tags {
		// Get tag SHA
		uri := fmt.Sprintf("repos/%s/git/ref/tags/%s", action, tagz)
		res, _, err := PerformApiCall(uri, bearer, nil)

		if err != nil {
			return nil, err
		}

		var tagRaw tag
		err = json.NewDecoder(res).Decode(&tagRaw)

		if err != nil {
			return nil, err
		}

		// Get commit SHA
		uri = fmt.Sprintf("repos/%s/git/tags/%s", action, tagRaw.Object.Sha)
		res, status, err := PerformApiCall(uri, bearer, nil)

		if err != nil || status != 200 {
			hashes[tagz] = tagRaw.Object.Sha
		} else {
			var commitRaw tag
			err = json.NewDecoder(res).Decode(&commitRaw)

			if err != nil {
				return nil, err
			}

			hashes[tagz] = commitRaw.Object.Sha
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

// getActionVersions saves all the commits, versions, components, and vendors retrieved in the Neo4j database. It returns true if it saved at least one release
func getActionVersions(action string, hashes map[string]string, repoPath string, driver neo4j.DriverWithContext, ctx context.Context) bool {
	versionToCommitMap := map[string][]string{}
	actionSplit := strings.Split(action, "/")

	for tag, hash := range hashes {
		for _, command := range []string{"tag", "branch"} {
			cmd := exec.Command("git", "-C", repoPath, command, "--contains", hash)
			out, err := cmd.Output()

			if err != nil {
				break
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

	writer := uilive.New()
	writer.Start()

	i := 0

	checkedDependencies := []string{}

	for version, hashes := range versionToCommitMap {
		i++

		subtype := "composite"

		if _, err := os.Stat(fmt.Sprintf("%s/package-lock.json", repoPath)); err == nil {
			subtype = "js+lock"
		} else if _, err := os.Stat(fmt.Sprintf("%s/yarn.lock", repoPath)); err == nil {
			subtype = "js+lock"
		} else if _, err := os.Stat(fmt.Sprintf("%s/package.json", repoPath)); err == nil {
			subtype = "js"
		}

		_, _ = fmt.Fprintf(
			writer,
			"[ACTIONS] Saving releases [%d/%d]\n",
			i, len(versionToCommitMap),
		)

		database.ExecuteQueryNeo(
			`MERGE (v:Vendor {name: $vendor})
			MERGE (c:Component {full_name: $component, name: $action, type: "action", subtype: $subtype, provider: "github"})
			MERGE (ve:Version {full_name: $version, name: $semver})
			MERGE (v)-[:PUBLISHES]->(c)
			MERGE (c)-[:DEPLOYS]->(ve)`,
			map[string]any{
				"vendor":    actionSplit[0],
				"component": action,
				"action":    actionSplit[1],
				"version":   action + "/" + version,
				"subtype":   subtype,
				"semver":    version,
			},
			driver, ctx,
		)

		for _, hash := range hashes {
			if res := database.ExecuteQueryWithRetNeo(
				`MATCH (c:Commit {full_name: $commit})
				WITH COUNT(c) > 0 as node_v
				RETURN node_v`,
				map[string]any{
					"commit": action + "/" + hash,
				},
				driver, ctx,
			); res[0].Values[0] == true {
				database.ExecuteQueryNeo(
					`MATCH (v:Version {full_name: $version})
					MATCH (c:Commit {full_name: $commit})
					MERGE (v)-[:PUSHES]->(c)`,
					map[string]any{
						"version": action + "/" + version,
						"commit":  action + "/" + hash,
					},
					driver, ctx,
				)
				
				continue
			}

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

			if vulns, err := getActionVulnerabilities(actionSplit[0], actionSplit[1], version, date); err == nil {
				for _, vuln := range vulns {
					ratio := math.Pow(10, 2)
					cvss := math.Round(float64(vuln.Cvss)*ratio) / ratio

					database.ExecuteQueryNeo(
						`MATCH (c:Commit {full_name: $commit})
						MERGE (v:Vulnerability {id: $id, cve: $cve, cwes: $cwes, cvss: $cvss, published: $published})
						MERGE (c)-[:VULNERABLE_TO]->(v)`,
						map[string]any{
							"commit":    action + "/" + hash,
							"id":        vuln.Id,
							"cve":       vuln.Cve,
							"cwes":      vuln.Cwes,
							"cvss":      cvss,
							"published": vuln.Published,
						},
						driver, ctx,
					)
				}
			}

			var lock []byte
			var lockType string

			if _, err := os.Stat(fmt.Sprintf("%s/package-lock.json", repoPath)); err == nil {
				lockType = "npm"
			} else if _, err := os.Stat(fmt.Sprintf("%s/yarn.lock", repoPath)); err == nil {
				lockType = "yarn"
			}

			cmd = exec.Command("git", "-C", repoPath, "show", fmt.Sprintf("%s:package.json", hash))
			pkgJson, err := cmd.Output()

			switch lockType {
			case "npm":
				cmd = exec.Command("git", "-C", repoPath, "show", fmt.Sprintf("%s:package-lock.json", hash))
				lock, err = cmd.Output()
			case "yarn":
				cmd = exec.Command("git", "-C", repoPath, "show", fmt.Sprintf("%s:yarn.lock", hash))
				lock, err = cmd.Output()
			}

			if err == nil {
				getTransitiveDependenciesAndVulnerabilities(pkgJson, lock, lockType, repoPath, action+"/"+hash, driver, ctx, &checkedDependencies)
			}
		}
	}

	_, _ = fmt.Fprintf(
		writer.Bypass(),
		"[ACTIONS] Saving releases \u001B[32m✓\u001B[0m\n",
	)

	time.Sleep(time.Millisecond * 25)
	writer.Stop()

	if len(versionToCommitMap) == 0 {
		return false
	}

	return true
}

func getPackages(lockFile []byte, repoPath, lockType string) (map[string]string, error) {
	dependencies := map[string]string{}

	switch lockType {
	case "npm":
		var npmLock lock

		if err := json.Unmarshal(lockFile, &npmLock); err != nil {
			return nil, err
		}

		it := npmLock.Packages

		if it == nil {
			it = npmLock.Dependencies
		}

		for name, version := range it {
			dependencies[name] = version.Version
		}
	case "yarn":
		cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("cd %s && yarn info --name-only -R --json", repoPath))
		out, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Print(string(out))
			return nil, err
		}

		lines := strings.Split(string(out), "\n")

		if len(lines) <= 2 || strings.HasPrefix(lines[0], "{\\\"") {
			// Try with `yarn list` if `yarn info` fails to produce dependencies
			lines = []string{}

			cmdF := exec.Command("/bin/bash", "-c", fmt.Sprintf("cd %s && yarn list --ignore-scripts --depth=0 --json", repoPath))
			out, err = cmdF.CombinedOutput()

			if err != nil {
				fmt.Print(string(out))
				return nil, err
			}

			var yarnLock yarnLock
			err = json.Unmarshal(out, &yarnLock)

			if err != nil || len(yarnLock.Data.Trees) == 0 {
				return nil, err
			}

			for _, line := range yarnLock.Data.Trees {
				lines = append(lines, line.Name)
			}

			if len(lines) == 0 {
				return nil, nil
			}
		}

		for _, dep := range lines {
			if dep == "" {
				continue
			}

			dep = strings.ReplaceAll(dep, "\"", "")

			re := regexp.MustCompile(`^(\S+)@\D*(.+)$`)
			matches := re.FindStringSubmatch(dep)

			if len(matches) < 2 {
				continue
			}

			if strings.ContainsRune(matches[1], ':') || matches[2] == "." {
				continue
			}

			dependencies[matches[1]] = matches[2]
		}
	}

	return dependencies, nil
}

func getTransitiveDependenciesAndVulnerabilities(pkg, lock []byte, lockType, repoPath, commit string,
	driver neo4j.DriverWithContext, ctx context.Context, checkedDependencies *[]string) {

	type PackageJson struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		OptDependencies map[string]string `json:"optDependencies"`
	}

	var directDeps []string
	var devDeps []string
	var optDeps []string

	var pkgJson PackageJson
	err := json.Unmarshal(pkg, &pkgJson)

	dependencies, err := getPackages(lock, repoPath, lockType)

	if err != nil {
		return
	}

	writer := uilive.New()
	writer.Start()

	i := 0

	for i, deps := range []map[string]string{pkgJson.Dependencies, pkgJson.DevDependencies, pkgJson.OptDependencies} {
		for dep := range deps {
			switch i {
			case 0:
				directDeps = append(directDeps, dep)
			case 1:
				devDeps = append(devDeps, dep)
			case 2:
				optDeps = append(optDeps, dep)
			}
		}
	}

	for name, version := range dependencies {
		if name == "" {
			continue
		}

		i++

		_, _ = fmt.Fprintf(
			writer,
			"[ACTIONS] Saving transitive dependencies [%d/%d]\n",
			i, len(dependencies),
		)

		nodeModules := strings.Split(name, "node_modules/")
		cleanedName := nodeModules[len(nodeModules)-1]

		moduleType := "indirect"

		if slices.Contains(directDeps, cleanedName) {
			moduleType = "direct"
		} else if slices.Contains(devDeps, cleanedName) {
			moduleType = "direct_dev"
		} else if slices.Contains(optDeps, cleanedName) {
			moduleType = "direct_opt"
		}

		if slices.Contains(*checkedDependencies, cleanedName+"/"+version) {
			database.ExecuteQueryNeo(
				`MATCH (c:Commit {full_name: $commit})
				MATCH (v:Version {full_name: $version})
				MERGE (c)-[:USES {type: $dep}]->(v)`,
				map[string]any{
					"commit":  commit,
					"version": cleanedName + "/" + version,
					"dep":     moduleType,
				},
				driver, ctx,
			)
		} else {
			if res := database.ExecuteQueryWithRetNeo(
				`MATCH (v:Version {full_name: $version})
				WITH COUNT(v) > 0 as node_v
				RETURN node_v`,
				map[string]any{
					"version": cleanedName + "/" + version,
				},
				driver, ctx,
			); res[0].Values[0] == true {
				database.ExecuteQueryNeo(
					`MATCH (c:Commit {full_name: $commit})
					MATCH (v:Version {full_name: $version})
					MERGE (c)-[:USES {type: $dep}]->(v)`,
					map[string]any{
						"commit":  commit,
						"version": cleanedName + "/" + version,
						"dep":     moduleType,
					},
					driver, ctx,
				)

				continue
			}

			database.ExecuteQueryNeo(
				`MATCH (c:Commit {full_name: $commit})
				MERGE (co:Component {full_name: $component, name: $cname, type: "package", provider: "npm"})
				MERGE (v:Version {full_name: $version, name: $vname})
				MERGE (co)-[:DEPLOYS]->(v)
				MERGE (c)-[:USES {type: $dep}]->(v)`,
				map[string]any{
					"commit":    commit,
					"component": cleanedName,
					"cname":     cleanedName,
					"version":   cleanedName + "/" + version,
					"vname":     version,
					"dep":       moduleType,
				},
				driver, ctx,
			)

			osv_url := "https://api.osv.dev/v1/query"
			body := map[string]any{
				"package": map[string]string{
					"purl": fmt.Sprintf("pkg:npm/%s@%s", cleanedName, version),
				},
			}

			jsonBody, err := json.Marshal(body)

			if err != nil {
				continue
			}

			res, err := http.Post(osv_url, "application/json", bytes.NewBuffer(jsonBody))

			if err != nil {
				continue
			}

			defer res.Body.Close()

			type osvVuln struct {
				Vulns []struct {
					Id        string   `json:"id"`
					Aliases   []string `json:"aliases"`
					Published string   `json:"published"`
					Advisory  struct {
						CWEs []string `json:"cwe_ids"`
					} `json:"database_specific"`
					Severity []struct {
						Type  string `json:"type"`
						Score string `json:"score"`
					} `json:"severity"`
				} `json:"vulns"`
			}

			var ovsVulns osvVuln

			json.NewDecoder(res.Body).Decode(&ovsVulns)

			for _, vuln := range ovsVulns.Vulns {
				cve := ""
				cvss := 0.0

				if len(vuln.Aliases) > 0 {
					cve = vuln.Aliases[0]
				}

				if len(vuln.Severity) > 0 {
					vector := vuln.Severity[0].Score

					switch vuln.Severity[0].Type {
					case "CVSS_V3":
						cvssObj, err := gocvss31.ParseVector(vector)

						if err != nil {
							cvss = 0.0
						} else {
							cvss = cvssObj.BaseScore()
						}
					case "CVSS_V4":
						cvssObj, err := gocvss40.ParseVector(vector)

						if err != nil {
							cvss = 0.0
						} else {
							cvss = cvssObj.Score()
						}
					default:
						cvssObj, err := gocvss20.ParseVector(vector)

						if err != nil {
							cvss = 0.0
						} else {
							cvss = cvssObj.BaseScore()
						}
					}
				}

				database.ExecuteQueryNeo(
					`MATCH (c:Version {full_name: $version})
					MERGE (v:Vulnerability {id: $id, cve: $cve, cwes: $cwes, cvss: $cvss, published: $published})
					MERGE (c)-[:VULNERABLE_TO]->(v)`,
					map[string]any{
						"version":   cleanedName + "/" + version,
						"id":        vuln.Id,
						"cve":       cve,
						"cwes":      vuln.Advisory.CWEs,
						"cvss":      cvss,
						"published": vuln.Published,
					},
					driver, ctx,
				)
			}

			*checkedDependencies = append(*checkedDependencies, cleanedName+"/"+version)
		}
	}

	_, _ = fmt.Fprintf(
		writer.Bypass(),
		"[ACTIONS] Saving transitive dependencies \u001B[32m✓\u001B[0m\n",
	)
}

func getActionVulnerabilities(vendor, action, version string, time time.Time) ([]cage.Vulnerability, error) {
	semver, err := cage.NewSemver(version)

	if err != nil {
		return nil, err
	}

	pkg, err := cage.NewPackage(vendor, action, time, semver)

	if err != nil {
		return nil, err
	}

	token, err := helpers.ReadConfig()

	if err != nil {
		return nil, err
	}

	gh := cage.Github{}
	gh.SetToken(token.Section("GITHUB").Key("TOKEN_VULNS").String())

	return pkg.IsVulnerable([]cage.Source{gh})
}

// GetActionsCommits retrieves all the versions and commits of all the Actions present in the repositories' workflows
func GetActionsCommits(repo model.Repository, config *ini.Section, driver neo4j.DriverWithContext, ctx context.Context) {
	bearer := config.Key("TOKEN").String()
	errorActions := []string{}

	for _, workflow := range repo.GetFiles() {
		for _, commit := range workflow.GetHistory() {
			for _, component := range commit.GetComponents() {
				if component.GetCategory() != "action" {
					continue
				}

				action := component.GetName()

				if slices.Contains(errorActions, action) {
					continue
				}

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
					errorActions = append(errorActions, action)
					continue
				}

				// Extract the commit hashes from the release tags
				hashes, err := getCommitHashes(action, tags, bearer)

				if err != nil {
					errorActions = append(errorActions, action)
					continue
				}

				// Pull Action repo
				repoPath, err := pullActionRepo(action)

				if err != nil {
					errorActions = append(errorActions, action)
					git.DeleteRepo(repoPath)
					continue
				}

				// Extract and save the versions of the Action
				if found := getActionVersions(action, hashes, repoPath, driver, ctx); !found {
					errorActions = append(errorActions, action)
				}

				// Delete Action repository
				git.DeleteRepo(repoPath)
			}
		}
	}
}
