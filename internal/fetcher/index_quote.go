package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// East Money batch quote endpoint. One request can resolve many indices, which
// keeps the anchor validation to a single extra HTTP call per refresh cycle.
const indexQuoteURL = "https://push2.eastmoney.com/api/qt/ulist.np/get?secids=%s&fields=f2,f3,f12,f13,f14"

// indexQuoteBatch caps how many secids go into one request.
const indexQuoteBatch = 50

type indexQuoteResponse struct {
	Data struct {
		Diff []struct {
			Code   string    `json:"f12"`
			Market int       `json:"f13"`
			Change flexFloat `json:"f3"`
		} `json:"diff"`
	} `json:"data"`
}

// flexFloat accepts both JSON numbers and the "-" placeholder East Money uses
// for instruments without a quote.
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "-" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

// FetchIndexChanges returns the intraday change percent of each index keyed by
// its secid (e.g. "2.H30269"). The secid keys match what fund metadata stores.
func (c *Client) FetchIndexChanges(secids []string) (map[string]float64, error) {
	return c.fetchIndexChanges(indexQuoteURL, secids)
}

func (c *Client) fetchIndexChanges(urlFormat string, secids []string) (map[string]float64, error) {
	changes := make(map[string]float64, len(secids))
	unique := dedupeStrings(secids)
	if len(unique) == 0 {
		return changes, nil
	}

	for start := 0; start < len(unique); start += indexQuoteBatch {
		end := min(start+indexQuoteBatch, len(unique))
		url := fmt.Sprintf(urlFormat, strings.Join(unique[start:end], ","))

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return changes, fmt.Errorf("create index quote request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://quote.eastmoney.com/")

		// This endpoint occasionally drops connections without a response, so
		// retry more aggressively than a single extra attempt.
		resp, err := c.DoWithRetry(req, 2)
		if err != nil {
			return changes, fmt.Errorf("fetch index quotes: %w", err)
		}

		var payload indexQuoteResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if decodeErr != nil {
			return changes, fmt.Errorf("decode index quotes: %w", decodeErr)
		}

		for _, d := range payload.Data.Diff {
			changes[fmt.Sprintf("%d.%s", d.Market, d.Code)] = float64(d.Change) / 100
		}
	}
	return changes, nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
