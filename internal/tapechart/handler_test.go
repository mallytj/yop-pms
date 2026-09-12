package tapechart

import (
	"testing"
)

func TestParseTapeChartQuery(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		include string
		wantErr bool
		want    IncludeMode
	}{
		{"valid full range defaults to include=full", "2026-07-13", "2026-07-20", "", false, IncludeFull},
		{"valid shallow", "2026-07-13", "2026-07-20", "shallow", false, IncludeShallow},
		{"valid explicit full", "2026-07-13", "2026-07-20", "full", false, IncludeFull},
		{"missing from", "", "2026-07-20", "", true, ""},
		{"missing to", "2026-07-13", "", "", true, ""},
		{"malformed from", "13-07-2026", "2026-07-20", "", true, ""},
		{"malformed to", "2026-07-13", "not-a-date", "", true, ""},
		{"to before from", "2026-07-20", "2026-07-13", "", true, ""},
		{"exactly 90 days is allowed", "2026-01-01", "2026-04-01", "", false, IncludeFull},
		{"beyond 90-day max is rejected", "2026-01-01", "2026-06-01", "", true, ""},
		{"invalid include value", "2026-07-13", "2026-07-20", "bogus", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, include, err := parseTapeChartQuery(tt.from, tt.to, tt.include)

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
	from, to, _, err := parseTapeChartQuery("2026-07-13", "2026-07-20", "")
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
