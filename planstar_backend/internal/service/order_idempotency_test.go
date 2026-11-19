package service

import (
	"strings"
	"testing"

	"planstar/internal/biz"
)

func sampleOrderInput() *biz.OrderInput {
	return &biz.OrderInput{
		ActivityID: 1001, ActivityEventID: 2002, TicketCategoryID: 3003,
		OrderType: 1, Quantity: 2, Amount: 38000, PayAmount: 38000, TotalAmount: 38000,
		QuoteID: "quote-abc", QuoteVersion: 1,
		Name: "购票人", Phone: "13800000001", AddressID: 7,
	}
}

func TestDeriveIdempotencyKeyIsStable(t *testing.T) {
	in := sampleOrderInput()
	a := deriveIdempotencyKey(42, in)
	b := deriveIdempotencyKey(42, in)
	if a != b {
		t.Fatalf("same request produced different keys: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "auto:") {
		t.Fatalf("derived key missing prefix: %q", a)
	}
	// 键长必须小于 order_idempotency.idem_key VARCHAR(128)
	if len(a) > 128 {
		t.Fatalf("derived key too long: %d", len(a))
	}
}

func TestDeriveIdempotencyKeyVariesWithMeaningfulFields(t *testing.T) {
	base := deriveIdempotencyKey(42, sampleOrderInput())

	cases := map[string]func(*biz.OrderInput){
		"user":     func(*biz.OrderInput) {},
		"quote":    func(in *biz.OrderInput) { in.QuoteID = "quote-xyz" },
		"quantity": func(in *biz.OrderInput) { in.Quantity = 3 },
		"amount":   func(in *biz.OrderInput) { in.TotalAmount = 39000 },
		"ticket":   func(in *biz.OrderInput) { in.TicketCategoryID = 9999 },
	}
	for name, mutate := range cases {
		if name == "user" {
			continue
		}
		in := sampleOrderInput()
		mutate(in)
		if got := deriveIdempotencyKey(42, in); got == base {
			t.Errorf("%s change did not alter idempotency key", name)
		}
	}

	other := deriveIdempotencyKey(43, sampleOrderInput())
	if other == base {
		t.Fatal("different users must not share a derived idempotency key")
	}
}
