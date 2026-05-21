package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"fund-trace/internal/model"
)

var jsonpRE = regexp.MustCompile(`^jsonpgz\((.+)\);?\s*$`)

// tiantianRaw is the raw JSONP response structure
type tiantianRaw struct {
	FundCode string `json:"fundcode"`
	Name     string `json:"name"`
	GSZ      string `json:"gsz"`    // estimated NAV
	DWJZ     string `json:"dwjz"`   // previous NAV
	JZRQ     string `json:"jzrq"`   // NAV date
	GSZZL    string `json:"gszzl"`  // daily change percent
	GZTIME   string `json:"gztime"` // update time
}

func (c *Client) FetchRealTime(code string) (*model.RealTimeFund, error) {
	timestamp := time.Now().UnixMilli()
	url := fmt.Sprintf("http://fundgz.1234567.com.cn/js/%s.js?rt=%d", code, timestamp)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch realtime %s: %w", code, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.DoWithRetry(req, 3)
	if err != nil {
		return nil, fmt.Errorf("fetch realtime %s: %w", code, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", code, err)
	}
	body := string(data)

	// Parse JSONP: jsonpgz({...});
	matches := jsonpRE.FindStringSubmatch(body)
	if len(matches) < 2 {
		// Empty response means data not available (e.g., non-index funds post-2022)
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

	rt := &model.RealTimeFund{
		Code:           raw.FundCode,
		Name:           raw.Name,
		NAVDate:        raw.JZRQ,
		UpdateTime:     raw.GZTIME,
		Available:      true,
		EstimatedNAV:   parseFloatSafe(raw.GSZ),
		PreviousNAV:    parseFloatSafe(raw.DWJZ),
		DailyChangePct: parseFloatSafe(raw.GSZZL),
	}
	return rt, nil
}

// FetchAllRealTime fetches real-time data for all codes concurrently
func (c *Client) FetchAllRealTime(codes []string) map[string]*model.RealTimeFund {
	type result struct {
		code string
		data *model.RealTimeFund
	}
	results := make(chan result, len(codes))

	for _, code := range codes {
		go func(fundCode string) {
			rt, err := c.FetchRealTime(fundCode)
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
