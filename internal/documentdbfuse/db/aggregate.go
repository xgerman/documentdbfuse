package db

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Aggregate runs an aggregation pipeline and returns results as pretty-printed JSON.
func (c *Client) Aggregate(ctx context.Context, dbName, collName string, pipeline []bson.D) ([]byte, error) {
	coll := c.client.Database(dbName).Collection(collName)

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
		results = append(results, doc)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if results == nil {
		results = []bson.M{}
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	data = append(data, '\n')
	return data, nil
}

// AggregateIDs runs an aggregation pipeline and returns matching document IDs as "id.json" strings.
func (c *Client) AggregateIDs(ctx context.Context, dbName, collName string, pipeline []bson.D) ([]string, error) {
	coll := c.client.Database(dbName).Collection(collName)

	// Append a $project stage to only fetch _id
	idPipeline := append(append([]bson.D{}, pipeline...), bson.D{{Key: "$project", Value: bson.D{{Key: "_id", Value: 1}}}})

	cursor, err := coll.Aggregate(ctx, idPipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		idVal := cursor.Current.Lookup("_id")
		ids = append(ids, formatID(idVal)+".json")
	}
	return ids, cursor.Err()
}
