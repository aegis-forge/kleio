package database

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gopkg.in/ini.v1"
)

func ConnectToNeo(config *ini.File) (neo4j.DriverWithContext, context.Context, error) {
	ctx := context.Background()

	dbUri := config.Section("NEO4J").Key("URI").String()
	dbUser := config.Section("NEO4J").Key("USERNAME").String()
	dbPassword := config.Section("NEO4J").Key("PASSWORD").String()

	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))

	defer func(driver neo4j.DriverWithContext, ctx context.Context) {
		if err = driver.Close(ctx); err != nil {
			panic(err)
		}
	}(driver, ctx)

	if err = driver.VerifyConnectivity(ctx); err != nil {
		panic(err)
	}

	fmt.Printf("Connection to %s established.", dbUri)

	return driver, ctx, err
}
