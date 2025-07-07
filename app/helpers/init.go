package helpers

import (
	"app/app/crawler"
	"app/app/database"
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"os"
	"path"
	"runtime"
)

func Initialize() (neo4j.DriverWithContext, context.Context) {
	fmt.Println("\u001B[37m[INIT]\u001B[0m \u001B[33mStarting initialization step")

	// Read config file
	config, err := ReadConfig()

	if err != nil {
		panic(err)
	}

	// Connect to DBs
	neoDriver, neoCtx, err := database.ConnectToNeo(config.Section("NEO4J"))

	if err != nil {
		panic(err)
	}

	// Retrieve top N URLs from GitHub (if file does not exist)
	_, filename, _, _ := runtime.Caller(0)
	reposPath := path.Join(path.Dir(filename), "../../out/repositories.txt")

	if _, err = os.Stat(reposPath); os.IsNotExist(err) {
		if err = crawler.GetTopRepositories(config.Section("GITHUB")); err != nil {
			panic(err)
		}
	}

	fmt.Print("\u001B[37m[INIT]\u001B[0m \u001B[32mInitialization complete\u001B[0m\n\n")

	return neoDriver, neoCtx
}
