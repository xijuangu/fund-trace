package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fund-trace/internal/model"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// Tencent 财经行情接口:
// http://qt.gtimg.cn/q=sh600519,sz000001
// 返回: v_sh600519="1~贵州茅台~600519~1410.01~1411.00~..."

const tencentQuoteURL = "http://qt.gtimg.cn/q=%s"

// FetchStockQuotes fetches real-time quotes for multiple A-share stocks in one batch request.
func (c *Client) FetchStockQuotes(symbols []string) (map[string]*model.Quote, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	url := fmt.Sprintf(tencentQuoteURL, strings.Join(symbols, ","))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("stock quote request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://finance.qq.com/")

	resp, err := c.DoWithRetry(req, 3)
	if err != nil {
		return nil, fmt.Errorf("stock quote fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stock quote read: %w", err)
	}

	dec := simplifiedchinese.GBK.NewDecoder()
	utf8Body, err := dec.Bytes(body)
	if err != nil {
		utf8Body = body
	}
	raw := string(utf8Body)

	now := time.Now()
	result := make(map[string]*model.Quote)
	for _, sym := range symbols {
		q := ParseTencentQuote(raw, sym, now)
		result[sym] = q
	}
	return result, nil
}

// ParseTencentQuote parses the Tencent raw response for one symbol.
// symbol should be in "sh600519" or "sz000001" format.
// capturedAt is used as fallback UpdateTime.
func ParseTencentQuote(raw, symbol string, capturedAt time.Time) *model.Quote {
	prefix := "v_" + symbol + "=\""
	start := strings.Index(raw, prefix)
	if start < 0 {
		return &model.Quote{
			Kind:      model.AssetKindStock,
			Market:    symbol[:2],
			Code:      symbol[2:],
			UpdateTime: capturedAt.Format("15:04:05"),
			Available: false,
		}
	}

	innerStart := start + len(prefix)
	end := strings.Index(raw[innerStart:], "\"")
	if end < 0 {
		return &model.Quote{
			Kind:      model.AssetKindStock,
			Market:    symbol[:2],
			Code:      symbol[2:],
			UpdateTime: capturedAt.Format("15:04:05"),
			Available: false,
		}
	}

	inner := raw[innerStart : innerStart+end]
	fields := strings.Split(inner, "~")
	if len(fields) < 30 {
		return &model.Quote{
			Kind:      model.AssetKindStock,
			Market:    symbol[:2],
			Code:      symbol[2:],
			UpdateTime: capturedAt.Format("15:04:05"),
			Available: false,
		}
	}

	name := fields[1]
	currentPrice := parseFloatSafe(fields[3])
	previousClose := parseFloatSafe(fields[4])

	if currentPrice <= 0 || previousClose <= 0 {
		return &model.Quote{
			Kind:       model.AssetKindStock,
			Market:     symbol[:2],
			Code:       symbol[2:],
			Name:       name,
			Value:      currentPrice,
			Previous:   previousClose,
			UpdateTime: capturedAt.Format("15:04:05"),
			Available:  false,
		}
	}

	// Always compute from prices — field layout differs between A-shares and
	// HK stocks (different number of order-book levels), so the indices for
	// change% and timestamp are not stable.  Deriving from known fields 3/4 is safer.
	changePct := (currentPrice - previousClose) / previousClose * 100

	// Scan for the timestamp field: A-shares use "YYYYMMDDHHMMSS" (len 14+),
	// HK uses "YYYY/MM/DD HH:MM:SS" (len 19).  Check a few candidate positions
	// first (30 for HK, 33 for A-share) then fall back to a wider scan.
	updateTime := findTimestamp(fields)

	switch {
	case updateTime == "":
		updateTime = capturedAt.Format("15:04:05")
	case strings.Contains(updateTime, " "):
		updateTime = updateTime[len(updateTime)-8:]
	case len(updateTime) >= 14:
		updateTime = updateTime[8:10] + ":" + updateTime[10:12] + ":" + updateTime[12:14]
	}

	return &model.Quote{
		Kind:       model.AssetKindStock,
		Market:     symbol[:2],
		Code:       symbol[2:],
		Name:       name,
		Value:      currentPrice,
		Previous:   previousClose,
		ChangePct:  changePct,
		UpdateTime: updateTime,
		Available:  true,
	}
}

// findTimestamp scans fields for a timestamp value.
// A-share: "20260522145015" (len ≥ 14, all digits).
// HK:      "2026/05/22 16:08:22" (len 19, contains "/" and " ").
func findTimestamp(fields []string) string {
	// Check common positions first: 30 (A-share with empty fields 30-31),
	// 29 (HK with 20 order-book levels).
	for _, i := range []int{30, 29, 33} {
		if i < len(fields) && isTimestamp(fields[i]) {
			return fields[i]
		}
	}
	// Wider scan for non-standard layouts.
	scanLimit := 35
	if scanLimit > len(fields) {
		scanLimit = len(fields)
	}
	for _, f := range fields[:scanLimit] {
		if isTimestamp(f) {
			return f
		}
	}
	return ""
}

func isTimestamp(s string) bool {
	if len(s) < 14 || len(s) > 19 {
		return false
	}
	// A-share: 14+ digits, e.g. "20260522145015"
	if len(s) >= 14 {
		allDigit := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			return true
		}
	}
	// HK: "YYYY/MM/DD HH:MM:SS" (len 19, "/" at 4 and 7, " " at 10, ":" at 13 and 16)
	if len(s) == 19 && s[4] == '/' && s[7] == '/' && s[10] == ' ' && s[13] == ':' && s[16] == ':' {
		return true
	}
	return false
}
