package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestLocalStorage(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	ls := NewLocalStorage(dir)
	content := "hello world!"
	key := "9f8b2c1ad4e3"
	newFlow := strings.NewReader(content)

	_, err := ls.Save(ctx, key, newFlow)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	file, err := ls.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(got) != content {
		t.Fatalf("Content does not match! Got %q, want %q", string(got), content)
	}
	t.Logf("Content does match! Got %q, want %q", string(got), content)
	file.Close()
	er := ls.Delete(ctx, key)
	if er != nil {
		t.Fatalf("Delete failed: %v", er)
	}

}
