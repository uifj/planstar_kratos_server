package data

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"planstar/internal/biz"
)

// ShopRepo 实现 biz.ShopRepo，落 planstar_shop。
type ShopRepo struct {
	d *Data
}

func NewShopRepo(d *Data) *ShopRepo { return &ShopRepo{d: d} }

func (r *ShopRepo) Goods(ctx context.Context) ([]*biz.Goods, error) {
	db, err := r.d.DB("shop")
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT goods_id, name, COALESCE(brief,''), COALESCE(description,''), COALESCE(cover,'')
		   FROM goods WHERE status = 1
		  ORDER BY sort ASC, goods_id ASC`)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.Goods, 0, 8)
	for rows.Next() {
		g := &biz.Goods{SKUs: []biz.GoodsSKU{}}
		if err := rows.Scan(&g.GoodsID, &g.Name, &g.Brief, &g.Description, &g.Cover); err != nil {
			_ = rows.Close()
			return nil, err
		}
		items = append(items, g)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, g := range items {
		skus, err := skusByGoods(ctx, db, g.GoodsID)
		if err != nil {
			return nil, err
		}
		g.SKUs = skus
	}
	return items, nil
}

func (r *ShopRepo) GoodsDetail(ctx context.Context, goodsID int64) (*biz.Goods, error) {
	db, err := r.d.DB("shop")
	if err != nil {
		return nil, err
	}
	g := &biz.Goods{SKUs: []biz.GoodsSKU{}}
	err = db.QueryRowContext(ctx,
		`SELECT goods_id, name, COALESCE(brief,''), COALESCE(description,''), COALESCE(cover,'')
		   FROM goods WHERE goods_id = ? AND status = 1 LIMIT 1`, goodsID).
		Scan(&g.GoodsID, &g.Name, &g.Brief, &g.Description, &g.Cover)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if g.SKUs, err = skusByGoods(ctx, db, goodsID); err != nil {
		return nil, err
	}
	return g, nil
}

// Redeem 星币兑换：单事务内锁定规格与钱包账户，扣币、减库存、写流水与订单。
func (r *ShopRepo) Redeem(ctx context.Context, userID, goodsID, skuID int64) (*biz.RedeemResult, error) {
	db, err := r.d.DB("shop")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var (
		goodsName string
		skuName   string
		price     int64
		stock     int64
		sold      int64
	)
	err = tx.QueryRowContext(ctx,
		`SELECT g.name, s.sku_name, s.price_coins, s.stock, s.sold
		   FROM planstar_shop.goods_sku s
		   JOIN planstar_shop.goods g ON g.goods_id = s.goods_id
		  WHERE s.goods_id = ? AND s.sku_id = ? AND s.status = 1 AND g.status = 1
		  FOR UPDATE`, goodsID, skuID).
		Scan(&goodsName, &skuName, &price, &stock, &sold)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if stock <= 0 {
		return nil, biz.ErrSoldOut
	}

	accountID, balance, err := ensureWalletTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	priceMinor := price * 100
	if balance < priceMinor {
		return nil, biz.ErrInsufficientCoins
	}

	balance, err = addBalanceTx(ctx, tx, userID, -priceMinor)
	if err != nil {
		return nil, err
	}
	title := fmt.Sprintf("兑换「%s·%s」", goodsName, skuName)
	if err := writeSequenceTx(ctx, tx, userID, accountID, title, -priceMinor, balance, "expense"); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE planstar_shop.goods_sku
		    SET stock = stock - 1, sold = sold + 1, gmt_modified = NOW()
		  WHERE sku_id = ?`, skuID); err != nil {
		return nil, err
	}

	var orderNo string
	for attempt := 0; attempt < 3; attempt++ {
		orderNo = newGoodsOrderNo(time.Now())
		_, err = tx.ExecContext(ctx,
			`INSERT INTO planstar_shop.goods_order
			   (order_no, user_id, goods_id, sku_id, sku_name, price_coins, status, gmt_create, gmt_modified)
			 VALUES (?, ?, ?, ?, ?, ?, 'paid', NOW(), NOW())`,
			orderNo, userID, goodsID, skuID, skuName, price)
		if err == nil {
			break
		}
		if !isDuplicateKey(err) {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &biz.RedeemResult{
		OrderNo: orderNo, Balance: balance, Stock: stock - 1, Sold: sold + 1,
	}, nil
}

// skusByGoods 查询商品的可售规格，返回值非 nil。
func skusByGoods(ctx context.Context, q *sql.DB, goodsID int64) ([]biz.GoodsSKU, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT sku_id, sku_name, price_coins, stock, sold
		   FROM goods_sku WHERE goods_id = ? AND status = 1 ORDER BY sku_id ASC`, goodsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skus := make([]biz.GoodsSKU, 0, 4)
	for rows.Next() {
		var s biz.GoodsSKU
		if err := rows.Scan(&s.SKUId, &s.Name, &s.PriceCoins, &s.Stock, &s.Sold); err != nil {
			return nil, err
		}
		skus = append(skus, s)
	}
	return skus, rows.Err()
}

// newGoodsOrderNo 生成 PS + 秒级时间戳 + 4 位随机数的订单号。
func newGoodsOrderNo(now time.Time) string {
	return fmt.Sprintf("PS%s%04d", now.Format("20060102150405"), rand.Intn(10000))
}
