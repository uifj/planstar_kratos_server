package biz

import "context"

// ActivityQuery 演出库列表查询参数（page/pageSize 为演出族分页）。
type ActivityQuery struct {
	Page       int
	PageSize   int
	CategoryID int64
	City       string
}

// CatalogRepo 是演出库仓储端口。
type CatalogRepo interface {
	Banners(ctx context.Context) ([]*Banner, error)
	Activities(ctx context.Context, q ActivityQuery, hot bool) ([]*ShowSummary, error)
	Activity(ctx context.Context, id int64) (*ShowSummary, error)
}

// CatalogUsecase 演出库用例。
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
