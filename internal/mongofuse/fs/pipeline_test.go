package fs

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestParsePipeline_Empty(t *testing.T) {
	p, err := ParsePipeline(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Stages) != 0 {
		t.Errorf("expected 0 stages, got %d", len(p.Stages))
	}
	if p.ExportFormat != "" {
		t.Errorf("expected empty export format, got %q", p.ExportFormat)
	}
}

func TestParsePipeline_MatchString(t *testing.T) {
	p, err := ParsePipeline([]string{".match", "status", "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(p.Stages))
	}
	expected := bson.D{{Key: "$match", Value: bson.D{{Key: "status", Value: "active"}}}}
	assertStageEqual(t, p.Stages[0], expected)
}

func TestParsePipeline_MatchNumeric(t *testing.T) {
	p, err := ParsePipeline([]string{".match", "age", "30"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(p.Stages))
	}
	stage := p.Stages[0]
	matchVal := stage[0].Value.(bson.D)[0].Value
	if matchVal != int64(30) {
		t.Errorf("expected int64(30), got %v (%T)", matchVal, matchVal)
	}
}

func TestParsePipeline_MatchBool(t *testing.T) {
	p, err := ParsePipeline([]string{".match", "active", "true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matchVal := p.Stages[0][0].Value.(bson.D)[0].Value
	if matchVal != true {
		t.Errorf("expected true, got %v (%T)", matchVal, matchVal)
	}
}

func TestParsePipeline_MatchNull(t *testing.T) {
	p, err := ParsePipeline([]string{".match", "deleted", "null"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matchVal := p.Stages[0][0].Value.(bson.D)[0].Value
	if matchVal != nil {
		t.Errorf("expected nil, got %v (%T)", matchVal, matchVal)
	}
}

func TestParsePipeline_SortAscending(t *testing.T) {
	p, err := ParsePipeline([]string{".sort", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := bson.D{{Key: "$sort", Value: bson.D{{Key: "name", Value: 1}}}}
	assertStageEqual(t, p.Stages[0], expected)
}

func TestParsePipeline_SortDescending(t *testing.T) {
	p, err := ParsePipeline([]string{".sort", "-created_at"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := bson.D{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}}
	assertStageEqual(t, p.Stages[0], expected)
}

func TestParsePipeline_Limit(t *testing.T) {
	p, err := ParsePipeline([]string{".limit", "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := bson.D{{Key: "$limit", Value: int64(10)}}
	assertStageEqual(t, p.Stages[0], expected)
}

func TestParsePipeline_LimitInvalid(t *testing.T) {
	_, err := ParsePipeline([]string{".limit", "abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric limit")
	}
}

func TestParsePipeline_Skip(t *testing.T) {
	p, err := ParsePipeline([]string{".skip", "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := bson.D{{Key: "$skip", Value: int64(5)}}
	assertStageEqual(t, p.Stages[0], expected)
}

func TestParsePipeline_SkipInvalid(t *testing.T) {
	_, err := ParsePipeline([]string{".skip", "xyz"})
	if err == nil {
		t.Fatal("expected error for non-numeric skip")
	}
}

func TestParsePipeline_Project(t *testing.T) {
	p, err := ParsePipeline([]string{".project", "name,email,age"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(p.Stages))
	}
	proj := p.Stages[0][0].Value.(bson.D)
	// Should have _id, name, email, age
	if len(proj) != 4 {
		t.Errorf("expected 4 project fields, got %d: %v", len(proj), proj)
	}
	// First field should be _id
	if proj[0].Key != "_id" {
		t.Errorf("expected first project field to be _id, got %s", proj[0].Key)
	}
}

func TestParsePipeline_ExportJSON(t *testing.T) {
	p, err := ParsePipeline([]string{".export", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ExportFormat != "json" {
		t.Errorf("expected export format 'json', got %q", p.ExportFormat)
	}
}

func TestParsePipeline_Chained(t *testing.T) {
	parts := []string{".match", "status", "active", ".sort", "-created_at", ".limit", "10"}
	p, err := ParsePipeline(parts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(p.Stages))
	}

	// Stage 0: $match
	if p.Stages[0][0].Key != "$match" {
		t.Errorf("stage 0: expected $match, got %s", p.Stages[0][0].Key)
	}
	// Stage 1: $sort
	if p.Stages[1][0].Key != "$sort" {
		t.Errorf("stage 1: expected $sort, got %s", p.Stages[1][0].Key)
	}
	sortField := p.Stages[1][0].Value.(bson.D)[0]
	if sortField.Key != "created_at" || sortField.Value != -1 {
		t.Errorf("stage 1: expected {created_at: -1}, got {%s: %v}", sortField.Key, sortField.Value)
	}
	// Stage 2: $limit
	if p.Stages[2][0].Key != "$limit" {
		t.Errorf("stage 2: expected $limit, got %s", p.Stages[2][0].Key)
	}
	if p.Stages[2][0].Value != int64(10) {
		t.Errorf("stage 2: expected 10, got %v", p.Stages[2][0].Value)
	}
}

func TestParsePipeline_ChainedWithExport(t *testing.T) {
	parts := []string{".match", "type", "order", ".limit", "5", ".export", "json"}
	p, err := ParsePipeline(parts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(p.Stages))
	}
	if p.ExportFormat != "json" {
		t.Errorf("expected export format 'json', got %q", p.ExportFormat)
	}
}

func TestParsePipeline_MatchMissingValue(t *testing.T) {
	_, err := ParsePipeline([]string{".match", "field"})
	if err == nil {
		t.Fatal("expected error for .match with missing value")
	}
}

func TestParsePipeline_SortMissingField(t *testing.T) {
	_, err := ParsePipeline([]string{".sort"})
	if err == nil {
		t.Fatal("expected error for .sort with missing field")
	}
}

func TestParsePipeline_UnknownSegment(t *testing.T) {
	_, err := ParsePipeline([]string{".unknown", "foo"})
	if err == nil {
		t.Fatal("expected error for unknown segment")
	}
}

func TestParsePipeline_MatchFloat(t *testing.T) {
	p, err := ParsePipeline([]string{".match", "score", "3.14"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matchVal := p.Stages[0][0].Value.(bson.D)[0].Value
	if matchVal != 3.14 {
		t.Errorf("expected 3.14, got %v (%T)", matchVal, matchVal)
	}
}

func TestExtractPipelineSegments(t *testing.T) {
	parts := []string{"mydb", "users", ".match", "status", "active"}
	before, pipeline := extractPipelineSegments(parts)
	if len(before) != 2 || before[0] != "mydb" || before[1] != "users" {
		t.Errorf("unexpected before: %v", before)
	}
	if len(pipeline) != 3 || pipeline[0] != ".match" {
		t.Errorf("unexpected pipeline: %v", pipeline)
	}
}

func TestExtractPipelineSegments_NoPipeline(t *testing.T) {
	parts := []string{"mydb", "users"}
	before, pipeline := extractPipelineSegments(parts)
	if len(before) != 2 {
		t.Errorf("unexpected before: %v", before)
	}
	if len(pipeline) != 0 {
		t.Errorf("expected no pipeline segments, got %v", pipeline)
	}
}

// assertStageEqual compares two bson.D by marshalling to JSON.
func assertStageEqual(t *testing.T, got, expected bson.D) {
	t.Helper()
	gotBytes, err1 := bson.MarshalExtJSON(got, false, false)
	expBytes, err2 := bson.MarshalExtJSON(expected, false, false)
	if err1 != nil || err2 != nil {
		t.Fatalf("marshal error: got=%v expected=%v", err1, err2)
	}
	if string(gotBytes) != string(expBytes) {
		t.Errorf("stage mismatch:\n  got:      %s\n  expected: %s", gotBytes, expBytes)
	}
}
