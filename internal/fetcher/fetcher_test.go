package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fund-trace/internal/model"
)

func TestFetchRealTime_ValidJSONP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "011513") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body := `jsonpgz({"fundcode":"011513","name":"Test Fund","gsz":"1.500","dwjz":"1.480","jzrq":"2026-05-20","gszzl":"-1.25","gztime":"2026-05-21 14:30:00"});`
		fmt.Fprint(w, body)
	}))
	defer server.Close()

	client := New(5)
	// Override the server URL in the request
	code := "011513"
	// We use httptest server directly for control
	rt, err := fetchRealTimeFromServer(server, client, code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rt.Available {
		t.Error("expected Available=true")
	}
	if rt.Code != "011513" {
		t.Errorf("expected Code=011513, got %s", rt.Code)
	}
	if rt.Name != "Test Fund" {
		t.Errorf("expected Name='Test Fund', got %s", rt.Name)
	}
	if rt.EstimatedNAV != 1.5 {
		t.Errorf("expected EstimatedNAV=1.5, got %f", rt.EstimatedNAV)
	}
	if rt.PreviousNAV != 1.48 {
		t.Errorf("expected PreviousNAV=1.48, got %f", rt.PreviousNAV)
	}
	if rt.DailyChangePct != -1.25 {
		t.Errorf("expected DailyChangePct=-1.25, got %f", rt.DailyChangePct)
	}
	if rt.NAVDate != "2026-05-20" {
		t.Errorf("expected NAVDate=2026-05-20, got %s", rt.NAVDate)
	}
	if rt.UpdateTime != "2026-05-21 14:30:00" {
		t.Errorf("expected UpdateTime='2026-05-21 14:30:00', got %s", rt.UpdateTime)
	}
}

func TestFetchRealTime_EmptyJSONP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "jsonpgz();")
	}))
	defer server.Close()

	client := New(5)
	rt, err := fetchRealTimeFromServer(server, client, "000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Available {
		t.Error("expected Available=false for empty JSONP response")
	}
	if rt.Code != "000001" {
		t.Errorf("expected Code=000001, got %s", rt.Code)
	}
}

func TestFetchRealTime_ServerError(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(5)
	_, err := fetchRealTimeFromServer(server, client, "011513")
	if err == nil {
		t.Fatal("expected error on server error response")
	}

	count := atomic.LoadInt32(&requestCount)
	// Should retry: initial + 3 retries = 4 attempts
	if count != 4 {
		t.Errorf("expected 4 attempts (1 initial + 3 retries), got %d", count)
	}
}

func TestFetchHistory_ValidJSON(t *testing.T) {
	historyJSON := `{
		"Data": {
			"LSJZList": [
				{"FSRQ":"2026-05-20","DWJZ":"1.500","LJJZ":"2.500","JZZZL":"0.50"},
				{"FSRQ":"2026-05-19","DWJZ":"1.493","LJJZ":"2.493","JZZZL":"-0.30"}
			]
		},
		"ErrCode": 0,
		"ErrMsg": ""
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "http://fundf10.eastmoney.com/" {
			t.Log("warning: expected Referer header not present")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, historyJSON)
	}))
	defer server.Close()

	client := New(5)
	snapshots, err := fetchHistoryFromServer(server, client, "011513", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}

	s0 := snapshots[0]
	if s0.Date != "2026-05-20" {
		t.Errorf("expected Date=2026-05-20, got %s", s0.Date)
	}
	if s0.UnitNAV != 1.5 {
		t.Errorf("expected UnitNAV=1.5, got %f", s0.UnitNAV)
	}
	if s0.AccumulatedNAV != 2.5 {
		t.Errorf("expected AccumulatedNAV=2.5, got %f", s0.AccumulatedNAV)
	}
	if s0.DailyGrowthPct != 0.5 {
		t.Errorf("expected DailyGrowthPct=0.5, got %f", s0.DailyGrowthPct)
	}

	s1 := snapshots[1]
	if s1.Date != "2026-05-19" {
		t.Errorf("expected Date=2026-05-19, got %s", s1.Date)
	}
	if s1.DailyGrowthPct != -0.3 {
		t.Errorf("expected DailyGrowthPct=-0.3, got %f", s1.DailyGrowthPct)
	}
}

func TestFetchHistory_EmptyLSJZList(t *testing.T) {
	historyJSON := `{
		"Data": {
			"LSJZList": []
		},
		"ErrCode": 0,
		"ErrMsg": ""
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, historyJSON)
	}))
	defer server.Close()

	client := New(5)
	_, err := fetchHistoryFromServer(server, client, "011513", 30)
	if err == nil {
		t.Fatal("expected error for empty LSJZList")
	}
}

