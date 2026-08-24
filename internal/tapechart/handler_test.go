package tapechart

import (
	"net/http/httptest"
	"testing"
)

func TestParseTapeChartQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
		want    IncludeMode
	}{
		{"valid full range defaults to include=full", "from=2026-07-13&to=2026-07-20", false, IncludeFull},
		{"valid shallow", "from=2026-07-13&to=2026-07-20&include=shallow", false, IncludeShallow},
		{"valid explicit full", "from=2026-07-13&to=2026-07-20&include=full", false, IncludeFull},
		{"missing from", "to=2026-07-20", true, ""},
		{"missing to", "from=2026-07-13", true, ""},
		{"malformed from", "from=13-07-2026&to=2026-07-20", true, ""},
		{"malformed to", "from=2026-07-13&to=not-a-date", true, ""},
		{"to before from", "from=2026-07-20&to=2026-07-13", true, ""},
		{"exactly 90 days is allowed", "from=2026-01-01&to=2026-04-01", false, IncludeFull},
		{"beyond 90-day max is rejected", "from=2026-01-01&to=2026-06-01", true, ""},
		{"invalid include value", "from=2026-07-13&to=2026-07-20&include=bogus", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.query, nil)
			_, _, include, err := parseTapeChartQuery(req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && include != tt.want {
				t.Errorf("include = %q, want %q", include, tt.want)
			}
		})
	}
}

func TestParseTapeChartQuery_FromToParsedAsUTCMidnight(t *testing.T) {
	req := httptest.NewRequest("GET", "/?from=2026-07-13&to=2026-07-20", nil)
	from, to, _, err := parseTapeChartQuery(req)
	if err != nil {
		t.Fatalf("parseTapeChartQuery: %v", err)
	}
	if from.Format("2006-01-02") != "2026-07-13" {
		t.Errorf("from = %v, want 2026-07-13", from)
	}
	if to.Format("2006-01-02") != "2026-07-20" {
		t.Errorf("to = %v, want 2026-07-20", to)
	}
}
