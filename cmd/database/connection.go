package database

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ConnectToNeo is used to connect to the Neo4j instance
func ConnectToNeo() (neo4j.DriverWithContext, context.Context, error) {
	ctx := context.Background()

	dbUri := os.Getenv("NEO_URI")
	dbUser := os.Getenv("NEO_USER")
	dbPassword := os.Getenv("NEO_PASS")

	fmt.Printf("\u001B[37m[INIT]\u001B[0m Connecting to Neo4j (\u001B[34m%s\u001B[0m)", dbUri)

	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))

	if err != nil {
		return nil, nil, err
	}

	if err = driver.VerifyConnectivity(ctx); err != nil {
		fmt.Println(" \u001B[31m𐄂\u001B[0m")

		return nil, nil, err
	}

	fmt.Println(" \u001B[32m✓\u001B[0m")

	return driver, ctx, err
}

func ConnectionToMongo() (mongo.Database, error) {
	dbUri := os.Getenv("MONGO_URI")
	dbUsername := os.Getenv("MONGO_USER")
	dbPassword := os.Getenv("MONGO_PASS")

	fmt.Printf("\u001B[37m[INIT]\u001B[0m Connecting to MongoDB (\u001B[34m%s\u001B[0m)", dbUri)

	client, err := mongo.Connect(
		options.Client().ApplyURI(dbUri).SetAuth(
			options.Credential{
				Username: dbUsername,
				Password: dbPassword,
			},
		),
	)

	if err != nil {
		return mongo.Database{}, err
	}

	fmt.Println(" \u001B[32m✓\u001B[0m")

	return *client.Database("kleio"), err
}