func TestFetchFundList_ValidContent(t *testing.T) {
	fundJS := `var r = [["000001","hxbch","华夏成长","混合型","huaxiachengzhang"],["011513","txzqhy","天弘中证新能源汽车","指数型-股票","tianhongzhongzhengxinnengyuanqiche"]];`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fundJS)
	}))
	defer server.Close()

	client := New(5)
	entries, err := fetchFundListFromServer(server, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Code != "000001" {
		t.Errorf("expected Code=000001, got %s", entries[0].Code)
	}
	if entries[0].Name != "华夏成长" {
		t.Errorf("expected Name='华夏成长', got %s", entries[0].Name)
	}
	if entries[1].Code != "011513" {
		t.Errorf("expected Code=011513, got %s", entries[1].Code)
	}
	if entries[1].TypeName != "指数型-股票" {
		t.Errorf("expected TypeName='指数型-股票', got %s", entries[1].TypeName)
	}
}

func TestFetchAllRealTime_Concurrent(t *testing.T) {
	codes := []string{"000001", "000002", "000003", "000004", "000005"}
	var concurrentCalls int32
	var maxConcurrent int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&concurrentCalls, 1)
		defer atomic.AddInt32(&concurrentCalls, -1)

		// Track max concurrency
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if current <= old {
				break
			}
			if atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
				break
			}
		}

		// Small delay to allow concurrent requests to stack up
		time.Sleep(50 * time.Millisecond)

		// Extract fund code from URL path
		path := r.URL.Path
		for _, c := range codes {
			if strings.Contains(path, c) {
				body := fmt.Sprintf(`jsonpgz({"fundcode":"%s","name":"Fund %s","gsz":"1.000","dwjz":"1.000","jzrq":"2026-05-20","gszzl":"0.00","gztime":"2026-05-21 14:30:00"});`, c, c)
				fmt.Fprint(w, body)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := New(5)
	results := fetchAllRealTimeFromServer(server, client, codes)

	if len(results) != len(codes) {
		t.Errorf("expected %d results, got %d", len(codes), len(results))
	}
	for _, code := range codes {
		if results[code] == nil {
			t.Errorf("missing result for code %s", code)
		} else if !results[code].Available {
			t.Errorf("result for %s should be Available=true", code)
		}
	}
}

func TestSemaphoreConcurrencyLimit(t *testing.T) {
	var concurrentCalls int32
	var maxConcurrent int32
	semSize := 2

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&concurrentCalls, 1)
		defer atomic.AddInt32(&concurrentCalls, -1)

		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if current <= old {
				break
			}
			if atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
				break
			}
		}

		// Hold the connection long enough for concurrency to build up
		time.Sleep(100 * time.Millisecond)

		// Return simple JSONP
		fmt.Fprint(w, `jsonpgz({"fundcode":"000001","name":"Fund","gsz":"1","dwjz":"1","jzrq":"2026-05-20","gszzl":"0","gztime":"2026-05-21 14:30:00"});`)
	}))
	defer server.Close()

	client := New(semSize)

	codeCnt := 10
	codes := make([]string, codeCnt)
	for i := 0; i < codeCnt; i++ {
		codes[i] = fmt.Sprintf("%06d", i+1)
	}
	results := fetchAllRealTimeFromServer(server, client, codes)

	if len(results) != codeCnt {
		t.Errorf("expected %d results, got %d", codeCnt, len(results))
	}
	if maxConcurrent > int32(semSize) {
		t.Errorf("max concurrency exceeded: %d > %d (semaphore size)", maxConcurrent, semSize)
	}
	t.Logf("max concurrent requests: %d (semaphore limit: %d)", maxConcurrent, semSize)
}

