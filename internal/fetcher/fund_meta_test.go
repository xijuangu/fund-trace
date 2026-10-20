package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const fundMetaSample = `{"Datas":{
	"FCODE":"020602",
	"FTYPE":"指数型-股票",
	"INDEXCODE":"H30269",
	"INDEXNAME":"中证红利低波动指数",
	"BENCH":"中证红利低波动指数收益率*95%+活期存款利率(税后)*5%"
}}`

const fundMetaActiveSample = `{"Datas":{
	"FCODE":"016068",
	"FTYPE":"混合型-偏股",
	"INDEXCODE":"--",
	"INDEXNAME":"--",
	"BENCH":"中证新能源汽车指数收益率*85%+中证港股通汽车产业主题指数(人民币)收益率*5%"
}}`

const indexSuggestSample = `{"QuotationCodeTable":{"Data":[
	{"Code":"H30269","QuoteID":"2.H30269","SecurityTypeName":"指数"}
]}}`

// 000001 resolves to a stock first; the index entry must be picked.
const indexSuggestAmbiguousSample = `{"QuotationCodeTable":{"Data":[
	{"Code":"000001","QuoteID":"0.000001","SecurityTypeName":"深A"},
	{"Code":"000001","QuoteID":"1.000001","SecurityTypeName":"指数"}
]}}`

func newMetaTestServer(t *testing.T, metaJSON, suggestJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/meta", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, metaJSON)
	})
	mux.HandleFunc("/suggest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, suggestJSON)
	})
	return httptest.NewServer(mux)
}

func TestFetchFundMeta_IndexFund(t *testing.T) {
	server := newMetaTestServer(t, fundMetaSample, indexSuggestSample)
	defer server.Close()

	client := New(2)
	meta, err := client.fetchFundMeta(server.URL+"/meta?FCODE=%s", server.URL+"/suggest?input=%s", "020602")
	if err != nil {
		t.Fatalf("fetchFundMeta: %v", err)
	}
	if meta.IndexCode != "H30269" {
		t.Errorf("IndexCode = %q, want H30269", meta.IndexCode)
	}
	if meta.IndexSecID != "2.H30269" {
		t.Errorf("IndexSecID = %q, want 2.H30269", meta.IndexSecID)
	}
	if meta.IndexWeight != 0.95 {
		t.Errorf("IndexWeight = %v, want 0.95", meta.IndexWeight)
	}
	if !meta.DeriveOK {
		t.Error("DeriveOK = false, want true for a domestic equity index fund")
	}
}

func TestFetchFundMeta_ActiveFund(t *testing.T) {
	server := newMetaTestServer(t, fundMetaActiveSample, indexSuggestSample)
	defer server.Close()

	client := New(2)
	meta, err := client.fetchFundMeta(server.URL+"/meta?FCODE=%s", server.URL+"/suggest?input=%s", "016068")
	if err != nil {
		t.Fatalf("fetchFundMeta: %v", err)
	}
	if meta.IndexCode != "" {
		t.Errorf("IndexCode = %q, want empty for an active fund", meta.IndexCode)
	}
	if meta.IndexSecID != "" {
		t.Errorf("IndexSecID = %q, want empty", meta.IndexSecID)
	}
}

func TestFetchFundMeta_SuggestPicksIndexNotStock(t *testing.T) {
	server := newMetaTestServer(t, `{"Datas":{"FTYPE":"指数型-股票","INDEXCODE":"000001","BENCH":"上证指数收益率*95%+活期*5%"}}`,
		indexSuggestAmbiguousSample)
	defer server.Close()

	client := New(2)
	meta, err := client.fetchFundMeta(server.URL+"/meta?FCODE=%s", server.URL+"/suggest?input=%s", "018536")
	if err != nil {
		t.Fatalf("fetchFundMeta: %v", err)
	}
	if meta.IndexSecID != "1.000001" {
		t.Errorf("IndexSecID = %q, want 1.000001 (index, not 平安银行)", meta.IndexSecID)
	}
}

func TestParseBenchWeight(t *testing.T) {
	cases := []struct {
		bench string
		want  float64
	}{
		{"中证红利低波动指数收益率*95%+活期存款利率(税后)*5%", 0.95},
		{"恒生科技指数收益率(使用估值汇率折算)*95%+人民币活期存款收益率(税后)*5", 0.95},
		{"沪深300指数收益率*90%+银行活期存款利率(税后)*10%", 0.90},
		{"中证新能源汽车指数收益率*85%+中证港股通汽车产业主题指数(人民币)收益率*5", 0.85},
		{"", 0},
		{"no weight here", 0},
	}
	for _, tc := range cases {
		if got := parseBenchWeight(tc.bench); got != tc.want {
			t.Errorf("parseBenchWeight(%q) = %v, want %v", tc.bench, got, tc.want)
		}
	}
}

func TestDeriveOK(t *testing.T) {
	cases := map[string]bool{
		"指数型-股票":   true,
		"指数型-海外股票": false,
		"指数型-其他":   false,
		"混合型-偏股":   false,
	}
	for fundType, want := range cases {
		if got := deriveOK(fundType); got != want {
			t.Errorf("deriveOK(%q) = %v, want %v", fundType, got, want)
		}
	}
}

func TestGuessIndexSecID(t *testing.T) {
	cases := map[string]string{
		"H30269": "2.H30269",
		"931494": "2.931494",
		"980030": "0.980030",
		"399986": "0.399986",
		"000300": "1.000300",
		"HSTECH": "124.HSTECH",
		"AU9999": "118.AU9999",
		"":       "",
		"ABCDEF": "",
	}
	for code, want := range cases {
		if got := guessIndexSecID(code); got != want {
			t.Errorf("guessIndexSecID(%q) = %q, want %q", code, got, want)
		}
	}
}
