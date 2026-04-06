package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// formatID converts a BSON value to a filesystem-safe string.
func formatID(val bson.RawValue) string {
	switch val.Type {
	case bson.TypeObjectID:
		oid, ok := val.ObjectIDOK()
		if ok {
			return oid.Hex()
		}
	case bson.TypeString:
		s, ok := val.StringValueOK()
		if ok {
			return sanitizeFilename(s)
		}
	case bson.TypeInt32:
		i, ok := val.Int32OK()
		if ok {
			return fmt.Sprintf("%d", i)
		}
	case bson.TypeInt64:
		i, ok := val.Int64OK()
		if ok {
			return fmt.Sprintf("%d", i)
		}
	case bson.TypeDouble:
		f, ok := val.DoubleOK()
		if ok {
			return fmt.Sprintf("%g", f)
		}
	}
	// Fallback: hex-encode the raw BSON
	return fmt.Sprintf("%x", val.Value)
}

// sanitizeFilename replaces filesystem-unsafe characters.
func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer("/", "_", "\x00", "_")
	return replacer.Replace(s)
}

// buildIDFilter creates a BSON filter for a document ID string.
// Tries ObjectID first, then falls back to string.
func buildIDFilter(docID string) (bson.D, error) {
	// Try parsing as ObjectID (24 hex chars)
	if len(docID) == 24 {
		oid, err := bson.ObjectIDFromHex(docID)
		if err == nil {
			return bson.D{{Key: "_id", Value: oid}}, nil
		}
	}
	// Fallback: use as string
	return bson.D{{Key: "_id", Value: docID}}, nil
}

// formatJSON converts a raw BSON document to pretty-printed JSON.
func formatJSON(raw bson.Raw) ([]byte, error) {
	// Convert BSON to a Go map for JSON serialization
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Add trailing newline
	data = append(data, '\n')
	return data, nil
}

// parseJSON converts JSON bytes to a BSON document for insertion.
func parseJSON(data []byte) (bson.D, error) {
	var doc bson.D
	if err := bson.UnmarshalExtJSON(data, false, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return doc, nil
}
