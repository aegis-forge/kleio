package crawler

import (
	"app/app/database"
	"app/pkg/git"
	"app/pkg/git/model"
	"bufio"
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"os"
	"strings"
)

func ExtractWorkflows(neoDriver neo4j.DriverWithContext, neoCtx context.Context) {
	f, err := os.Open("./out/repositories.txt")

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		url := scanner.Text()
		// Extract all workflows from repository
		workflows, err := git.ExtractWorkflows(url)

		if err != nil {
			panic(err)
		}

		var repo model.Repository
		repo.Init(strings.TrimPrefix(url, "https://github.com/"), url, workflows)

		// Save repo to neo4j
		database.SendToNeo(repo, neoDriver, neoCtx)
	}
}
