// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"
)

// tdLinks returns a CommonPageData with the footer/nav Links populated, needed
// by the utility pages that reference .Links / .RequestURI.
func tdLinks(requestURI string) *CommonPageData {
	cd := tdCommon()
	cd.RequestURI = requestURI
	cd.Links = &links{
		InsightAPIDocs: "https://github.com/decred/dcrdata#insight-api",
		TestnetSearch:  "https://testnet.decred.org/?search=",
		MainnetSearch:  "https://explorer.dcrdata.org/?search=",
	}
	return cd
}

func TestV2VerifyMessageGET(t *testing.T) {
	// GET form: VerifyMessageResult is nil, so no result banner should appear.
	out := renderV2(t, "verify_message", struct {
		*CommonPageData
		VerifyMessageResult *verifyMessageResult
	}{tdCommon(), nil})
	mustContain(t, out, "verify_message",
		"class=\"sidebar\"",
		"Verify Message",
		"action=\"/verify-message\"",
		"method=\"post\"",
		"name=\"address\"",
		"name=\"message\"",
		"name=\"signature\"",
	)
	if strings.Contains(out, "Matching signature") || strings.Contains(out, "Verification error") {
		t.Errorf("GET verify_message should not render a result banner")
	}
}

func TestV2VerifyMessageValid(t *testing.T) {
	out := renderV2(t, "verify_message", struct {
		*CommonPageData
		VerifyMessageResult *verifyMessageResult
	}{tdCommon(), &verifyMessageResult{
		Address: "DsXXXabc", Message: "hello world", Signature: "ZZsig==", Valid: true,
	}})
	mustContain(t, out, "verify_message",
		"Matching signature",
		"value=\"DsXXXabc\"", // address echoed back into the form
		"hello world",        // message echoed into the textarea
	)
}

func TestV2VerifyMessageError(t *testing.T) {
	out := renderV2(t, "verify_message", struct {
		*CommonPageData
		VerifyMessageResult *verifyMessageResult
	}{tdCommon(), &verifyMessageResult{
		Address: "DsBad", Signature: "x", Message: "m", Error: "malformed base64 encoding",
	}})
	mustContain(t, out, "verify_message",
		"Verification error",
		"malformed base64 encoding",
	)
}

func TestV2VerifyMessageNotSigned(t *testing.T) {
	out := renderV2(t, "verify_message", struct {
		*CommonPageData
		VerifyMessageResult *verifyMessageResult
	}{tdCommon(), &verifyMessageResult{
		Address: "DsX", Signature: "s", Message: "m", Valid: false,
	}})
	mustContain(t, out, "verify_message", "not signed by this address")
}

func TestV2InsightRootRenders(t *testing.T) {
	out := renderV2(t, "insight_root", struct {
		*CommonPageData
	}{tdLinks("/insight")})
	mustContain(t, out, "insight_root",
		"class=\"sidebar\"",
		"Insight API",
		"api/status", // prefixPath built the example endpoint
		"https://github.com/decred/dcrdata#insight-api", // docs link
	)
}

// statusData mirrors the anonymous struct StatusPage execs.
func statusData(cd *CommonPageData, sType expStatus, code, msg, info string) interface{} {
	return struct {
		*CommonPageData
		StatusType     expStatus
		Code           string
		Message        string
		AdditionalInfo string
	}{cd, sType, code, msg, info}
}

func TestV2StatusNotFound(t *testing.T) {
	out := renderV2(t, "status", statusData(tdCommon(), ExpStatusNotFound,
		"404", "could not find that transaction", "deadbeef"))
	mustContain(t, out, "status",
		"Not Found",
		"No matching page",
		"could not find that transaction",
	)
}

func TestV2StatusSyncing(t *testing.T) {
	out := renderV2(t, "status", statusData(tdCommon(), ExpStatusSyncing,
		"202", "Blockchain sync is running. Please wait.", ""))
	mustContain(t, out, "status",
		"Blocks Syncing",
		"sync is running",
		"dot syncing",            // animated indicator
		"http-equiv=\"refresh\"", // auto-reloads while syncing
	)
}

func TestV2StatusWrongNet(t *testing.T) {
	// NetName is Mainnet (from tdCommon); a message mentioning "testnet" plus a
	// non-empty AdditionalInfo should surface the testnet switch link.
	out := renderV2(t, "status", statusData(tdLinks("/tx/abc"), ExpStatusWrongNetwork,
		"422", "This is a testnet transaction", "abc"))
	mustContain(t, out, "status",
		"Wrong Network",
		"This is a testnet transaction",
		"switch to testnet",
		"https://testnet.decred.org/?search=abc",
	)
}

func TestV2StatusError(t *testing.T) {
	out := renderV2(t, "status", statusData(tdCommon(), ExpStatusError,
		"500", "Something went very wrong", ""))
	mustContain(t, out, "status", "Error", "Something went very wrong")
}
