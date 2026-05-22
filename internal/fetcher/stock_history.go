package fetcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fund-trace/internal/model"
)

const eastMoneyKLineURL = "https://push2his.eastmoney.com/api/qt/stock/kline/get"

func (c *Client) FetchStockHistory(market, code string, days int) ([]model.PriceSnapshot, error) {
	if days <= 0 {
		days = 60
	}

	secid := "0." + code
	if market == "sh" {
		secid = "1." + code
	}

	url := fmt.Sprintf(
		"%s?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61&klt=101&fqt=1&end=20500101&lmt=%d",
		eastMoneyKLineURL, secid, days,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create kline request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := c.DoWithRetry(req, 2)
	if err != nil {
		return nil, fmt.Errorf("fetch kline %s:%s: %w", market, code, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read kline body %s:%s: %w", market, code, err)
	}

	snapshots, err := ParseEastMoneyKLine(body)
	if err != nil {
		return nil, fmt.Errorf("parse kline %s:%s: %w", market, code, err)
	}

	for i := range snapshots {
		snapshots[i].Market = market
		snapshots[i].Code = code
	}

	return snapshots, nil
}

type eastMoneyKLineResponse struct {
	Data *eastMoneyKLineData `json:"data"`
}

type eastMoneyKLineData struct {
	KLines []string `json:"klines"`
}

func ParseEastMoneyKLine(jsonBytes []byte) ([]model.PriceSnapshot, error) {
	var raw eastMoneyKLineResponse
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal kline json: %w", err)
	}

	if raw.Data == nil || len(raw.Data.KLines) == 0 {
		return []model.PriceSnapshot{}, nil
	}

	now := time.Now()
	var snapshots []model.PriceSnapshot

	for _, line := range raw.Data.KLines {
		fields := strings.Split(line, ",")
		if len(fields) < 9 {
			continue
		}

		snapshots = append(snapshots, model.PriceSnapshot{
			Kind:       model.AssetKindStock,
			Date:       fields[0],
			Open:       parseFloatSafe(fields[1]),
			Close:      parseFloatSafe(fields[2]),
			High:       parseFloatSafe(fields[3]),
			Low:        parseFloatSafe(fields[4]),
			Volume:     parseFloatSafe(fields[5]),
			Amount:     parseFloatSafe(fields[6]),
			ChangePct:  parseFloatSafe(fields[8]),
			RecordedAt: now,
		})
	}

	return snapshots, nil
}
