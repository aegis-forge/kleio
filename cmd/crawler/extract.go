package crawler

import (
	"kleio/cmd/database"
	"kleio/pkg/git"
	"kleio/pkg/git/model"
	"kleio/pkg/github"
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
	f, err := os.Open("./repositories.txt")

	if err != nil {
		panic(err)
	}

	repositories := []string{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		repositories = append(repositories, scanner.Text())
	}

	// progressBar := progress.NewPBar()
	// progressBar.Total = uint16(len(repositories))

	for _, url := range repositories {
		// progressBar.RenderPBar(index)

		// Extract all workflows from repository
		workflows, err := git.ExtractWorkflows(url)

		if err != nil {
			urlSplit := strings.Split(url, "/")
			git.DeleteRepo(urlSplit[len(urlSplit)-1])

			if strings.Contains(err.Error(), "no such file") {
				fmt.Println(" \u001B[31m𐄂\u001B[0m \u001B[34m(No workflows found)\u001B[0m")
				fmt.Println()
			}

			continue
		}

		var repo model.Repository
		repo.Init(strings.TrimPrefix(url, "https://github.com/"), url, workflows)

		// Retrieve Actions Commits
		github.GetActionsCommits(repo, neoDriver, neoCtx)

		// Save repo to databases
		database.SendToDB(repo, neoDriver, neoCtx, mongoClient)
	}

	// progressBar.CleanUp()
}
