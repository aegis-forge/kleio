package git

import (
	"app/pkg/git/model"
	"fmt"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"maps"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strings"
)

// buildComponent appends a [Component] struct to the slice of components
func buildComponent(cType string, sep string, component string, components map[string]*model.Component) {
	componentSplit := strings.Split(component, sep)
	componentName := componentSplit[0]
	componentVersion := ""

	if len(componentSplit) == 2 {
		componentVersion = componentSplit[1]
	}

	if _, ok := components[componentName]; !ok {
		componentStruct := model.Component{}
		versionStruct := model.Version{}

		versionStruct.Init(componentVersion)
		componentStruct.Init(componentName, cType, []*model.Version{&versionStruct})

		components[componentName] = &componentStruct
	} else {
		for _, version := range components[componentName].GetHistory() {
			if version.GetVersionString() == componentVersion {
				version.AddUses(1)
			}
		}
	}
}

// extractComponents returns a slice of [Component] structs extracted from a workflow
func extractComponents(content string) ([]*model.Component, error) {
	var yamlStruct yaml.Node

	components := make(map[string]*model.Component)
	actionDockerPath, err := yamlpath.NewPath("$..uses")

	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal([]byte(content), &yamlStruct); err != nil {
		return nil, nil
	}

	actionOut, err := actionDockerPath.Find(&yamlStruct)

	if err != nil {
		return []*model.Component{}, nil
	}

	for _, component := range actionOut {
		if strings.Contains(component.Value, "docker://") {
			buildComponent("docker", ":", strings.TrimPrefix(component.Value, "docker://"), components)
		} else if strings.Contains(component.Value, ".yml") || strings.Contains(component.Value, ".yaml") {
			buildComponent("workflow", "@", component.Value, components)
		} else {
			buildComponent("action", "@", component.Value, components)
		}
	}

	return slices.Collect(maps.Values(components)), nil
}

// ExtractWorkflows returns a slice of [File] structs with their histories given the URL of a GitHub repository
func ExtractWorkflows(url string) ([]model.File, error) {
	var workflows []model.File

	_, filename, _, _ := runtime.Caller(0)

	urlSplit := strings.Split(url, "/")
	repoName := urlSplit[len(urlSplit)-1]
	reposPath := path.Join(path.Dir(filename), "../../tmp/repos")
	repoPath := path.Join(reposPath, repoName)

	err := os.MkdirAll(reposPath, 0755)

	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(path.Join(repoPath)); err != nil {
		if os.IsNotExist(err) {
			fmt.Print("Repo \033[31m" + repoName + "\033[0m not in filesystem, cloning (might take some time)")

			cmd := exec.Command("git", "clone", url, repoPath)
			err = cmd.Run()

			if err != nil {
				fmt.Println(" \u001B[31m𐄂\u001B[0m")
				return nil, err
			}

			fmt.Println(" \u001B[32m✓\u001B[0m")
		}
	}

	fmt.Println("Extracting workflows from \033[31m" + repoName + "\033[0m and reading histories")

	_, err = os.Stat(path.Join(repoPath, ".github/workflows"))

	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(path.Join(repoPath, ".github/workflows"))

	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if history, err := getFileHistory(repoPath, ".github/workflows/"+f.Name()); err == nil {
			workflows = append(workflows, history)
		} else {
			return nil, err
		}
	}

	//err = os.RemoveAll(repoPath)
	//
	//if err != nil {
	//	return nil, err
	//}

	return workflows, nil
}
