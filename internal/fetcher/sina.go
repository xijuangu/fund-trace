package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"fund-trace/internal/model"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// 新浪财经基金盘中估值接口:
// https://hq.sinajs.cn/list=fu_000217,fu_001595
// 返回(GBK 编码，每只基金一行):
//
//	var hq_str_fu_000217="华安黄金ETF联接C,11:31:00,2.9656,2.9401,2.9401,0.0101,0.8673,2026-07-21,2.9691,0.9864";
//
// 字段: 名称,更新时间,估算净值,昨日净值,累计净值,涨跌额,涨跌幅%,日期,估算净值(指数口径),涨跌幅%(指数口径)
// 无估值数据的基金(纯债、货币等)返回空字符串: var hq_str_fu_110037="";
//
// 注意: 第 3/7 字段与第 9/10 字段是两套独立更新的估值，盘中可能一套已经
// 刷新、另一套还停在旧值（表现为涨跌幅在 0.0x% 与真实值之间跳变）。因此
// 第 10 字段只作为交叉校验的辅助值，最终展示值还要经过指数锚定校验。
//
// 注: 原天天基金估值接口 fundgz.1234567.com.cn 已于 2026 年 7 月下线。
const sinaFundQuoteURL = "https://hq.sinajs.cn/list=%s"

var sinaLineRE = regexp.MustCompile(`^var hq_str_fu_(\d{6})="(.*)";$`)

// ParseSinaFundQuote parses a single line from the Sina fund quote response.
// Returns (nil, false) if the line does not match the expected format.
// Returns (&RealTimeFund{Available: false}, true) if the fund has no estimate
// data (e.g., bond funds and money-market funds never publish intraday estimates).
func ParseSinaFundQuote(line string) (*model.RealTimeFund, bool) {
	m := sinaLineRE.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return nil, false
	}

	code := m[1]
	data := m[2]

	// Empty payload means the fund has no intraday estimate.
	if data == "" {
		return &model.RealTimeFund{Code: code, Available: false}, true
	}

	fields := strings.Split(data, ",")
	if len(fields) < 8 {
		return &model.RealTimeFund{Code: code, Available: false}, true
	}

	// The second valuation pair (fields 9/10) is optional: older payloads and
	// some fund types only carry the first pair.
	var altChange float64
	if len(fields) >= 10 {
		altChange = parseFloatSafe(fields[9])
	}

	return &model.RealTimeFund{
		Code:           code,
		Name:           fields[0],
		UpdateTime:     fields[7] + " " + fields[1],
		EstimatedNAV:   parseFloatSafe(fields[2]),
		PreviousNAV:    parseFloatSafe(fields[3]),
		DailyChangePct: parseFloatSafe(fields[6]),
		NAVDate:        fields[7],
		AltChangePct:   altChange,
		Available:      true,
	}, true
}

// FetchAllRealTime fetches real-time fund NAV estimates from Sina Finance
// in a single batch request. Funds without estimate data (bond/money-market
// funds) are returned with Available=false.
func (c *Client) FetchAllRealTime(codes []string) map[string]*model.RealTimeFund {
	return c.fetchAllRealTime(sinaFundQuoteURL, codes)
}

func (c *Client) fetchAllRealTime(urlFormat string, codes []string) map[string]*model.RealTimeFund {
	funds := make(map[string]*model.RealTimeFund, len(codes))
	if len(codes) == 0 {
		return funds
	}

	symbols := make([]string, len(codes))
	for i, code := range codes {
		symbols[i] = "fu_" + code
	}

	url := fmt.Sprintf(urlFormat, strings.Join(symbols, ","))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.markAllUnavailable(funds, codes)
		return funds
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://finance.sina.com.cn/")

	resp, err := c.DoWithRetry(req, 3)
	if err != nil {
		c.logger.Warn("sina fund quote fetch failed", "error", err)
		c.markAllUnavailable(funds, codes)
		return funds
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("sina fund quote read failed", "error", err)
		c.markAllUnavailable(funds, codes)
		return funds
	}

	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil {
		decoded = raw // fallback: treat as UTF-8
	}

	for _, line := range strings.Split(string(decoded), "\n") {
		rt, ok := ParseSinaFundQuote(line)
		if ok {
			funds[rt.Code] = rt
		}
	}

	// Codes missing from the response are marked unavailable.
	for _, code := range codes {
		if _, exists := funds[code]; !exists {
			funds[code] = &model.RealTimeFund{Code: code, Available: false}
		}
	}
	return funds
}

func (c *Client) markAllUnavailable(funds map[string]*model.RealTimeFund, codes []string) {
	for _, code := range codes {
		funds[code] = &model.RealTimeFund{Code: code, Available: false}
	}
}
