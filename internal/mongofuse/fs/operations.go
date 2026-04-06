package fs

import (
	"context"
	"strings"

	"github.com/xgerman/mongofuse/internal/mongofuse/db"
)

// Operations handles all filesystem operations by delegating to the MongoDB client.
type Operations struct {
	client *db.Client
}

// NewOperations creates a new filesystem operations handler.
func NewOperations(client *db.Client) *Operations {
	return &Operations{client: client}
}

// PathInfo represents a parsed filesystem path.
type PathInfo struct {
	Database   string
	Collection string
	DocumentID string    // empty if listing collection
	Extension  string    // "json", "bson", etc.
	Pipeline   *Pipeline // non-nil when aggregation segments are present
}

// ParsePath converts a filesystem path to a PathInfo.
// Expected format: /<database>/<collection>/<docid>.json
// or with pipeline: /<database>/<collection>/.match/field/value/.sort/field/.export/json
func ParsePath(path string) PathInfo {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	info := PathInfo{}
	if len(parts) >= 1 && parts[0] != "" {
		info.Database = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		info.Collection = parts[1]
	}
	if len(parts) > 2 {
		remaining := parts[2:]
		before, pipelineParts := extractPipelineSegments(remaining)

		if len(pipelineParts) > 0 {
			// Pipeline path: parse aggregation segments
			pipeline, err := ParsePipeline(pipelineParts)
			if err == nil {
				info.Pipeline = pipeline
				if pipeline.ExportFormat != "" {
					info.Extension = pipeline.ExportFormat
				}
			}
			// If there are non-pipeline parts before the pipeline, treat as doc ID
			if len(before) > 0 {
				filename := strings.Join(before, "/")
				if idx := strings.LastIndex(filename, "."); idx > 0 {
					info.Extension = filename[idx+1:]
					info.DocumentID = filename[:idx]
				} else {
					info.DocumentID = filename
				}
			}
		} else if len(before) > 0 {
			// No pipeline segments — original behavior
			filename := strings.Join(before, "/")
			if idx := strings.LastIndex(filename, "."); idx > 0 {
				info.Extension = filename[idx+1:]
				info.DocumentID = filename[:idx]
			} else {
				info.DocumentID = filename
			}
		}
	}

	return info
}

// ReadDir lists entries for a given path.
func (o *Operations) ReadDir(ctx context.Context, path string) ([]string, error) {
	info := ParsePath(path)

	switch {
	case info.Database == "":
		// Root: list databases
		return o.client.ListDatabases(ctx)

	case info.Collection == "":
		// Database: list collections
		return o.client.ListCollections(ctx, info.Database)

	default:
		// Collection: list document IDs
		ids, err := o.client.ListDocumentIDs(ctx, info.Database, info.Collection)
		if err != nil {
			return nil, err
		}
		// Append .json extension for display
		for i, id := range ids {
			ids[i] = id + ".json"
		}
		return ids, nil
	}
}

// ReadFile returns document content as JSON, or aggregation results if pipeline segments are present.
func (o *Operations) ReadFile(ctx context.Context, path string) ([]byte, error) {
	info := ParsePath(path)

	// Pipeline aggregation: return results as a file
	if info.Pipeline != nil && len(info.Pipeline.Stages) > 0 {
		if info.Database == "" || info.Collection == "" {
			return nil, ErrNotFound
		}
		return o.client.Aggregate(ctx, info.Database, info.Collection, info.Pipeline.Stages)
	}

	if info.DocumentID == "" {
		return nil, ErrIsDirectory
	}
	return o.client.GetDocument(ctx, info.Database, info.Collection, info.DocumentID)
}

// WriteFile creates or replaces a document.
func (o *Operations) WriteFile(ctx context.Context, path string, data []byte) error {
	info := ParsePath(path)
	if info.DocumentID == "" {
		return ErrIsDirectory
	}

	// Upsert: replace if exists, insert if not
	return o.client.ReplaceDocument(ctx, info.Database, info.Collection, info.DocumentID, data)
}

// Remove deletes a document or drops a collection.
func (o *Operations) Remove(ctx context.Context, path string, isDir bool) error {
	info := ParsePath(path)

	if isDir || info.DocumentID == "" {
		if info.Collection != "" {
			return o.client.DropCollection(ctx, info.Database, info.Collection)
		}
		return ErrNotSupported
	}

	return o.client.DeleteDocument(ctx, info.Database, info.Collection, info.DocumentID)
}

// MkDir creates a collection (or implicitly a database).
func (o *Operations) MkDir(ctx context.Context, path string) error {
	info := ParsePath(path)
	if info.Collection == "" {
		return ErrNotSupported // Can't explicitly create databases in MongoDB
	}
	return o.client.CreateCollection(ctx, info.Database, info.Collection)
}
