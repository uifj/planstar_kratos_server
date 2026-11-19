package biz

import (
	"testing"
	"time"
)

func TestFen(t *testing.T) {
	cases := []struct {
		in   float64
		want int64
	}{
		{0, 0},
		{19.9, 1990},
		{0.01, 1},
		{1234.56, 123456},
	}
	for _, c := range cases {
		if got := Fen(c.in); got != c.want {
			t.Fatalf("Fen(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestFmtDT(t *testing.T) {
	ts := time.Date(2026, 11, 6, 19, 30, 5, 0, time.Local)
	if got, want := FmtDT(ts), "2026-11-06 19:30:05"; got != want {
		t.Fatalf("FmtDT = %q, want %q", got, want)
	}
	if got := FmtDTp(nil); got != "" {
		t.Fatalf("FmtDTp(nil) = %q, want empty", got)
	}
	if got := FmtDTp(&ts); got != "2026-11-06 19:30:05" {
		t.Fatalf("FmtDTp = %q", got)
	}
}

func TestBirthTs(t *testing.T) {
	if got := BirthTs(nil); got != 0 {
		t.Fatalf("BirthTs(nil) = %d, want 0", got)
	}
	ts := time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	if got, want := BirthTs(&ts), ts.UnixMilli(); got != want {
		t.Fatalf("BirthTs = %d, want %d", got, want)
	}
}

func TestNewEvent(t *testing.T) {
	ev := NewEvent(EventOrderCreated, 42, map[string]any{"orderNumber": "PN1"})
	if ev.ID == "" || ev.Type != EventOrderCreated || ev.UserID != 42 {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if ev.OccurredAt.IsZero() {
		t.Fatal("OccurredAt should be set")
	}
	if NewEvent(EventOrderCreated, 1, nil).ID == ev.ID {
		t.Fatal("event IDs must be unique")
	}
}
