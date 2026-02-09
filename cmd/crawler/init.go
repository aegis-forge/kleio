package crawler

import (
	"app/cmd/database"
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Initialize initializes the configuration file, db, and repositories' URLs
func Initialize() (neo4j.DriverWithContext, context.Context, mongo.Database) {
	fmt.Println("\u001B[37m[INIT]\u001B[0m \u001B[33mStarting initialization step")

	// Connect to DBs
	neoDriver, neoCtx, err := database.ConnectToNeo()

	if err != nil {
		panic(err)
	}

	// Connect to MongoDB
	mongoClient, err := database.ConnectionToMongo()

	if err != nil {
		panic(err)
	}

	// Retrieve top N URLs from GitHub (if file does not exist)
	reposPath := "./repositories.txt"

	if _, err = os.Stat(reposPath); os.IsNotExist(err) {
		if err = getTopRepositories(); err != nil {
			panic(err)
		}
	}

	fmt.Print("\u001B[37m[INIT]\u001B[0m \u001B[32mInitialization complete\u001B[0m\n\n")

	return neoDriver, neoCtx, mongoClient
}
