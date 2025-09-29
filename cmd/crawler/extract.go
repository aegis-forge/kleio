package crawler

import (
	"app/cmd/database"
	"app/cmd/helpers"
	"app/pkg/git"
	"app/pkg/git/model"
	"app/pkg/github"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ExtractWorkflows extracts the workflows from the Repository
func ExtractWorkflows(neoDriver neo4j.DriverWithContext, neoCtx context.Context, mongoClient mongo.Database) {
	// Read config
	config, err := helpers.ReadConfig()

	if err != nil {
		panic(err)
	}

	f, err := os.Open("./repositories.txt")

	if err != nil {
		panic(err)
	}

	repositories := []string{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		repositories = append(repositories, scanner.Text())
	}

	section, err := config.GetSection("GENERAL")

	// progressBar := progress.NewPBar()
	// progressBar.Total = uint16(len(repositories))

	if err != nil {
		panic(err)
	}

	for _, url := range repositories {
		// progressBar.RenderPBar(index)

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

		// Save repo to databases
		database.SendToDB(repo, neoDriver, neoCtx, mongoClient)
	}

	// progressBar.CleanUp()
}
