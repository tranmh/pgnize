package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestFSRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "uploads/anon/2026/06/abc.jpg"
	want := []byte("fake-jpeg-bytes")

	if err := fs.Put(ctx, key, bytes.NewReader(want), int64(len(want)), "image/jpeg"); err != nil {
		t.Fatalf("put: %v", err)
	}
	rc, ct, err := fs.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %q want image/jpeg", ct)
	}
	if err := fs.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, _, err := fs.Get(ctx, key); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestFSTraversalContained(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFS(dir)
	ctx := context.Background()
	// A traversal-looking key must stay within Root.
	if err := fs.Put(ctx, "../../etc/evil", bytes.NewReader([]byte("x")), 1, "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, _, err := fs.Get(ctx, "../../etc/evil"); err != nil {
		t.Fatalf("get: %v", err)
	}
}
