// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDownsampleChartJSON(t *testing.T) {
	// 10 points into 5 buckets of 2: t keeps bucket-start integers, y series
	// are bucket-averaged, scalars pass through, and the response is flagged.
	in := `{"t":[0,10,20,30,40,50,60,70,80,90],` +
		`"supply":[1,2,3,4,5,6,7,8,9,10],"bin":"day"}`
	out, err := downsampleChartJSON([]byte(in), 5)
	if err != nil {
		t.Fatalf("downsampleChartJSON: %v", err)
	}
	var got struct {
		T           []float64 `json:"t"`
		Supply      []float64 `json:"supply"`
		Bin         string    `json:"bin"`
		Downsampled bool      `json:"downsampled"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal downsampled response: %v", err)
	}
	if want := []float64{0, 20, 40, 60, 80}; !reflect.DeepEqual(got.T, want) {
		t.Errorf("t = %v, want bucket starts %v", got.T, want)
	}
	if want := []float64{1.5, 3.5, 5.5, 7.5, 9.5}; !reflect.DeepEqual(got.Supply, want) {
		t.Errorf("supply = %v, want bucket averages %v", got.Supply, want)
	}
	if got.Bin != "day" {
		t.Errorf("bin = %q, want scalar passthrough \"day\"", got.Bin)
	}
	if !got.Downsampled {
		t.Error("downsampled flag not set")
	}
	// The t array must remain integer-valued in the JSON encoding.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatal(err)
	}
	if s := string(raw["t"]); s != "[0,20,40,60,80]" {
		t.Errorf("t encoded as %s, want integer literals", s)
	}
}

func TestDownsampleChartJSONPassthrough(t *testing.T) {
	// Already fits: unchanged.
	in := `{"t":[1,2,3],"supply":[4,5,6]}`
	if out, _ := downsampleChartJSON([]byte(in), 5); string(out) != in {
		t.Errorf("small series changed: %s", out)
	}

	// No explicit x array (index-implicit height/window binning): must be
	// returned at full resolution, since dropping elements would remap every
	// point to a wrong height.
	in = `{"price":[1,2,3,4,5,6,7,8],"count":[8,7,6,5,4,3,2,1],"window":144}`
	if out, _ := downsampleChartJSON([]byte(in), 4); string(out) != in {
		t.Errorf("index-implicit series was downsampled: %s", out)
	}

	// Unparseable input: returned as-is, not an error.
	in = `not json`
	if out, err := downsampleChartJSON([]byte(in), 4); err != nil || string(out) != in {
		t.Errorf("unparseable input: out=%s err=%v", out, err)
	}
}