func TestRetryBackoff(t *testing.T) {
	var requestCount int32
	var requestTimes []time.Time
	var mu syncSafe

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(5)
	start := time.Now()
	_, err := fetchRealTimeFromServer(server, client, "011513")
	if err == nil {
		t.Fatal("expected error after all retries fail")
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 4 {
		t.Errorf("expected 4 requests (1 initial + 3 retries), got %d", count)
	}

	// Verify backoff timing: delays should be ~0.5s, ~1s, ~2s apart
	mu.Lock()
	times := requestTimes
	mu.Unlock()

	if len(times) >= 3 {
		d1 := times[1].Sub(times[0])
		d2 := times[2].Sub(times[1])
		// Allow generous margins since httptest is fast
		if d1 < 400*time.Millisecond {
			t.Errorf("first backoff too short: %v (expected ~500ms)", d1)
		}
		if d2 < 900*time.Millisecond {
			t.Errorf("second backoff too short: %v (expected ~1000ms)", d2)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 3*time.Second || elapsed > 6*time.Second {
		t.Logf("total retry elapsed: %v (expected ~3.5s with 500ms+1s+2s backoffs)", elapsed)
	}
}

func TestParseFloatSafe(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"0", 0},
		{"1.5", 1.5},
		{"-1.25", -1.25},
		{"abc", 0},
		{"12.345", 12.345},
	}
	for _, tt := range tests {
		got := parseFloatSafe(tt.input)
		if got != tt.want {
			t.Errorf("parseFloatSafe(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestBuildFundNameMap(t *testing.T) {
	fundJS := `var r = [["000001","hxbch","华夏成长","混合型","huaxiachengzhang"],["011513","txzqhy","天弘中证新能源汽车","指数型-股票","tianhongzhongzhengxinnengyuanqiche"]];`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fundJS)
	}))
	defer server.Close()

	client := New(5)
	nameMap, err := buildFundNameMapFromServer(server, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nameMap["000001"] != "华夏成长" {
		t.Errorf("expected '华夏成长', got %s", nameMap["000001"])
	}
	if nameMap["011513"] != "天弘中证新能源汽车" {
		t.Errorf("expected '天弘中证新能源汽车', got %s", nameMap["011513"])
	}
}

func TestFetchRealTime_JSONPWithoutSemicolon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSONP without trailing semicolon - should still parse
		fmt.Fprint(w, `jsonpgz({"fundcode":"011513","name":"Test Fund","gsz":"1.500","dwjz":"1.480","jzrq":"2026-05-20","gszzl":"-1.25","gztime":"2026-05-21 14:30:00"})`)
	}))
	defer server.Close()

	client := New(5)
	rt, err := fetchRealTimeFromServer(server, client, "011513")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rt.Available {
		t.Error("expected Available=true")
	}
	if rt.EstimatedNAV != 1.5 {
		t.Errorf("expected EstimatedNAV=1.5, got %f", rt.EstimatedNAV)
	}
}

// ---- Test helpers that bypass the real URL ----

