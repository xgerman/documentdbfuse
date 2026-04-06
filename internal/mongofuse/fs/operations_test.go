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
