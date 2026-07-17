// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
)

// AssetManager produces content-hashed URLs for static files served out of a
// root directory (typically "public"). It is the no-build replacement for
// webpack's content-hash cache-busting: templates call the "asset" function
// (registered in makeTemplateFuncMap) to turn a logical path like
// "css/v2/main.css" into "/css/v2/main.css?v=<hash>", so browsers may cache
// aggressively yet pick up new files immediately after a rebuild + restart.
type AssetManager struct {
	root string
	mu   sync.RWMutex
	ver  map[string]string
}

// NewAssetManager returns an AssetManager that hashes files found under root.
func NewAssetManager(root string) *AssetManager {
	return &AssetManager{root: root, ver: make(map[string]string)}
}

// URL maps a logical asset path (relative to root, forward-slashed) to a
// rooted, version-stamped URL such as "/css/v2/main.css?v=0a1b2c3d4e5f". The
// file's content hash is computed once and cached for the life of the process
// (a deploy restarts dcrdata, so a rebuild always re-hashes). A file that
// cannot be read falls back to an unstamped URL, so a missing asset is merely
// uncached rather than fatal.
func (a *AssetManager) URL(p string) string {
	a.mu.RLock()
	v, ok := a.ver[p]
	a.mu.RUnlock()
	if ok {
		return urlFor(p, v)
	}

	v = a.hash(p)
	// Don't cache a failed hash: a transient read error (e.g. a file mid-swap
	// during deploy) would otherwise pin the unstamped URL for the process
	// lifetime, disabling cache busting until a restart. Retry next request.
	if v != "" {
		a.mu.Lock()
		a.ver[p] = v
		a.mu.Unlock()
	}
	return urlFor(p, v)
}

func urlFor(p, v string) string {
	if v == "" {
		return "/" + p
	}
	return "/" + p + "?v=" + v
}

func (a *AssetManager) hash(p string) string {
	b, err := os.ReadFile(filepath.Join(a.root, filepath.FromSlash(p)))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:12]
}
