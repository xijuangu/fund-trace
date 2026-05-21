package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"fund-trace/internal/model"
)

type historyResponse struct {
	Data struct {
		LSJZList []struct {
			FSRQ  string `json:"FSRQ"`
			DWJZ  string `json:"DWJZ"`
			LJJZ  string `json:"LJJZ"`
			JZZZL string `json:"JZZZL"`
		} `json:"LSJZList"`
	} `json:"Data"`
	ErrCode int    `json:"ErrCode"`
	ErrMsg  string `json:"ErrMsg"`
}

func (c *Client) FetchHistory(code string, days int) ([]model.NavSnapshot, error) {
	if days <= 0 {
		days = 30
	}

	url := fmt.Sprintf(
		"http://api.fund.eastmoney.com/f10/lsjz?fundCode=%s&pageIndex=1&pageSize=%d",
		code, days,
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request %s: %w", code, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "http://fundf10.eastmoney.com/")

	resp, err := c.DoWithRetry(req, 2)
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

// fundcodeSearch response: var r = [["code","pinyin","name","type","full_pinyin"], ...]
var fundListRE = regexp.MustCompile(`"([^"]+)","([^"]+)","([^"]+)","([^"]+)","([^"]+)"`)

func (c *Client) FetchFundList() ([]model.FundListEntry, error) {
	url := "http://fund.eastmoney.com/js/fundcode_search.js"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create fund list request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.DoWithRetry(req, 2)
	if err != nil {
		return nil, fmt.Errorf("fetch fund list: %w", err)
	}
	defer resp.Body.Close()

	// Read full body as string for regex parsing
	var buf = make([]byte, 1024*1024) // 1MB should be enough for ~27k entries
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

// BuildFundNameMap builds a code→name lookup map
func (c *Client) BuildFundNameMap() (map[string]string, error) {
	entries, err := c.FetchFundList()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Code] = e.Name
	}
	return m, nil
}
