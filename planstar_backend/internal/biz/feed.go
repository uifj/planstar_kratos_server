package biz

import "context"

// FeedQuery 动态列表查询参数（page/pageSize/nextPageKey 为动态族分页）。
type FeedQuery struct {
	Page        int
	PageSize    int
	NextPageKey int64
}

// FeedRepo 是动态仓储端口。
type FeedRepo interface {
	Feeds(ctx context.Context, q FeedQuery, hot bool) ([]*FeedPost, bool, int64, error)
	Feed(ctx context.Context, id int64) (*FeedPost, error)
	Comments(ctx context.Context, tweetID int64) ([]*FeedComment, error)
}

// FeedUsecase 动态用例。
type FeedUsecase struct {
	repo FeedRepo
}

func NewFeedUsecase(repo FeedRepo) *FeedUsecase {
	return &FeedUsecase{repo: repo}
}

func (uc *FeedUsecase) Feeds(ctx context.Context, q FeedQuery, hot bool) (*FeedPage, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	items, hasMore, nextKey, err := uc.repo.Feeds(ctx, q, hot)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*FeedPost{}
	}
	return &FeedPage{Data: items, HasMore: hasMore, NextPageKey: nextKey}, nil
}

func (uc *FeedUsecase) Feed(ctx context.Context, id int64) (*FeedPost, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.Feed(ctx, id)
}

func (uc *FeedUsecase) Comments(ctx context.Context, tweetID int64) ([]*FeedComment, error) {
	if tweetID <= 0 {
		return nil, ErrBadRequest
	}
	items, err := uc.repo.Comments(ctx, tweetID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*FeedComment{}
	}
	return items, nil
}