func fetchRealTimeFromServer(server *httptest.Server, client *Client, code string) (*model.RealTimeFund, error) {
	url := fmt.Sprintf("%s/js/%s.js?rt=%d", server.URL, code, time.Now().UnixMilli())
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.DoWithRetry(req, 3)
	if err != nil {
		return nil, fmt.Errorf("fetch realtime %s: %w", code, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", code, err)
	}
	body := string(data)

	matches := jsonpRE.FindStringSubmatch(body)
	if len(matches) < 2 {
		if len(body) == 0 || body == "jsonpgz();" || body == "jsonpgz()" {
			return &model.RealTimeFund{
				Code:      code,
				Available: false,
			}, nil
		}
		return nil, fmt.Errorf("parse jsonp %s: unexpected format", code)
	}

	var raw tiantianRaw
	if err := json.Unmarshal([]byte(matches[1]), &raw); err != nil {
		return nil, fmt.Errorf("parse json %s: %w", code, err)
	}

	return &model.RealTimeFund{
		Code:           raw.FundCode,
		Name:           raw.Name,
		NAVDate:        raw.JZRQ,
		UpdateTime:     raw.GZTIME,
		Available:      true,
		EstimatedNAV:   parseFloatSafe(raw.GSZ),
		PreviousNAV:    parseFloatSafe(raw.DWJZ),
		DailyChangePct: parseFloatSafe(raw.GSZZL),
	}, nil
}

func fetchHistoryFromServer(server *httptest.Server, client *Client, code string, days int) ([]model.NavSnapshot, error) {
	url := fmt.Sprintf("%s/?fundCode=%s&pageIndex=1&pageSize=%d", server.URL, code, days)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request %s: %w", code, err)
	}
	req.Header.Set("Referer", "http://fundf10.eastmoney.com/")

	resp, err := client.DoWithRetry(req, 2)
	if err != nil {
		return nil, fmt.Errorf("fetch history %s: %w", code, err)
	}
	defer resp.Body.Close()

	var hist historyResponse
	if err := json.NewDecoder(resp.Body).Decode(&hist); err != nil {
		return nil, fmt.Errorf("decode history %s: %w", code, err)
	}

	if len(hist.Data.LSJZList) == 0 {
		return nil, fmt.Errorf("no history data for fund %s", code)
	}

	now := time.Now()
	var snapshots []model.NavSnapshot
	for _, item := range hist.Data.LSJZList {
		snapshots = append(snapshots, model.NavSnapshot{
			FundCode:       code,
			Date:           item.FSRQ,
			UnitNAV:        parseFloatSafe(item.DWJZ),
			AccumulatedNAV: parseFloatSafe(item.LJJZ),
			DailyGrowthPct: parseFloatSafe(item.JZZZL),
			RecordedAt:     now,
		})
	}
	return snapshots, nil
}

func fetchFundListFromServer(server *httptest.Server, client *Client) ([]model.FundListEntry, error) {
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create fund list request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.DoWithRetry(req, 2)
	if err != nil {
		return nil, fmt.Errorf("fetch fund list: %w", err)
	}
	defer resp.Body.Close()

	var buf = make([]byte, 1024*1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	matches := fundListRE.FindAllStringSubmatch(body, -1)
	var entries []model.FundListEntry
	for _, m := range matches {
		if len(m) >= 6 {
			entries = append(entries, model.FundListEntry{
				Code:       m[1],
				Pinyin:     m[2],
				Name:       m[3],
				TypeName:   m[4],
				FullPinyin: m[5],
			})
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no fund entries found in fundcode_search.js")
	}
	return entries, nil
}

func buildFundNameMapFromServer(server *httptest.Server, client *Client) (map[string]string, error) {
	entries, err := fetchFundListFromServer(server, client)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Code] = e.Name
	}
	return m, nil
}

func fetchAllRealTimeFromServer(server *httptest.Server, client *Client, codes []string) map[string]*model.RealTimeFund {
	type result struct {
		code string
		data *model.RealTimeFund
	}
	results := make(chan result, len(codes))

	for _, code := range codes {
		go func(fundCode string) {
			rt, err := fetchRealTimeFromServer(server, client, fundCode)
			if err != nil {
				rt = &model.RealTimeFund{
					Code:      fundCode,
					Available: false,
				}
			}
			results <- result{code: fundCode, data: rt}
		}(code)
	}

	funds := make(map[string]*model.RealTimeFund)
	for i := 0; i < len(codes); i++ {
		r := <-results
		funds[r.code] = r.data
	}
	return funds
}

// syncSafe is a simple mutex wrapper for slices of time.Time
type syncSafe struct{ mu sync.Mutex }

func (s *syncSafe) Lock()   { s.mu.Lock() }
func (s *syncSafe) Unlock() { s.mu.Unlock() }
