package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client wraps a MongoDB client with helper methods for filesystem operations.
type Client struct {
	client *mongo.Client
	uri    string
}

// NewClient creates a new MongoDB client.
func NewClient(ctx context.Context, uri string) (*Client, error) {
	opts := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &Client{client: client, uri: uri}, nil
}

// Close disconnects the MongoDB client.
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// ListDatabases returns database names.
func (c *Client) ListDatabases(ctx context.Context) ([]string, error) {
	return c.client.ListDatabaseNames(ctx, map[string]interface{}{})
}

// ListCollections returns collection names for a database.
func (c *Client) ListCollections(ctx context.Context, dbName string) ([]string, error) {
	return c.client.Database(dbName).ListCollectionNames(ctx, map[string]interface{}{})
}

// ListDocumentIDs returns document _id values as strings for a collection.
func (c *Client) ListDocumentIDs(ctx context.Context, dbName, collName string) ([]string, error) {
	coll := c.client.Database(dbName).Collection(collName)
	cursor, err := coll.Find(ctx, map[string]interface{}{}, options.Find().SetProjection(map[string]int{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		raw := cursor.Current
		idVal := raw.Lookup("_id")
		ids = append(ids, formatID(idVal))
	}
	return ids, cursor.Err()
}

// GetDocument returns a document as JSON bytes.
func (c *Client) GetDocument(ctx context.Context, dbName, collName, docID string) ([]byte, error) {
	coll := c.client.Database(dbName).Collection(collName)
	filter, err := buildIDFilter(docID)
	if err != nil {
		return nil, err
	}

	result := coll.FindOne(ctx, filter)
	if result.Err() != nil {
		return nil, fmt.Errorf("document not found: %w", result.Err())
	}

	raw, err := result.Raw()
	if err != nil {
		return nil, err
	}

	return formatJSON(raw)
}

// InsertDocument inserts a new document from JSON bytes.
func (c *Client) InsertDocument(ctx context.Context, dbName, collName string, jsonData []byte) error {
	coll := c.client.Database(dbName).Collection(collName)
	doc, err := parseJSON(jsonData)
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	_, err = coll.InsertOne(ctx, doc)
	return err
}

// ReplaceDocument replaces a document by _id from JSON bytes.
func (c *Client) ReplaceDocument(ctx context.Context, dbName, collName, docID string, jsonData []byte) error {
	coll := c.client.Database(dbName).Collection(collName)
	filter, err := buildIDFilter(docID)
	if err != nil {
		return err
	}

	doc, err := parseJSON(jsonData)
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	_, err = coll.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true))
	return err
}

// DeleteDocument deletes a document by _id.
func (c *Client) DeleteDocument(ctx context.Context, dbName, collName, docID string) error {
	coll := c.client.Database(dbName).Collection(collName)
	filter, err := buildIDFilter(docID)
	if err != nil {
		return err
	}

	_, err = coll.DeleteOne(ctx, filter)
	return err
}

// CreateCollection creates a new collection.
func (c *Client) CreateCollection(ctx context.Context, dbName, collName string) error {
	return c.client.Database(dbName).CreateCollection(ctx, collName)
}

// DropCollection drops a collection.
func (c *Client) DropCollection(ctx context.Context, dbName, collName string) error {
	return c.client.Database(dbName).Collection(collName).Drop(ctx)
}
