package fs

import (
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Pipeline represents a parsed aggregation pipeline from path segments.
type Pipeline struct {
	Stages       []bson.D
	ExportFormat string // "json" or empty
}

// ParsePipeline extracts aggregation pipeline stages from path segments.
// Segments are expected to start with a dot prefix (e.g., ".match", ".sort").
func ParsePipeline(pathParts []string) (*Pipeline, error) {
	p := &Pipeline{}
	if len(pathParts) == 0 {
		return p, nil
	}

	i := 0
	for i < len(pathParts) {
		seg := pathParts[i]

		switch seg {
		case ".match":
			if i+2 >= len(pathParts) {
				return nil, fmt.Errorf(".match requires field and value: .match/field/value")
			}
			field := pathParts[i+1]
			value := parseMatchValue(pathParts[i+2])
			p.Stages = append(p.Stages, bson.D{{Key: "$match", Value: bson.D{{Key: field, Value: value}}}})
			i += 3

		case ".sort":
			if i+1 >= len(pathParts) {
				return nil, fmt.Errorf(".sort requires a field: .sort/field or .sort/-field")
			}
			fieldArg := pathParts[i+1]
			order := 1
			if strings.HasPrefix(fieldArg, "-") {
				order = -1
				fieldArg = fieldArg[1:]
			}
			if fieldArg == "" {
				return nil, fmt.Errorf(".sort field name cannot be empty")
			}
			p.Stages = append(p.Stages, bson.D{{Key: "$sort", Value: bson.D{{Key: fieldArg, Value: order}}}})
			i += 2

		case ".limit":
			if i+1 >= len(pathParts) {
				return nil, fmt.Errorf(".limit requires a number: .limit/N")
			}
			n, err := strconv.ParseInt(pathParts[i+1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf(".limit value must be a number: %w", err)
			}
			p.Stages = append(p.Stages, bson.D{{Key: "$limit", Value: n}})
			i += 2

		case ".skip":
			if i+1 >= len(pathParts) {
				return nil, fmt.Errorf(".skip requires a number: .skip/N")
			}
			n, err := strconv.ParseInt(pathParts[i+1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf(".skip value must be a number: %w", err)
			}
			p.Stages = append(p.Stages, bson.D{{Key: "$skip", Value: n}})
			i += 2

		case ".project":
			if i+1 >= len(pathParts) {
				return nil, fmt.Errorf(".project requires fields: .project/f1,f2,f3")
			}
			fields := strings.Split(pathParts[i+1], ",")
			proj := bson.D{{Key: "_id", Value: 1}}
			for _, f := range fields {
				f = strings.TrimSpace(f)
				if f != "" && f != "_id" {
					proj = append(proj, bson.E{Key: f, Value: 1})
				}
			}
			p.Stages = append(p.Stages, bson.D{{Key: "$project", Value: proj}})
			i += 2

		case ".export":
			if i+1 >= len(pathParts) {
				return nil, fmt.Errorf(".export requires a format: .export/json")
			}
			p.ExportFormat = pathParts[i+1]
			i += 2

		default:
			return nil, fmt.Errorf("unknown pipeline segment: %s", seg)
		}
	}

	return p, nil
}

// parseMatchValue tries to interpret a string value as a typed value.
// Tries int, float, bool, null in order, falls back to string.
func parseMatchValue(s string) interface{} {
	if s == "null" {
		return nil
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// extractPipelineSegments splits path parts into pre-pipeline parts and pipeline parts.
// Pipeline parts start at the first element beginning with ".".
func extractPipelineSegments(parts []string) (before []string, pipelineParts []string) {
	for i, p := range parts {
		if strings.HasPrefix(p, ".") {
			return parts[:i], parts[i:]
		}
	}
	return parts, nil
}
