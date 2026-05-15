package daemon

import "testing"

func TestParseMerchantOrderID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		body string
		want string
	}{
		{`{"orderId":"GRC-001","status":"confirmed"}`, "GRC-001"},
		{`{"status":"confirmed"}`, ""},
		{"", ""},
		{"not json", ""},
	}
	for _, tc := range tests {
		got := parseMerchantOrderID([]byte(tc.body))
		if got != tc.want {
			t.Errorf("parseMerchantOrderID(%q) = %q, want %q", tc.body, got, tc.want)
		}
	}
}
