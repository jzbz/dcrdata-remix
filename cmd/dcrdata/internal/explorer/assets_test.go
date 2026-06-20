// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetManagerURL(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "css", "v2"), 0o755); err != nil {
		t.Fatal(err)
	}
	cssPath := filepath.Join(root, "css", "v2", "main.css")
	if err := os.WriteFile(cssPath, []byte("body{color:red}"), 0o644); err != nil {
		t.Fatal(err)
	}

	am := NewAssetManager(root)

	// A present file gets a 12-hex-char version stamp on a rooted URL.
	got := am.URL("css/v2/main.css")
	prefix := "/css/v2/main.css?v="
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("URL = %q, want prefix %q", got, prefix)
	}
	ver := strings.TrimPrefix(got, prefix)
	if len(ver) != 12 {
		t.Errorf("version %q: want 12 chars, got %d", ver, len(ver))
	}

	// The hash is cached: same path returns the same stamp even if the file
	// changes underneath (a deploy restart re-hashes).
	if err := os.WriteFile(cssPath, []byte("body{color:blue}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got2 := am.URL("css/v2/main.css"); got2 != got {
		t.Errorf("cached URL changed: %q != %q", got2, got)
	}

	// Distinct content yields a distinct stamp.
	jsPath := filepath.Join(root, "css", "v2", "other.css")
	if err := os.WriteFile(jsPath, []byte("a{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if am.URL("css/v2/other.css") == got {
		t.Errorf("different file produced the same version stamp")
	}

	// A missing file falls back to an unstamped, rooted URL (not fatal).
	if got := am.URL("css/v2/missing.css"); got != "/css/v2/missing.css" {
		t.Errorf("missing asset URL = %q, want %q", got, "/css/v2/missing.css")
	}
}
