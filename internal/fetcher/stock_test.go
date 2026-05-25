package fetcher

import (
	"fund-trace/internal/model"
	"strings"
	"testing"
	"time"
)

// Real Tencent response sample for sh600519 (贵州茅台) and sz000001 (平安银行).
const tencentSampleResponse = `v_sh600519="1~贵州茅台~600519~1410.01~1411.00~1408.00~113928~4222~71716~1410.00~163~1408.00~50~1407.00~12~1406.00~27~1405.00~1~1410.01~128~1411.00~209~1412.00~63~1413.00~30~1414.00~2~1415.00~1~~~20260522113020~-0.99~-0.07~1410.49~1402.16~1410.01/113928/1605265185~113928~16053~0.87~8.45~~1410.49~16053~1.14~3.28~1728.00~1905.00~1.94~4.72~1905.00~1410.49~0.84~10.55~9.63~-1.95~-0.07~-0.25~1605265185~1605265185~~0~52.35";` + "\n" +
	`v_sz000001="51~平安银行~000001~11.20~11.18~11.14~578042~186997~391045~11.19~1623~11.18~2093~11.17~2593~11.16~1024~11.15~451~11.20~2428~11.21~2097~11.22~2196~11.23~1634~11.24~307~11.25~339~~~20260522113020~0.02~0.18~11.19~11.12~11.20/578042/646306071~578042~64631~0.11~5.43~~11.19~64631~0.59~2.50~13.25~15.60~0.96~0.52~15.60~11.19~0.46~0.32~3.38~-1.32~0.18~0.31~646306071~646306071~~0~37.20";`

var capturedAt = time.Date(2026, 5, 22, 11, 30, 20, 0, time.FixedZone("CST", 8*3600))

func TestParseTencentQuote_GuizhouMoutai(t *testing.T) {
	q := ParseTencentQuote(tencentSampleResponse, "sh600519", capturedAt)

	if q.Kind != model.AssetKindStock {
		t.Errorf("expected Stock kind, got %d", q.Kind)
	}
	if q.Market != "sh" {
		t.Errorf("expected market sh, got %s", q.Market)
	}
	if q.Code != "600519" {
		t.Errorf("expected code 600519, got %s", q.Code)
	}
	if q.Name != "贵州茅台" {
		t.Errorf("expected name 贵州茅台, got %s", q.Name)
	}
	if q.Value <= 0 {
		t.Errorf("expected positive price, got %.2f", q.Value)
	}
	if q.Previous <= 0 {
		t.Errorf("expected positive previous close, got %.2f", q.Previous)
	}
	if !q.Available {
		t.Error("expected Available=true")
	}
	if q.UpdateTime == "" {
		t.Error("expected non-empty UpdateTime")
	}
}

func TestParseTencentQuote_PingAnBank(t *testing.T) {
	q := ParseTencentQuote(tencentSampleResponse, "sz000001", capturedAt)

	if q.Market != "sz" {
		t.Errorf("expected market sz, got %s", q.Market)
	}
	if q.Code != "000001" {
		t.Errorf("expected code 000001, got %s", q.Code)
	}
	if q.Name != "平安银行" {
		t.Errorf("expected name 平安银行, got %s", q.Name)
	}
	if q.Value <= 0 {
		t.Errorf("expected positive price, got %.2f", q.Value)
	}
	if !q.Available {
		t.Error("expected Available=true")
	}
}

func TestParseTencentQuote_MissingStock(t *testing.T) {
	q := ParseTencentQuote(tencentSampleResponse, "sz300750", capturedAt)

	if q.Available {
		t.Error("expected Available=false for missing stock")
	}
	if q.Code != "300750" {
		t.Errorf("expected code 300750, got %s", q.Code)
	}
}

func TestParseTencentQuote_EmptyResponse(t *testing.T) {
	q := ParseTencentQuote("", "sh600519", capturedAt)

	if q.Available {
		t.Error("expected Available=false for empty response")
	}
	if q.Code != "600519" {
		t.Errorf("expected code 600519, got %s", q.Code)
	}
}

