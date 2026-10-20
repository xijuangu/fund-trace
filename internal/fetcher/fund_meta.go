package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"fund-trace/internal/model"
)

// East Money fund detail: provides the benchmark index code and the tracking
// index weight (e.g. "中证红利低波动指数收益率*95%+活期存款利率(税后)*5%").
const fundMetaURL = "https://fundmobapi.eastmoney.com/FundMNewApi/FundMNDetailInformation" +
	"?FCODE=%s&deviceid=1&plat=Android&product=EFund&version=1"

// East Money quotation search: resolves an index code to its secid
// (e.g. H30269 -> "2.H30269", 980030 -> "0.980030").
const indexSuggestURL = "https://searchapi.eastmoney.com/api/suggest/get" +
	"?input=%s&type=14&token=D43BF722C8E33BDC906FB84D85E326E8&count=10"

// DefaultIndexWeight is used when the benchmark string does not expose a
// weight. Index feeder funds hold ~95% of the target index plus cash.
const DefaultIndexWeight = 0.95

var benchWeightRE = regexp.MustCompile(`\*(\d{1,3}(?:\.\d+)?)\s*%`)

type fundMetaResponse struct {
	Datas struct {
		IndexCode string `json:"INDEXCODE"`
		IndexName string `json:"INDEXNAME"`
		Bench     string `json:"BENCH"`
		FType     string `json:"FTYPE"`
	} `json:"Datas"`
	ErrCode int    `json:"ErrCode"`
	Success bool   `json:"Success"`
	ErrMsg  string `json:"ErrMsg"`
}

type suggestResponse struct {
	QuotationCodeTable struct {
		Data []struct {
			Code             string `json:"Code"`
			QuoteID          string `json:"QuoteID"`
			SecurityTypeName string `json:"SecurityTypeName"`
		} `json:"Data"`
	} `json:"QuotationCodeTable"`
}

// FetchFundMeta looks up a fund's benchmark index and the weight it assigns to
// that index. Funds without a benchmark index (active funds) return a meta with
// an empty IndexCode.
func (c *Client) FetchFundMeta(code string) (*model.FundMeta, error) {
	return c.fetchFundMeta(fundMetaURL, indexSuggestURL, code)
}

// East Money's mobile API is picky about request headers: a desktop browser
// UA or an eastmoney Referer makes it answer "网络繁忙，请稍后重试"
// (ErrCode 61136403). A bare Mozilla/5.0 agent with no Referer works.
const mobileAPIUserAgent = "Mozilla/5.0"

func (c *Client) fetchFundMeta(metaURLFmt, suggestURLFmt, code string) (*model.FundMeta, error) {
	meta := &model.FundMeta{Code: code, IndexWeight: DefaultIndexWeight}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(metaURLFmt, code), nil)
	if err != nil {
		return nil, fmt.Errorf("create fund meta request %s: %w", code, err)
	}
	req.Header.Set("User-Agent", mobileAPIUserAgent)

	resp, err := c.DoWithRetry(req, 2)
	if err != nil {
		return nil, fmt.Errorf("fetch fund meta %s: %w", code, err)
	}
	defer resp.Body.Close()

	var payload fundMetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode fund meta %s: %w", code, err)
	}
	if payload.ErrCode != 0 {
		return nil, fmt.Errorf("fund meta %s: code %d: %s", code, payload.ErrCode, payload.ErrMsg)
	}

	indexCode := strings.TrimSpace(payload.Datas.IndexCode)
	if indexCode == "" || indexCode == "--" {
		return meta, nil // active fund: no index anchor available
	}
	meta.IndexCode = indexCode
	if weight := parseBenchWeight(payload.Datas.Bench); weight > 0 {
		meta.IndexWeight = weight
	}
	meta.IndexSecID = c.resolveIndexSecID(suggestURLFmt, indexCode)
	meta.DeriveOK = deriveOK(payload.Datas.FType)
	return meta, nil
}

// parseBenchWeight extracts the benchmark index weight from a benchmark string
// such as "中证红利低波动指数收益率*95%+活期存款利率(税后)*5%". The first
// weight is the index weight; the remainder is the cash component.
func parseBenchWeight(bench string) float64 {
	m := benchWeightRE.FindStringSubmatch(bench)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil || v <= 0 || v > 100 {
		return 0
	}
	return v / 100
}

// deriveOK reports whether the index-anchored estimate may override the
// upstream value. QDII funds ("指数型-海外股票") settle NAV with FX and a
// one-day lag; commodity funds ("指数型-其他") track non-equity underlyings.
// For both, the index move is only a loose proxy, so they are left untouched.
func deriveOK(fundType string) bool {
	return strings.Contains(fundType, "指数型-股票")
}

func (c *Client) resolveIndexSecID(suggestURLFmt, indexCode string) string {
	if secid, err := c.lookupIndexSecID(suggestURLFmt, indexCode); err == nil && secid != "" {
		return secid
	}
	return guessIndexSecID(indexCode)
}

func (c *Client) lookupIndexSecID(suggestURLFmt, indexCode string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(suggestURLFmt, indexCode), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := c.DoWithRetry(req, 1)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload suggestResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	for _, row := range payload.QuotationCodeTable.Data {
		if !strings.EqualFold(row.Code, indexCode) || row.QuoteID == "" {
			continue
		}
		// The same code can resolve to a stock, a fund and an index
		// (e.g. 000001 is both 平安银行 and 上证指数); keep only indices
		// and spot commodities.
		switch row.SecurityTypeName {
		case "指数", "现货":
			return row.QuoteID, nil
		}
	}
	return "", fmt.Errorf("index %s not found", indexCode)
}

// guessIndexSecID is the offline fallback for resolveIndexSecID. It covers the
// index families used by Chinese public funds.
func guessIndexSecID(indexCode string) string {
	code := strings.ToUpper(strings.TrimSpace(indexCode))
	switch {
	case code == "":
		return ""
	case strings.HasPrefix(code, "AU"), strings.HasPrefix(code, "AG"):
		return "118." + code // Shanghai Gold Exchange
	case strings.HasPrefix(code, "HSTECH"), code == "HSI":
		return "124." + code // Hang Seng indices
	case strings.HasPrefix(code, "H"), strings.HasPrefix(code, "93"):
		return "2." + code // CSI indices
	case strings.HasPrefix(code, "399"), strings.HasPrefix(code, "98"):
		return "0." + code // Shenzhen / CNI indices
	case strings.HasPrefix(code, "000"):
		return "1." + code // Shanghai indices
	}
	return ""
}
