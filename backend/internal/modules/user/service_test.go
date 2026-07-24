package user

import "testing"

const validLogin = "locnguyen"

func TestSanitizeUsername(t *testing.T) {
	tests := []struct {
		name  string
		login string
		want  string
	}{
		{"keeps a valid login unchanged", validLogin, validLogin},
		{"keeps allowed separators", "loc_nguyen-04", "loc_nguyen-04"},
		{"strips disallowed characters", "loc.nguyen!", "locnguyen"},
		{"pads a too-short login", "ab", "abdev"},
		{"truncates a too-long login", "abcdefghijklmnopqrstuvwxyz0123456789", "abcdefghijklmnopqrstuvwxyz0123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeUsername(tt.login)
			if got != tt.want {
				t.Errorf("sanitizeUsername(%q) = %q, want %q", tt.login, got, tt.want)
			}
			if len(got) < 3 || len(got) > 30 {
				t.Errorf("sanitizeUsername(%q) length = %d, want within [3,30]", tt.login, len(got))
			}
		})
	}
}
