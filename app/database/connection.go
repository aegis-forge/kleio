package database

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gopkg.in/ini.v1"
)

// ConnectToNeo is used to connect to the Neo4j instance
func ConnectToNeo(config *ini.Section) (neo4j.DriverWithContext, context.Context, error) {
	ctx := context.Background()

	dbUri := config.Key("URI").String()
	dbUser := config.Key("USERNAME").String()
	dbPassword := config.Key("PASSWORD").String()

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
