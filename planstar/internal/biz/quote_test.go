package biz

import (
	"errors"
	"testing"
	"time"
)

func TestCalcQuoteTotal(t *testing.T) {
	total, err := CalcQuoteTotal(1990, 2)
	if err != nil || total != 3980 {
		t.Fatalf("CalcQuoteTotal(1990,2) = %d,%v; want 3980,nil", total, err)
	}
	if _, err := CalcQuoteTotal(1990, 0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("quantity 0 err = %v, want ErrBadRequest", err)
	}
	if _, err := CalcQuoteTotal(0, 1); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("unit price 0 err = %v, want ErrBadRequest", err)
	}
	if _, err := CalcQuoteTotal(1990, MaxQuantityPerOrder+1); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("over limit err = %v, want ErrLimitExceeded", err)
	}
}

func TestVerifyQuote(t *testing.T) {
	now := time.Now()
	base := &QuoteSnapshot{
		UnitPrice: 1990, Quantity: 2, TotalAmount: 3980,
		Version: quoteVersion, ExpiresAt: now.Add(time.Minute).UnixMilli(),
	}
	if err := VerifyQuote(base, 3980, quoteVersion, now); err != nil {
		t.Fatalf("valid quote err = %v, want nil", err)
	}
	if err := VerifyQuote(base, 3979, quoteVersion, now); !errors.Is(err, ErrPriceChanged) {
		t.Fatalf("amount mismatch err = %v, want ErrPriceChanged", err)
	}
	if err := VerifyQuote(base, 3980, quoteVersion+1, now); !errors.Is(err, ErrPriceChanged) {
		t.Fatalf("version mismatch err = %v, want ErrPriceChanged", err)
	}
	expired := *base
	expired.ExpiresAt = now.Add(-time.Second).UnixMilli()
	if err := VerifyQuote(&expired, 3980, quoteVersion, now); !errors.Is(err, ErrQuoteExpired) {
		t.Fatalf("expired err = %v, want ErrQuoteExpired", err)
	}
	if err := VerifyQuote(nil, 3980, quoteVersion, now); !errors.Is(err, ErrQuoteExpired) {
		t.Fatalf("nil quote err = %v, want ErrQuoteExpired", err)
	}
}

func TestVerifyQuoteRejectsTamperedTotal(t *testing.T) {
	now := time.Now()
	snap := &QuoteSnapshot{
		UnitPrice: 1990, Quantity: 2, TotalAmount: 1,
		Version: quoteVersion, ExpiresAt: now.Add(time.Minute).UnixMilli(),
	}
	if err := VerifyQuote(snap, 1, quoteVersion, now); !errors.Is(err, ErrPriceChanged) {
		t.Fatalf("tampered total err = %v, want ErrPriceChanged", err)
	}
}