func TestParseTencentQuote_TruncatedResponse(t *testing.T) {
	short := "v_sh600519=\"1~贵\""
	q := ParseTencentQuote(short, "sh600519", capturedAt)

	if q.Available {
		t.Error("expected Available=false for truncated response")
	}
}

func TestParseTencentQuote_ZeroPrice(t *testing.T) {
	zeroPrice := `v_sh600519="1~测试~600519~0.00~0.00~0.00~0~0~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~0.00~0~~20260522~0.00~0.00~0.00~0.00~0.00/0/0~0~0~0.00~0.00~~0.00~0~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0~0~0.00";`
	q := ParseTencentQuote(zeroPrice, "sh600519", capturedAt)

	if q.Available {
		t.Error("expected Available=false for zero price")
	}
	if q.Name != "测试" {
		t.Errorf("expected name 测试, got %s", q.Name)
	}
}

func TestParseTencentQuote_ComputedChangePct(t *testing.T) {
	noChangePct := `v_sh600519="1~测试股~600519~10.50~10.00~10.00~0~0~0~10.50~0~10.00~0~10.00~0~10.00~0~10.00~0~10.50~0~10.51~0~10.51~0~10.51~0~10.51~0~10.51~0~~20260522113030~0~0~0.00~0.00~10.50/0/0~0~0~0.00~0.00~~0.00~0~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0.00~0~0~0.00";`
	q := ParseTencentQuote(noChangePct, "sh600519", capturedAt)

	if !q.Available {
		t.Error("expected Available=true")
	}
	expected := (10.50 - 10.00) / 10.00 * 100
	if q.ChangePct < expected-0.01 || q.ChangePct > expected+0.01 {
		t.Errorf("expected change pct around %.2f, got %.2f", expected, q.ChangePct)
	}
}

func TestParseTencentQuote_FallbackUpdateTime(t *testing.T) {
	noTime := `v_sz000001="51~平安银行~000001~11.20~11.18~11.14~578042~186997~391045~11.19~1623~11.18~2093~11.17~2593~11.16~1024~11.15~451~11.20~2428~11.21~2097~11.22~2196~11.23~1634~11.24~307~11.25~339~0~0~0~0.02~0.18~11.19~11.12~11.20/578042/646306071~578042~64631~0.11~5.43~~11.19~64631~0.59~2.50~13.25~15.60~0.96~0.52~15.60~11.19~0.46~0.32~3.38~-1.32~0.18~0.31~646306071~646306071~~0~37.20";`
	q := ParseTencentQuote(noTime, "sz000001", capturedAt)

	if q.UpdateTime != capturedAt.Format("15:04:05") {
		t.Errorf("expected fallback update time %s, got %s", capturedAt.Format("15:04:05"), q.UpdateTime)
	}
}

func TestParseTencentQuote_MultiLineResponse(t *testing.T) {
	q := ParseTencentQuote(tencentSampleResponse, "sh600519", capturedAt)
	if !strings.Contains(tencentSampleResponse, "\n") {
		t.Skip("sample already single-line")
	}
	if !q.Available {
		t.Error("expected Available=true with multi-line response")
	}
}

func TestInferStockMarket(t *testing.T) {
	tests := []struct {
		code    string
		wantMkt string
		wantErr bool
	}{
		{"600519", "sh", false},
		{"000001", "sz", false},
		{"300750", "sz", false},
		{"688981", "sh", false},
		{"430001", "", true},  // Beijing
		{"830001", "", true},  // Beijing
		{"12345", "hk", false},
		{"1234567", "", true}, // too long
		{"200001", "", true},  // unknown prefix
	}

	for _, tt := range tests {
		mkt, err := model.InferStockMarket(tt.code)
		if tt.wantErr {
			if err == nil {
				t.Errorf("InferStockMarket(%s): expected error, got market=%s", tt.code, mkt)
			}
		} else {
			if err != nil {
				t.Errorf("InferStockMarket(%s): unexpected error: %v", tt.code, err)
			}
			if mkt != tt.wantMkt {
				t.Errorf("InferStockMarket(%s): got %s, want %s", tt.code, mkt, tt.wantMkt)
			}
		}
	}
}
