// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"net/http/httptest"
	"testing"

	"github.com/decred/dcrdata/v8/explorer/types"
)

// TestPublicBaseURL verifies the base-URL derivation for absolute links,
// notably that a request whose Host was blanked by the AllowedHosts middleware
// cannot smuggle a host (or scheme) in via forwarded headers.
func TestPublicBaseURL(t *testing.T) {
	allowed := []string{"explorer.example.com", "alt.example.com"}

	// Normal proxied request: Host matched the allowlist, proxy set the proto.
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "explorer.example.com"
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := publicBaseURL(r, allowed); got != "https://explorer.example.com" {
		t.Errorf("proxied request: got %q", got)
	}

	// Host blanked by AllowedHosts + attacker-supplied forwarded headers: the
	// canonical host must win and the spoofed scheme must be ignored.
	r = httptest.NewRequest("GET", "/", nil)
	r.Host = ""
	r.Header.Set("X-Forwarded-Host", "evil.example")
	r.Header.Set("X-Forwarded-Proto", "http")
	if got := publicBaseURL(r, allowed); got != "https://explorer.example.com" {
		t.Errorf("blanked host: got %q", got)
	}

	// Dev setup: no allowlist, no proxy, no TLS.
	r = httptest.NewRequest("GET", "/", nil)
	r.Host = "localhost:7777"
	if got := publicBaseURL(r, nil); got != "http://localhost:7777" {
		t.Errorf("dev request: got %q", got)
	}
}

// TestHomeInfoCopyRace exercises homeInfoCopy against a writer mutating the
// shared HomeInfo in place, as Store does on every block. Run with -race; the
// old pattern of reading fields after RUnlock fails this.
func TestHomeInfoCopyRace(t *testing.T) {
	exp := &explorerUI{pageData: &pageData{HomeInfo: &types.HomeInfo{}}}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			exp.pageData.Lock()
			exp.pageData.HomeInfo.CoinSupply++
			exp.pageData.HomeInfo.RewardPeriod = "28.13 days"
			exp.pageData.HomeInfo.PoolInfo = types.TicketPoolInfo{Size: uint32(i)}
			exp.pageData.Unlock()
		}
	}()
	for alive := true; alive; {
		select {
		case <-done:
			alive = false
		default:
		}
		hi := exp.homeInfoCopy()
		_, _, _ = hi.CoinSupply, hi.RewardPeriod, hi.PoolInfo.Size
	}
}
