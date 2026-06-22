package tools

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantContain string
		wantAbsent  string
	}{
		{
			name:        "bearer token in Authorization header",
			input:       `curl -H "Authorization: Bearer sk-abc123def456" https://x`,
			wantContain: "Bearer ***",
			wantAbsent:  "sk-abc123def456",
		},
		{
			name:       "export TOKEN assignment",
			input:      `export TOKEN=supersecretvalue`,
			wantAbsent: "supersecretvalue",
		},
		{
			name:       "AWS access key id env var",
			input:      `API_KEY=AKIAEXAMPLE1234567890 ./run`,
			wantAbsent: "AKIAEXAMPLE1234567890",
		},
		{
			name:        "bare AWS access key id",
			input:       `echo AKIA0123456789ABCDEF`,
			wantContain: "AKIA****************",
			wantAbsent:  "AKIA0123456789ABCDEF",
		},
		{
			name:       "password colon value",
			input:      `password: hunter2longenough`,
			wantAbsent: "hunter2longenough",
		},
		{
			name:        "benign command unchanged",
			input:       `ls -la /tmp && go test ./...`,
			wantContain: `ls -la /tmp && go test ./...`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := redactSecrets(tc.input)
			if tc.wantContain != "" && !strings.Contains(got, tc.wantContain) {
				t.Errorf("redactSecrets(%q) = %q: want it to contain %q", tc.input, got, tc.wantContain)
			}
			if tc.wantAbsent != "" && strings.Contains(got, tc.wantAbsent) {
				t.Errorf("redactSecrets(%q) = %q: secret %q must not appear in output", tc.input, got, tc.wantAbsent)
			}
		})
	}
}
