package biz

import "context"

// CartUsecase owns the shopping cart. Prices are always re-read from the SKU
// catalogue on the server, never trusted from the client.
type CartUsecase struct {
	repo  CartRepo
	goods GoodsRepo
}

func NewCartUsecase(repo CartRepo, goods GoodsRepo) *CartUsecase {
	return &CartUsecase{repo: repo, goods: goods}
}

func (uc *CartUsecase) List(ctx context.Context, userID string) ([]*CartItem, error) {
	return uc.repo.ListByUser(ctx, userID)
}

func (uc *CartUsecase) Add(ctx context.Context, userID string, c *CartItem) (*CartItem, error) {
	if c == nil || c.SkuID == "" || c.GoodsNum <= 0 {
		return nil, ErrGoodsInvalidArgument
	}
	sku, err := uc.goods.FindSkuByID(ctx, c.SkuID)
	if err != nil {
		return nil, err
	}
	c.ID = newID()
	c.UserID = userID
	c.GoodsID = sku.GoodsID
	c.SkuID = sku.ID
	c.GoodsPrice = sku.Price
	c.GoodsSn = sku.SkuCode
	c.GoodsName = sku.SkuName
	c.Status = RowActive
	return uc.repo.Add(ctx, c)
}

func (uc *CartUsecase) Update(ctx context.Context, userID, id string, num int32, selected *bool) error {
	if id == "" || num <= 0 {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.UpdateNum(ctx, userID, id, num, selected)
}

func (uc *CartUsecase) Delete(ctx context.Context, userID, id string) error {
	if id == "" {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.Delete(ctx, userID, id)
}

// Remove deletes several cart rows after a successful order create.
func (uc *CartUsecase) Remove(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return uc.repo.DeleteMany(ctx, userID, ids)
}

// BuildOrderLines re-validates a set of cart lines against the current SKU
// prices. It returns the authoritative line items and the total in cents.
// A price change between add-to-cart and checkout is surfaced as
// ErrOrderPriceChanged rather than silently charging the stale price.
func (uc *CartUsecase) BuildOrderLines(ctx context.Context, userID string, inputs []OrderItemInput) ([]*OrderGoods, []StockItem, int64, error) {
	if len(inputs) == 0 {
		return nil, nil, 0, ErrOrderInvalidArgument
	}
	lines := make([]*OrderGoods, 0, len(inputs))
	stock := make([]StockItem, 0, len(inputs))
	var total int64
	for _, in := range inputs {
		if in.SkuID == "" || in.Num <= 0 {
			return nil, nil, 0, ErrOrderInvalidArgument
		}
		sku, err := uc.goods.FindSkuByID(ctx, in.SkuID)
		if err != nil {
			return nil, nil, 0, err
		}
		if !sku.OnSale {
			return nil, nil, 0, ErrGoodsNotFound
		}
		lineTotal := sku.Price * int64(in.Num)
		lines = append(lines, &OrderGoods{
			ID: newID(), SkuID: sku.ID, SkuName: sku.SkuName,
			SkuPrice: sku.Price, Num: in.Num, TotalPrice: lineTotal,
		})
		stock = append(stock, StockItem{SkuID: sku.ID, Num: in.Num})
		total += lineTotal
	}
	return lines, stock, total, nil
}
