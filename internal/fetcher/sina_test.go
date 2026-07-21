package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// Real captured samples from hq.sinajs.cn (2026-07-21, GBK-decoded).
const sinaSampleNormal = `var hq_str_fu_000217="华安黄金ETF联接C,11:31:00,2.9656,2.9401,2.9401,0.0101,0.8673,2026-07-21,2.9691,0.9864";`
const sinaSampleNegative = `var hq_str_fu_001595="天弘中证银行ETF联接C,11:31:00,1.6926,1.7227,1.7227,-0.1475,-1.7473,2026-07-21,1.6917,-1.7995";`
const sinaSampleEmpty = `var hq_str_fu_110037="";`
const sinaSampleETF = `var hq_str_fu_562500="华夏中证机器人ETF,11:32:00,0.9662,0.9349,0.9349,0.0621,3.348,2026-07-21,0.9655,3.2731";`

func TestParseSinaFundQuote_Normal(t *testing.T) {
	rt, ok := ParseSinaFundQuote(sinaSampleNormal)
	if !ok {
		t.Fatal("expected line to match")
	}
	if !rt.Available {
		t.Fatal("expected Available=true")
	}
	if rt.Code != "000217" {
		t.Errorf("Code = %q, want 000217", rt.Code)
	}
	if rt.Name != "华安黄金ETF联接C" {
		t.Errorf("Name = %q, want 华安黄金ETF联接C", rt.Name)
	}
	if rt.EstimatedNAV != 2.9656 {
		t.Errorf("EstimatedNAV = %f, want 2.9656", rt.EstimatedNAV)
	}
	if rt.PreviousNAV != 2.9401 {
		t.Errorf("PreviousNAV = %f, want 2.9401", rt.PreviousNAV)
	}
	if rt.DailyChangePct != 0.8673 {
		t.Errorf("DailyChangePct = %f, want 0.8673", rt.DailyChangePct)
	}
	if rt.NAVDate != "2026-07-21" {
		t.Errorf("NAVDate = %q, want 2026-07-21", rt.NAVDate)
	}
	if rt.UpdateTime != "2026-07-21 11:31:00" {
		t.Errorf("UpdateTime = %q, want '2026-07-21 11:31:00'", rt.UpdateTime)
	}
}

func TestParseSinaFundQuote_NegativeChange(t *testing.T) {
	rt, ok := ParseSinaFundQuote(sinaSampleNegative)
	if !ok {
		t.Fatal("expected line to match")
	}
	if !rt.Available {
		t.Fatal("expected Available=true")
	}
	if rt.DailyChangePct != -1.7473 {
		t.Errorf("DailyChangePct = %f, want -1.7473", rt.DailyChangePct)
	}
	if rt.EstimatedNAV != 1.6926 {
		t.Errorf("EstimatedNAV = %f, want 1.6926", rt.EstimatedNAV)
	}
}

func TestParseSinaFundQuote_EmptyPayload(t *testing.T) {
	rt, ok := ParseSinaFundQuote(sinaSampleEmpty)
	if !ok {
		t.Fatal("expected line to match")
	}
	if rt.Available {
		t.Error("expected Available=false for fund without estimate")
	}
	if rt.Code != "110037" {
		t.Errorf("Code = %q, want 110037", rt.Code)
	}
}

func TestParseSinaFundQuote_ExchangeTradedFund(t *testing.T) {
	rt, ok := ParseSinaFundQuote(sinaSampleETF)
	if !ok {
		t.Fatal("expected line to match")
	}
	if !rt.Available {
		t.Fatal("expected Available=true")
	}
	if rt.Code != "562500" {
		t.Errorf("Code = %q, want 562500", rt.Code)
	}
	if rt.Name != "华夏中证机器人ETF" {
		t.Errorf("Name = %q, want 华夏中证机器人ETF", rt.Name)
	}
}

func TestParseSinaFundQuote_MalformedLines(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"empty line", ""},
		{"stock quote", `var hq_str_sh600519="贵州茅台...";`},
		{"no semicolon", `var hq_str_fu_000217="data"`},
		{"random text", `v_pv_none_match="1";`},
		{"truncated", `var hq_str_fu_000217="华安黄金ETF联接C,11:31:00";`},
		{"too few fields", `var hq_str_fu_000217="a,b,c";`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt, ok := ParseSinaFundQuote(tc.line)
			switch tc.name {
			case "truncated", "too few fields":
				// Matches regex but < 8 fields → unavailable, not a parse failure
				if !ok || rt.Available {
					t.Error("expected (unavailable, true) for short payload")
				}
			default:
				if ok {
					t.Errorf("expected no match for %q", tc.line)
				}
			}
		})
	}
}

func TestFetchAllRealTime_BatchRequest(t *testing.T) {
	var requestCount int32
	response := sinaSampleNormal + "\n" + sinaSampleNegative + "\n" + sinaSampleEmpty + "\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		if r.Header.Get("Referer") == "" {
			t.Error("missing required Referer header")
		}

		fullURL := r.URL.Path + "?" + r.URL.RawQuery
		for _, want := range []string{"fu_000217", "fu_001595", "fu_110037"} {
			if !strings.Contains(fullURL, want) {
				t.Errorf("request missing symbol %s", want)
			}
		}

		// Encode as GBK like the real endpoint
		gbk, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(response))
		if err != nil {
			t.Fatalf("encode GBK: %v", err)
		}
		w.Write(gbk)
	}))
	defer server.Close()

	client := New(5)
	funds := client.fetchAllRealTime(server.URL+"/list=%s", []string{"000217", "001595", "110037"})

	if n := atomic.LoadInt32(&requestCount); n != 1 {
		t.Errorf("expected exactly 1 batch request, got %d", n)
	}
	if len(funds) != 3 {
		t.Fatalf("expected 3 results, got %d", len(funds))
	}
	if !funds["000217"].Available || funds["000217"].EstimatedNAV != 2.9656 {
		t.Errorf("000217: got %+v", funds["000217"])
	}
	if !funds["001595"].Available || funds["001595"].DailyChangePct != -1.7473 {
		t.Errorf("001595: got %+v", funds["001595"])
	}
	if funds["110037"].Available {
		t.Error("110037 should be Available=false (bond fund, no estimate)")
	}
}

func TestFetchAllRealTime_MissingCodesMarkedUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, sinaSampleNormal+"\n")
	}))
	defer server.Close()

	client := New(5)
	funds := client.fetchAllRealTime(server.URL+"/list=%s", []string{"000217", "999999"})

	if len(funds) != 2 {
		t.Fatalf("expected 2 results, got %d", len(funds))
	}
	if !funds["000217"].Available {
		t.Error("000217 should be available")
	}
	if funds["999999"] == nil || funds["999999"].Available {
		t.Error("999999 should exist with Available=false")
	}
}

func TestFetchAllRealTime_ServerError_AllUnavailable(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(5)
	funds := client.fetchAllRealTime(server.URL+"/list=%s", []string{"000217", "001595"})

	// 1 initial + 3 retries
	if n := atomic.LoadInt32(&requestCount); n != 4 {
		t.Errorf("expected 4 attempts (1+3 retries), got %d", n)
	}
	for _, code := range []string{"000217", "001595"} {
		if funds[code] == nil || funds[code].Available {
			t.Errorf("%s should be Available=false after server error", code)
		}
	}
}

func TestFetchAllRealTime_EmptyCodes(t *testing.T) {
	client := New(5)
	funds := client.FetchAllRealTime(nil)
	if len(funds) != 0 {
		t.Errorf("expected empty map, got %d entries", len(funds))
	}
}
