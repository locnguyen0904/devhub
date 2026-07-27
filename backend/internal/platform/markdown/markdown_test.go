package markdown

import (
	"strings"
	"testing"
)

func TestRenderStripsDangerousHTML(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		name       string
		source     string
		mustReject string
	}{
		{"script tag", "Hello <script>alert('xss')</script>", "<script"},
		{"onerror handler", `<img src=x onerror="steal()">`, "onerror"},
		{"javascript href", "[click](javascript:alert(1))", "javascript:"},
		{"iframe", "<iframe src=evil></iframe>", "<iframe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := r.Render(tt.source)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if strings.Contains(html, tt.mustReject) {
				t.Errorf("Render(%q) = %q, must not contain %q", tt.source, html, tt.mustReject)
			}
		})
	}
}

func TestRenderKeepsSafeMarkdown(t *testing.T) {
	r := NewRenderer()

	html, err := r.Render("# Title\n\nSome **bold** and a [link](https://example.com).")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, want := range []string{"<h1", "<strong>bold</strong>", `href="https://example.com"`} {
		if !strings.Contains(html, want) {
			t.Errorf("Render() = %q, want it to contain %q", html, want)
		}
	}
}

func TestRenderHighlightsCodeWithClasses(t *testing.T) {
	r := NewRenderer()

	html, err := r.Render("```go\nfunc main() {}\n```")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	// Classed highlighting means CSS controls colour, so the theme can follow
	// light/dark mode instead of being frozen into inline styles.
	if strings.Contains(html, "style=") {
		t.Errorf("Render() emitted inline styles, want class-based highlighting: %q", html)
	}
	if !strings.Contains(html, "class=") {
		t.Errorf("Render() has no highlight classes: %q", html)
	}
}

func TestSanitizeHeadlineKeepsBoldStripsRest(t *testing.T) {
	r := NewRenderer()
	tests := []struct {
		name       string
		snippet    string
		mustHave   string
		mustReject string
	}{
		{"keeps bold markers", "a <b>match</b> here", "<b>match</b>", ""},
		{"strips script", "<b>x</b> <script>steal()</script>", "<b>x</b>", "<script"},
		{"strips img onerror", `<b>y</b> <img src=x onerror=e>`, "<b>y</b>", "onerror"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.SanitizeHeadline(tt.snippet)
			if !strings.Contains(got, tt.mustHave) {
				t.Errorf("SanitizeHeadline(%q) = %q, want it to contain %q", tt.snippet, got, tt.mustHave)
			}
			if tt.mustReject != "" && strings.Contains(got, tt.mustReject) {
				t.Errorf("SanitizeHeadline(%q) = %q, must not contain %q", tt.snippet, got, tt.mustReject)
			}
		})
	}
}

func TestReadingMinutes(t *testing.T) {
	tests := []struct {
		name  string
		words int
		want  int
	}{
		{"empty is at least one minute", 0, 1},
		{"short is at least one minute", 50, 1},
		{"exactly two hundred words", 200, 1},
		{"four hundred words", 400, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := strings.Repeat("word ", tt.words)
			if got := ReadingMinutes(source); got != tt.want {
				t.Errorf("ReadingMinutes(%d words) = %d, want %d", tt.words, got, tt.want)
			}
		})
	}
}
