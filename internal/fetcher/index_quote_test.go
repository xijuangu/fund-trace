package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Captured from push2.eastmoney.com/api/qt/ulist.np/get. f3 is scaled by 100
// and may be "-" for instruments without a quote.
const indexQuoteSample = `{"rc":0,"data":{"total":3,"diff":[
	{"f2":1119222,"f3":-137,"f12":"H30269","f13":2,"f14":"红利低波"},
	{"f2":505780,"f3":250,"f12":"980030","f13":0,"f14":"消费电子"},
	{"f2":0,"f3":"-","f12":"399999","f13":0,"f14":"停牌指数"}
]}}`

func TestFetchIndexChanges(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RawQuery
		fmt.Fprint(w, indexQuoteSample)
	}))
	defer server.Close()

	client := New(2)
	changes, err := client.fetchIndexChanges(server.URL+"/q?secids=%s&fields=f2,f3,f12,f13,f14",
		[]string{"2.H30269", "0.980030", "0.399999", "2.H30269"})
	if err != nil {
		t.Fatalf("fetchIndexChanges: %v", err)
	}

	if got, want := changes["2.H30269"], -1.37; got != want {
		t.Errorf("H30269 change = %v, want %v", got, want)
	}
	if got, want := changes["0.980030"], 2.5; got != want {
		t.Errorf("980030 change = %v, want %v", got, want)
	}
	if got := changes["0.399999"]; got != 0 {
		t.Errorf("missing quote should decode as 0, got %v", got)
	}

	// Duplicate secids must be sent only once.
	if n := countOccurrences(gotPath, "2.H30269"); n != 1 {
		t.Errorf("duplicate secid sent %d times, want 1 (query=%s)", n, gotPath)
	}
}

func TestFetchIndexChanges_Empty(t *testing.T) {
	client := New(2)
	changes, err := client.fetchIndexChanges("http://127.0.0.1:1/q?secids=%s", nil)
	if err != nil {
		t.Fatalf("expected no error for empty input, got %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected empty map, got %v", changes)
	}
}

func TestFlexFloat(t *testing.T) {
	var v flexFloat
	if err := v.UnmarshalJSON([]byte(`-1.37`)); err != nil || float64(v) != -1.37 {
		t.Errorf("number: got %v, err %v", float64(v), err)
	}
	if err := v.UnmarshalJSON([]byte(`"-"`)); err != nil || float64(v) != 0 {
		t.Errorf("dash: got %v, err %v", float64(v), err)
	}
}

func countOccurrences(s, sub string) int {
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
		}
	}
	return n
}
