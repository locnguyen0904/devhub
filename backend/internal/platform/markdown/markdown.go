// Package markdown renders post markdown to sanitized HTML.
//
// The order is render-then-sanitize, and it is not negotiable: markdown allows
// raw HTML, so sanitizing the source would both miss injected tags and corrupt
// valid markdown. Skipping the sanitize step is a stored-XSS hole — one post
// could hijack the session of everyone who reads it. See docs/01-architecture §5.
package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// wordsPerMinute is the reading-speed constant behind the reading-time estimate.
// 200 wpm is the common average for technical prose.
const wordsPerMinute = 200

// Renderer turns markdown into sanitized HTML. It is safe for concurrent use and
// is built once at startup because compiling the goldmark pipeline and the
// bluemonday policy is not free.
type Renderer struct {
	md       goldmark.Markdown
	policy   *bluemonday.Policy
	headline *bluemonday.Policy
}

// NewRenderer builds the renderer.
func NewRenderer() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			// WithClasses emits <span class="k"> etc. instead of inline styles,
			// so the code theme is CSS-controlled and can follow light/dark mode.
			highlighting.NewHighlighting(highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			)),
		),
		goldmark.WithParserOptions(
			// Auto-generate heading ids so the frontend can build a table of
			// contents and deep-link to sections.
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			// Do NOT enable WithUnsafe: raw HTML passes through to be handled by
			// the sanitizer below, which is the single place HTML is made safe.
			gmhtml.WithHardWraps(),
		),
	)

	// headline allows only the <b> tags ts_headline wraps matches in.
	headline := bluemonday.NewPolicy()
	headline.AllowElements("b")

	return &Renderer{md: md, policy: buildPolicy(), headline: headline}
}

// Render converts markdown to HTML that is safe to store and serve.
func (r *Renderer) Render(source string) (string, error) {
	var buf bytes.Buffer
	if err := r.md.Convert([]byte(source), &buf); err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}
	return r.policy.Sanitize(buf.String()), nil
}

// SanitizeHeadline makes a Postgres ts_headline snippet safe to render. The
// snippet is built from raw post markdown, which may contain user HTML, so
// everything but the <b> match markers is stripped — otherwise a post could
// smuggle script into search results.
func (r *Renderer) SanitizeHeadline(snippet string) string {
	return r.headline.Sanitize(snippet)
}

// ReadingMinutes estimates reading time from the raw markdown, never below one.
func ReadingMinutes(source string) int {
	words := len(strings.Fields(source))
	minutes := words / wordsPerMinute
	if minutes < 1 {
		return 1
	}
	return minutes
}

// buildPolicy is the allowlist of HTML the sanitizer permits. It starts from
// bluemonday's UGC policy and widens it just enough for chroma's syntax
// highlighting and heading anchors.
func buildPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// chroma marks tokens with short class names on span/pre/code; heading ids
	// come through as id attributes on h1-h6.
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^(chroma|language-|[a-z]{1,3})$`)).
		OnElements("span", "code", "pre", "div")
	p.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	// Footnote links reference ids within the document.
	p.AllowAttrs("id", "role").OnElements("li", "section", "sup")
	p.AllowAttrs("href").OnElements("a")

	return p
}
