//go:build integration

package data

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"planstar/internal/biz"
	"planstar/internal/testutil"
)

// newOpsRepoForTest wires an OpsRepo onto real per-domain MySQL connections,
// mirroring production: each domain pool carries its own default database so
// unqualified statements resolve, while ReviewRefund's cross-DB qualified names
// still span payment and order on the same instance.
func newOpsRepoForTest(t *testing.T) (*OpsRepo, *sql.DB) {
	t.Helper()
	d := &Data{dbs: map[string]*sql.DB{
		"community": testutil.MySQLDatabase(t, "planstar_community"),
		"shop":      testutil.MySQLDatabase(t, "planstar_shop"),
		"payment":   testutil.MySQLDatabase(t, "planstar_payment"),
		"order":     testutil.MySQLDatabase(t, "planstar_order"),
	}}
	// Seed/read statements below are fully qualified, so any pool works.
	return NewOpsRepo(d), d.dbs["community"]
}

func TestOpsSetStatusIntegration(t *testing.T) {
	repo, db := newOpsRepoForTest(t)
	ctx := context.Background()

	testutil.Exec(t, db, `INSERT INTO planstar_community.feed (feed_id,user_id,status) VALUES (999101,999101,1)`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_community.feed WHERE feed_id=999101`)
	testutil.Exec(t, db, `INSERT INTO planstar_shop.goods (goods_id,goods_sn,name,status) VALUES (999101,'TMPIT999101','temp',1)`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_shop.goods WHERE goods_id=999101`)
	testutil.Exec(t, db, `INSERT INTO planstar_shop.goods_sku (sku_id,goods_id,sku_name,price_coins,status) VALUES (999101,999101,'temp',10,1)`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_shop.goods_sku WHERE sku_id=999101`)

	cases := []struct {
		name    string
		set     func(context.Context, int64, int32) error
		table   string
		idCol   string
		id      int64
		status  int32
		wantErr error
	}{
		{"feed hide", repo.SetFeedStatus, "planstar_community.feed", "feed_id", 999101, 0, nil},
		{"feed show", repo.SetFeedStatus, "planstar_community.feed", "feed_id", 999101, 1, nil},
		{"feed same value is not not-found", repo.SetFeedStatus, "planstar_community.feed", "feed_id", 999101, 1, nil},
		{"feed missing", repo.SetFeedStatus, "planstar_community.feed", "feed_id", 999999, 0, biz.ErrNotFound},
		{"goods off", repo.SetGoodsStatus, "planstar_shop.goods", "goods_id", 999101, 0, nil},
		{"goods on", repo.SetGoodsStatus, "planstar_shop.goods", "goods_id", 999101, 1, nil},
		{"goods missing", repo.SetGoodsStatus, "planstar_shop.goods", "goods_id", 999999, 0, biz.ErrNotFound},
		{"sku off", repo.SetGoodsSKUStatus, "planstar_shop.goods_sku", "sku_id", 999101, 0, nil},
		{"sku on", repo.SetGoodsSKUStatus, "planstar_shop.goods_sku", "sku_id", 999101, 1, nil},
		{"sku missing", repo.SetGoodsSKUStatus, "planstar_shop.goods_sku", "sku_id", 999999, 0, biz.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.set(ctx, tc.id, tc.status)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			var got int32
			q := "SELECT status FROM " + tc.table + " WHERE " + tc.idCol + " = ?"
			if err := db.QueryRowContext(ctx, q, tc.id).Scan(&got); err != nil {
				t.Fatalf("read back: %v", err)
			}
			if got != tc.status {
				t.Fatalf("status = %d, want %d", got, tc.status)
			}
		})
	}
}

func TestOpsReviewRefundIntegration(t *testing.T) {
	repo, db := newOpsRepoForTest(t)
	ctx := context.Background()

	testutil.Exec(t, db, `INSERT INTO planstar_order.order_info
		(id,order_id,order_number,user_id,status,pay_amount,total_amount,refund_amount)
		VALUES (999101,999101,'TMPITORD999101',999101,2,50.00,50.00,0.00),
		       (999102,999102,'TMPITORD999102',999102,2,30.00,30.00,0.00)`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_order.order_info WHERE id IN (999101,999102)`)
	testutil.Exec(t, db, `INSERT INTO planstar_payment.order_refund
		(id,refund_no,order_id,order_number,user_id,refundable_amount,status)
		VALUES (999101,'TMPITREF999101',999101,'TMPITORD999101',999101,50.00,'applied'),
		       (999102,'TMPITREF999102',999102,'TMPITORD999102',999102,30.00,'applied')`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_payment.order_refund WHERE id IN (999101,999102)`)

	t.Run("approve writes refund and order in one tx", func(t *testing.T) {
		out, err := repo.ReviewRefund(ctx, 999101, true)
		if err != nil {
			t.Fatalf("approve: %v", err)
		}
		if out.Status != "success" || out.OrderStatus != 7 || out.RefundAmount != 5000 {
			t.Fatalf("review = %+v, want success/7/5000", out)
		}
		var rStatus string
		var oStatus int32
		var oAmount float64
		if err := db.QueryRowContext(ctx, `SELECT status FROM planstar_payment.order_refund WHERE id=999101`).Scan(&rStatus); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT status,refund_amount FROM planstar_order.order_info WHERE order_id=999101`).Scan(&oStatus, &oAmount); err != nil {
			t.Fatal(err)
		}
		if rStatus != "success" || oStatus != 7 || oAmount != 50.00 {
			t.Fatalf("db refund=%s order=%d/%.2f, want success/7/50.00", rStatus, oStatus, oAmount)
		}
	})

	t.Run("reject leaves order untouched", func(t *testing.T) {
		out, err := repo.ReviewRefund(ctx, 999102, false)
		if err != nil {
			t.Fatalf("reject: %v", err)
		}
		if out.Status != "rejected" || out.OrderStatus != 2 {
			t.Fatalf("review = %+v, want rejected/2", out)
		}
		var oStatus int32
		var oAmount float64
		if err := db.QueryRowContext(ctx, `SELECT status,refund_amount FROM planstar_order.order_info WHERE order_id=999102`).Scan(&oStatus, &oAmount); err != nil {
			t.Fatal(err)
		}
		if oStatus != 2 || oAmount != 0.00 {
			t.Fatalf("order = %d/%.2f, want unchanged 2/0.00", oStatus, oAmount)
		}
	})

	t.Run("second approve is not allowed", func(t *testing.T) {
		if _, err := repo.ReviewRefund(ctx, 999101, true); !errors.Is(err, biz.ErrRefundNotAllowed) {
			t.Fatalf("err = %v, want ErrRefundNotAllowed", err)
		}
	})

	t.Run("missing refund is not found", func(t *testing.T) {
		if _, err := repo.ReviewRefund(ctx, 999999, true); !errors.Is(err, biz.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
