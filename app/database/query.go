package database

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ExecuteQueryNeo sends the actual query, together with the content, to neo4j
func ExecuteQueryNeo(query string, content map[string]any, driver neo4j.DriverWithContext, ctx context.Context) {
	if _, err := neo4j.ExecuteQuery(
		ctx, driver, query, content, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"),
	); err != nil {
		panic(err)
	}
}

// ExecuteQueryNeo send the actual query, together with the content, to neo4j. Finally, it returns the resulting records
func ExecuteQueryWithRetNeo(query string, content map[string]any, driver neo4j.DriverWithContext, ctx context.Context) []*neo4j.Record {
	if result, err := neo4j.ExecuteQuery(
		ctx, driver, query, content, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"),
	); err != nil {
		panic(err)
	} else {
		return result.Records
	}
}

// ExecuteQueryMongo executes the selected query on the respective mongodb collection
func ExecuteQueryMongo(content any, collection string, typz string, client mongo.Database) string {
	switch typz {
	case "insert":
		res, err := client.Collection(collection).InsertOne(context.Background(), content)

		if err != nil {
			panic(err)
		}

		return res.InsertedID.(bson.ObjectID).Hex()
	default:
		panic("the type of operation does not exist")
	}
}
