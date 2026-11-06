package captcha

import (
	"bytes"
	"image/png"
	"testing"
)

func TestNew(t *testing.T) {
	id, code, raw, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(id) != 16 {
		t.Fatalf("captcha id length = %d, want 16", len(id))
	}
	if len(code) != 4 {
		t.Fatalf("code = %q, want 4 digits", code)
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			t.Fatalf("code contains non-digit: %q", code)
		}
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 120 || b.Dy() != 44 {
		t.Fatalf("image size = %dx%d", b.Dx(), b.Dy())
	}

	id2, _, _, _ := New()
	if id == id2 {
		t.Fatal("captcha ids must be unique")
	}
}
