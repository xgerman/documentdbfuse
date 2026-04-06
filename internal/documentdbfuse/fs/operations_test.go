package fs

import (
	"testing"
)

func TestParsePath_Root(t *testing.T) {
	info := ParsePath("/")
	if info.Database != "" || info.Collection != "" || info.DocumentID != "" {
		t.Errorf("expected empty PathInfo for root, got %+v", info)
	}
}

func TestParsePath_Database(t *testing.T) {
	info := ParsePath("/mydb")
	if info.Database != "mydb" || info.Collection != "" {
		t.Errorf("expected Database=mydb, got %+v", info)
	}
}

func TestParsePath_Collection(t *testing.T) {
	info := ParsePath("/mydb/users")
	if info.Database != "mydb" || info.Collection != "users" || info.DocumentID != "" {
		t.Errorf("expected Database=mydb Collection=users, got %+v", info)
	}
}

func TestParsePath_Document(t *testing.T) {
	info := ParsePath("/mydb/users/507f1f77bcf86cd7994.json")
	if info.Database != "mydb" || info.Collection != "users" || info.DocumentID != "507f1f77bcf86cd7994" || info.Extension != "json" {
		t.Errorf("unexpected PathInfo: %+v", info)
	}
}

func TestParsePath_NoExtension(t *testing.T) {
	info := ParsePath("/mydb/users/123")
	if info.DocumentID != "123" || info.Extension != "" {
		t.Errorf("expected DocumentID=123 no extension, got %+v", info)
	}
}

func TestParsePath_Pipeline(t *testing.T) {
info := ParsePath("/sampledb/users/.match/isActive/true/.export/json")
if info.Database != "sampledb" || info.Collection != "users" {
t.Errorf("unexpected db/coll: %+v", info)
}
if info.Pipeline == nil {
t.Fatal("expected Pipeline to be non-nil")
}
if len(info.Pipeline.Stages) != 1 {
t.Errorf("expected 1 stage, got %d: %+v", len(info.Pipeline.Stages), info.Pipeline.Stages)
}
if info.Pipeline.ExportFormat != "json" {
t.Errorf("expected ExportFormat=json, got %q", info.Pipeline.ExportFormat)
}
t.Logf("PathInfo: %+v", info)
}
