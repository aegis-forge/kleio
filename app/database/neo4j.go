package database

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// executeQueryNeo sends the actual query, together with the content, to neo4j
func executeQueryNeo(query string, content map[string]any, driver neo4j.DriverWithContext, ctx context.Context) {
	if _, err := neo4j.ExecuteQuery(
		ctx, driver, query, content, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"),
	); err != nil {
		panic(err)
	}
}

// executeQueryNeo send the actual query, together with the content, to neo4j. Finally, it returns the resulting records
func executeQueryWithRetNeo(query string, content map[string]any, driver neo4j.DriverWithContext, ctx context.Context) []*neo4j.Record {
	if result, err := neo4j.ExecuteQuery(
		ctx, driver, query, content, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"),
	); err != nil {
		panic(err)
	} else {
		return result.Records
	}
}
