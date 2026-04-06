package db

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AggregateFormat runs an aggregation pipeline and returns results in the specified format.
// Supported formats: "json" (default), "csv", "tsv".
func (c *Client) AggregateFormat(ctx context.Context, dbName, collName string, pipeline []bson.D, format string) ([]byte, error) {
	results, err := c.aggregateRaw(ctx, dbName, collName, pipeline)
	if err != nil {
		return nil, err
	}

	switch format {
	case "csv":
		return formatDelimited(results, ',')
	case "tsv":
		return formatDelimited(results, '\t')
	default:
		return formatJSONArray(results)
	}
}

// Aggregate runs an aggregation pipeline and returns results as pretty-printed JSON.
func (c *Client) Aggregate(ctx context.Context, dbName, collName string, pipeline []bson.D) ([]byte, error) {
	return c.AggregateFormat(ctx, dbName, collName, pipeline, "json")
}

// aggregateRaw runs the pipeline and returns raw results.
func (c *Client) aggregateRaw(ctx context.Context, dbName, collName string, pipeline []bson.D) ([]bson.M, error) {
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
	return results, nil
}

// formatJSONArray formats results as a pretty-printed JSON array.
func formatJSONArray(results []bson.M) ([]byte, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}
	data = append(data, '\n')
	return data, nil
}

// formatDelimited formats results as CSV or TSV with a header row.
func formatDelimited(results []bson.M, delimiter rune) ([]byte, error) {
	if len(results) == 0 {
		return []byte{}, nil
	}

	// Collect all keys across all documents for consistent columns
	keySet := map[string]bool{}
	for _, doc := range results {
		for k := range doc {
			keySet[k] = true
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Comma = delimiter

	// Header
	if err := w.Write(keys); err != nil {
		return nil, err
	}

	// Rows
	for _, doc := range results {
		row := make([]string, len(keys))
		for i, k := range keys {
			if v, ok := doc[k]; ok {
				row[i] = formatCellValue(v)
			}
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// formatCellValue converts a BSON value to a string for CSV/TSV cells.
func formatCellValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bson.M, bson.D, bson.A:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// AggregateCount appends a $count stage to the pipeline and returns the count.
func (c *Client) AggregateCount(ctx context.Context, dbName, collName string, pipeline []bson.D) (int64, error) {
	countPipeline := append(append([]bson.D{}, pipeline...), bson.D{{Key: "$count", Value: "count"}})
	results, err := c.aggregateRaw(ctx, dbName, collName, countPipeline)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	switch v := results[0]["count"].(type) {
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unexpected count type: %T", v)
	}
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
