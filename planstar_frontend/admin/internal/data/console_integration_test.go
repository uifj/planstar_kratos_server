//go:build integration

package data

import (
	"context"
	"testing"

	"planstar/admin/internal/testutil"
)

// TestGetGoodsSkuIDIsSkuIDIntegration pins the SKU identifier contract: the read
// surface exposes goods_sku.sku_id (the business key the ops write targets), not
// the auto-increment row id. Seeding them to different values makes a regression
// to the wrong column fail loudly instead of passing by coincidence.
func TestGetGoodsSkuIDIsSkuIDIntegration(t *testing.T) {
	db := testutil.MySQL(t)
	repo := NewConsoleRepo(&Data{readDB: db})
	ctx := context.Background()

	testutil.Exec(t, db, `INSERT INTO planstar_shop.goods (goods_id,goods_sn,name,status) VALUES (999101,'TMPIT999101','temp',1)`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_shop.goods WHERE goods_id=999101`)
	testutil.Exec(t, db, `INSERT INTO planstar_shop.goods_sku (id,sku_id,goods_id,sku_name,price_coins,stock,sold,status) VALUES (999101,999201,999101,'temp',10,5,0,1)`)
	testutil.Cleanup(t, db, `DELETE FROM planstar_shop.goods_sku WHERE goods_id=999101`)

	detail, err := repo.GetGoods(ctx, 999101)
	if err != nil {
		t.Fatalf("GetGoods: %v", err)
	}
	if len(detail.Skus) != 1 {
		t.Fatalf("skus = %d, want 1", len(detail.Skus))
	}
	sku := detail.Skus[0]
	if sku.ID != 999201 {
		t.Fatalf("sku.ID = %d, want sku_id 999201 (not row id 999101)", sku.ID)
	}
	if sku.PriceCoins != 10 || sku.Stock != 5 || sku.Status != 1 {
		t.Fatalf("sku = %+v, want price 10 stock 5 status 1", sku)
	}
}
