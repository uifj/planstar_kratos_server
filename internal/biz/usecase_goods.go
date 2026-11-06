package biz

import (
	"context"
	"strings"
)

// GoodsUsecase owns the catalogue, the Elasticsearch read path and the admin
// catalogue CRUD. It also implements GoodsEventHandler.HandleOrderPaid.
type GoodsUsecase struct {
	repo      GoodsRepo
	search    GoodsSearch
	inventory InventoryRepo
	stock     StockCache
}

func NewGoodsUsecase(repo GoodsRepo, search GoodsSearch, inventory InventoryRepo, stock StockCache) *GoodsUsecase {
	return &GoodsUsecase{repo: repo, search: search, inventory: inventory, stock: stock}
}

// EnsureIndex creates the ES index if missing and reindexes every active
// goods row. Called once at application startup.
func (uc *GoodsUsecase) EnsureIndex(ctx context.Context) error {
	if err := uc.search.EnsureIndex(ctx); err != nil {
		return err
	}
	goods, err := uc.repo.AllForIndex(ctx)
	if err != nil {
		return err
	}
	for _, g := range goods {
		if err := uc.search.Index(ctx, uc.document(g)); err != nil {
			return err
		}
	}
	return nil
}

// List searches Elasticsearch for matching goods ids and hydrates them from
// MySQL. If ES is unavailable *or the index is not populated yet* (for
// example when demo data is seeded after the app has booted), the repo's SQL
// filter is used as a fallback so the API still works.
func (uc *GoodsUsecase) List(ctx context.Context, q GoodsQuery) ([]*Goods, int32, error) {
	q.Page, q.PageSize = pageBounds(q.Page, q.PageSize)
	q.OnlyOnSale = true
	ids, total, err := uc.search.Search(ctx, q)
	if err == nil && len(ids) > 0 {
		q.IDs = ids
		goods, _, err := uc.repo.List(ctx, q)
		if err != nil {
			return nil, 0, err
		}
		return goods, total, nil
	}
	// Empty ES result is not authoritative: fall back to SQL, which does its
	// own on_sale/category/brand filtering and pagination.
	q.IDs = nil
	return uc.repo.List(ctx, q)
}

// AdminList uses the SQL filter directly (includes off-sale goods).
func (uc *GoodsUsecase) AdminList(ctx context.Context, q GoodsQuery) ([]*Goods, int32, error) {
	q.Page, q.PageSize = pageBounds(q.Page, q.PageSize)
	q.IDs = nil
	return uc.repo.List(ctx, q)
}

func (uc *GoodsUsecase) Detail(ctx context.Context, id string) (*Goods, error) {
	if id == "" {
		return nil, ErrGoodsInvalidArgument
	}
	return uc.repo.FindByID(ctx, id)
}

func (uc *GoodsUsecase) Create(ctx context.Context, g *Goods) error {
	if g == nil || strings.TrimSpace(g.Name) == "" || g.GoodsSn == "" {
		return ErrGoodsInvalidArgument
	}
	g.ID = newID()
	g.Status = RowActive
	// The aggregate owns its children's identity and lifecycle defaults. The
	// data layer writes whatever ids it is given, so leave none at the zero
	// value: an empty id would defeat the UUIDv7 default and a zero status
	// would hide the row from every active-only read.
	for _, sku := range g.Skus {
		if sku.ID == "" {
			sku.ID = newID()
		}
		if sku.SkuCode == "" {
			sku.SkuCode = "SKU" + randomSuffix(6)
		}
		if sku.Status == 0 {
			sku.Status = RowActive
		}
		sku.GoodsID = g.ID
	}
	for _, im := range g.Images {
		if im.ID == "" {
			im.ID = newID()
		}
		if im.Status == 0 {
			im.Status = RowActive
		}
		im.GoodsID = g.ID
	}
	created, err := uc.repo.Create(ctx, g)
	if err != nil {
		return err
	}
	// Elasticsearch is a read-side projection: a failed write must never roll
	// back a committed SQL transaction. Startup re-indexing plus the SQL
	// fallback in List keep the index eventually consistent when ES is down.
	_ = uc.search.Index(ctx, uc.document(created))
	// Seed one inventory row per SKU so TryLock can take a row lock.
	for _, sku := range created.Skus {
		if err := uc.inventory.EnsureSku(ctx, sku.ID, sku.Inventory); err != nil {
			return err
		}
	}
	return nil
}

func (uc *GoodsUsecase) Update(ctx context.Context, g *Goods) error {
	if g == nil || g.ID == "" {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.Update(ctx, g)
}

func (uc *GoodsUsecase) SetOnSale(ctx context.Context, id string, onSale bool) error {
	if id == "" {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.SetOnSale(ctx, id, onSale)
}

func (uc *GoodsUsecase) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *GoodsUsecase) Categories(ctx context.Context) ([]*Category, error) {
	return uc.repo.Categories(ctx)
}

func (uc *GoodsUsecase) Brands(ctx context.Context, page, pageSize int) ([]*Brand, error) {
	page, pageSize = pageBounds(page, pageSize)
	return uc.repo.Brands(ctx, page, pageSize)
}

func (uc *GoodsUsecase) CreateCategory(ctx context.Context, c *Category) (*Category, error) {
	if c == nil || strings.TrimSpace(c.Name) == "" {
		return nil, ErrGoodsInvalidArgument
	}
	c.ID = newID()
	c.Status = RowActive
	return uc.repo.CreateCategory(ctx, c)
}

func (uc *GoodsUsecase) UpdateCategory(ctx context.Context, c *Category) error {
	if c == nil || c.ID == "" {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.UpdateCategory(ctx, c)
}

func (uc *GoodsUsecase) DeleteCategory(ctx context.Context, id string) error {
	if id == "" {
		return ErrGoodsInvalidArgument
	}
	return uc.repo.DeleteCategory(ctx, id)
}

// HandleOrderPaid increments sold_num, refreshes the ES document and drops
// the cached stock value. It is called by the order.paid consumer, which is
// idempotent via consumed_event.
func (uc *GoodsUsecase) HandleOrderPaid(ctx context.Context, event *DomainEvent) error {
	for _, item := range event.Items {
		if err := uc.repo.IncrSoldNum(ctx, item.SkuID, int64(item.Num)); err != nil {
			return err
		}
		// Best-effort read-side update, same reasoning as Create above.
		_ = uc.search.UpdateSoldNum(ctx, item.SkuID, int64(item.Num))
		_ = uc.stock.Invalidate(ctx, item.SkuID)
	}
	return nil
}

func (uc *GoodsUsecase) document(g *Goods) *GoodsDocument {
	d := &GoodsDocument{
		ID: g.ID, Name: g.Name, GoodsSn: g.GoodsSn,
		MarketPrice: g.MarketPrice, OnSale: g.OnSale,
		IsNew: g.IsNew, IsHot: g.IsHot,
		CategoryID: g.CategoryID, BrandID: g.BrandID,
		CategoryName: g.CategoryName, BrandName: g.BrandName,
		SoldNum: g.SoldNum,
	}
	for _, sku := range g.Skus {
		d.SkuCodes = append(d.SkuCodes, sku.SkuCode)
		d.SkuIDs = append(d.SkuIDs, sku.ID)
	}
	return d
}
