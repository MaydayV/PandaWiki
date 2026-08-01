package bot

import "testing"

func TestParseWebhookTarget(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantURL string
		wantSec string
	}{
		{name: "url only", input: "https://example.com/hook", wantURL: "https://example.com/hook"},
		{name: "url with secret", input: "https://example.com/hook|SEC123", wantURL: "https://example.com/hook", wantSec: "SEC123"},
		{name: "trims spaces", input: "  https://example.com/hook | SEC123  ", wantURL: "https://example.com/hook", wantSec: "SEC123"},
		{name: "empty", input: "   "},
		{name: "pipe at start ignored as secret", input: "|only", wantURL: "|only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotSec := ParseWebhookTarget(tt.input)
			if gotURL != tt.wantURL || gotSec != tt.wantSec {
				t.Fatalf("ParseWebhookTarget(%q)=(%q,%q), want (%q,%q)", tt.input, gotURL, gotSec, tt.wantURL, tt.wantSec)
			}
		})
	}
}
