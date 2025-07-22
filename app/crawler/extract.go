package crawler

import (
	"app/app/database"
	"app/app/helpers"
	"app/pkg/git"
	"app/pkg/git/model"
	"app/pkg/github"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ExtractWorkflows extracts the workflows from the Repository
func ExtractWorkflows(neoDriver neo4j.DriverWithContext, neoCtx context.Context) {
	// Read config
	config, err := helpers.ReadConfig()

	if err != nil {
		panic(err)
	}
	
	f, err := os.Open("../repositories.txt")

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		section, err := config.GetSection("GENERAL")
		
		if err != nil {
			panic(err)
		}
		
		url := scanner.Text()
		// Extract all workflows from repository
		workflows, err := git.ExtractWorkflows(url, section)

		if err != nil {
			urlSplit := strings.Split(url, "/")
			git.DeleteRepo(urlSplit[len(urlSplit)-1])

			if strings.Contains(err.Error(), "no such file") {
				fmt.Println(" \u001B[31m𐄂\u001B[0m \u001B[34m(No workflows found)\u001B[0m")
				fmt.Println()
				continue
			}

			panic(err)
		}

		var repo model.Repository
		repo.Init(strings.TrimPrefix(url, "https://github.com/"), url, workflows)

		// Read config
		config, err := helpers.ReadConfig()

		if err != nil {
			panic(err)
		}

		// Retrieve Actions Commits
		github.GetActionsCommits(repo, config.Section("GITHUB"), neoDriver, neoCtx)

		// Save repo to neo4j
		database.SendToNeo(repo, neoDriver, neoCtx)
	}
}
