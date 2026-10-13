package localstorage

import (
	"context"
	"io"
	"testing"
)

func TestSaveAndOpen(t *testing.T) {
	storage, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(context.Background(), "job_123.png", []byte("image")); err != nil {
		t.Fatal(err)
	}
	reader, err := storage.Open(context.Background(), "job_123.png")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "image" {
		t.Fatalf("data = %q", data)
	}
}

func TestRejectsPathTraversal(t *testing.T) {
	storage, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(context.Background(), "../secret", []byte("x")); err == nil {
		t.Fatal("Save() accepted path traversal")
	}
}
