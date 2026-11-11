package biz

import "context"

// ActivityQuery 企划库列表查询参数（page/pageSize 为企划族分页）。
type ActivityQuery struct {
	Page       int
	PageSize   int
	CategoryID int64
	City       string
}

// CatalogRepo 是企划库仓储端口（读 legacy catalog.activity，见 POSITIONING §12）。
type CatalogRepo interface {
	Banners(ctx context.Context) ([]*Banner, error)
	Activities(ctx context.Context, q ActivityQuery, hot bool) ([]*ShowSummary, error)
	Activity(ctx context.Context, id int64) (*ShowSummary, error)
}

// ActivitySnapshot 是下单/报价所需的企划权威快照（价格单位为分）。
type ActivitySnapshot struct {
	ActivityID       int64
	TicketCategoryID int64
	Title            string
	Poster           string
	UnitPrice        int64
	SoldOut          bool
}

// CatalogReader 是企划权威数据的只读端口，由 data 层用 catalog / ticket 库连接实现。
// 订单链路不得在 order 库连接上跨库查询。
type CatalogReader interface {
	GetActivitySnapshot(ctx context.Context, activityID int64) (*ActivitySnapshot, error)
}

// CatalogUsecase 企划库用例。
type CatalogUsecase struct {
	repo CatalogRepo
}

func NewCatalogUsecase(repo CatalogRepo) *CatalogUsecase {
	return &CatalogUsecase{repo: repo}
}

func (uc *CatalogUsecase) Banners(ctx context.Context) ([]*Banner, error) {
	return uc.repo.Banners(ctx)
}

func (uc *CatalogUsecase) Activities(ctx context.Context, q ActivityQuery, hot bool) ([]*ShowSummary, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return uc.repo.Activities(ctx, q, hot)
}

func (uc *CatalogUsecase) Activity(ctx context.Context, id int64) (*ShowSummary, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.Activity(ctx, id)
}
