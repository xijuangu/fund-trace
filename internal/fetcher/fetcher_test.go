package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fund-trace/internal/model"
)

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

// ---- Test helpers that bypass the real URL ----

func fetchHistoryFromServer(server *httptest.Server, client *Client, code string, days int) ([]model.NavSnapshot, error) {
	now := time.Now()
	var allSnapshots []model.NavSnapshot

	for pageIndex := 1; len(allSnapshots) < days; pageIndex++ {
		url := fmt.Sprintf("%s/?fundCode=%s&pageIndex=%d&pageSize=20", server.URL, code, pageIndex)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request %s: %w", code, err)
		}
		req.Header.Set("Referer", "http://fundf10.eastmoney.com/")

		resp, err := client.DoWithRetry(req, 2)
		if err != nil {
			return nil, fmt.Errorf("fetch history %s: %w", code, err)
		}

		var hist historyResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&hist)
		resp.Body.Close()

		if decodeErr != nil {
			return nil, fmt.Errorf("decode history %s page %d: %w", code, pageIndex, decodeErr)
		}

		for _, item := range hist.Data.LSJZList {
			allSnapshots = append(allSnapshots, model.NavSnapshot{
				FundCode:       code,
				Date:           item.FSRQ,
				UnitNAV:        parseFloatSafe(item.DWJZ),
				AccumulatedNAV: parseFloatSafe(item.LJJZ),
				DailyGrowthPct: parseFloatSafe(item.JZZZL),
				RecordedAt:     now,
			})
		}

		if len(hist.Data.LSJZList) < 20 {
			break
		}
	}

	if len(allSnapshots) == 0 {
		return nil, fmt.Errorf("no history data for fund %s", code)
	}

	return allSnapshots, nil
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
