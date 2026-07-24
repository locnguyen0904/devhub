package post

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrips(t *testing.T) {
	want := feedCursor{
		publishedAt: time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC),
		id:          uuid.MustParse("01890000-0000-7000-8000-000000000001"),
	}

	got, err := decodeCursor(encodeCursor(want))
	if err != nil {
		t.Fatalf("decodeCursor() error = %v", err)
	}
	if got == nil {
		t.Fatal("decodeCursor() = nil, want a cursor")
	}
	if !got.publishedAt.Equal(want.publishedAt) || got.id != want.id {
		t.Errorf("round trip = {%v, %v}, want {%v, %v}", got.publishedAt, got.id, want.publishedAt, want.id)
	}
}

func TestDecodeEmptyCursorIsFirstPage(t *testing.T) {
	got, err := decodeCursor("")
	if err != nil {
		t.Fatalf("decodeCursor(\"\") error = %v", err)
	}
	if got != nil {
		t.Errorf("decodeCursor(\"\") = %v, want nil for the first page", got)
	}
}

func TestDecodeGarbageCursorErrors(t *testing.T) {
	if _, err := decodeCursor("!!!not-base64!!!"); err == nil {
		t.Error("decodeCursor(garbage) = nil error, want an error")
	}
}

func TestExcerptStripsHTMLAndTruncates(t *testing.T) {
	got := excerpt("<h1>Title</h1><p>Hello <strong>world</strong></p>")
	if got != "Title Hello world" {
		t.Errorf("excerpt() = %q, want %q", got, "Title Hello world")
	}
}
