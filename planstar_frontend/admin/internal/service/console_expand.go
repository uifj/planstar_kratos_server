package service

import (
	"context"

	"go.einride.tech/aip/pagination"

	consolev1 "planstar/admin/api/planstar/admin/v1"
	"planstar/admin/internal/biz"
)

// Read-side expansion handlers. All are viewer-gated reads; none write.

func convertIdea(idea *biz.Idea) *consolev1.Idea {
	return &consolev1.Idea{
		Id:            idea.ID,
		UserId:        idea.UserID,
		Title:         idea.Title,
		Summary:       idea.Summary,
		Cover:         idea.Cover,
		CategoryId:    idea.CategoryID,
		CategoryName:  idea.CategoryName,
		Stage:         idea.Stage,
		PublishStatus: idea.PublishStatus,
		Status:        idea.Status,
		SupportCount:  idea.SupportCount,
		OpposeCount:   idea.OpposeCount,
		CoinIntentSum: idea.CoinIntentSum,
		CreatedAt:     timestamp(idea.CreatedAt),
	}
}

func convertIdeaVote(vote *biz.IdeaVote) *consolev1.IdeaVote {
	return &consolev1.IdeaVote{
		Id:         vote.ID,
		IdeaId:     vote.IdeaID,
		UserId:     vote.UserID,
		VoteType:   vote.VoteType,
		CoinTier:   vote.CoinTier,
		ReasonCode: vote.ReasonCode,
		ReasonText: vote.ReasonText,
		CreatedAt:  timestamp(vote.CreatedAt),
	}
}

func convertGoods(goods *biz.Goods) *consolev1.Goods {
	return &consolev1.Goods{
		Id:        goods.ID,
		GoodsSn:   goods.GoodsSN,
		Name:      goods.Name,
		Brief:     goods.Brief,
		Cover:     goods.Cover,
		Category:  goods.Category,
		Status:    goods.Status,
		Sort:      goods.Sort,
		CreatedAt: timestamp(goods.CreatedAt),
	}
}

func convertGoodsSku(sku *biz.GoodsSku) *consolev1.GoodsSku {
	return &consolev1.GoodsSku{
		Id:         sku.ID,
		GoodsId:    sku.GoodsID,
		SkuName:    sku.SkuName,
		PriceCoins: sku.PriceCoins,
		Stock:      sku.Stock,
		Sold:       sku.Sold,
		Status:     sku.Status,
	}
}

func convertGoodsOrder(order *biz.GoodsOrder) *consolev1.GoodsOrder {
	return &consolev1.GoodsOrder{
		Id:         order.ID,
		OrderNo:    order.OrderNo,
		UserId:     order.UserID,
		GoodsId:    order.GoodsID,
		SkuId:      order.SkuID,
		SkuName:    order.SkuName,
		PriceCoins: order.PriceCoins,
		Status:     order.Status,
		CreatedAt:  timestamp(order.CreatedAt),
	}
}

func convertPayOrder(order *biz.PayOrder) *consolev1.PayOrder {
	out := &consolev1.PayOrder{
		Id:           order.ID,
		PayNo:        order.PayNo,
		OrderNo:      order.OrderNo,
		OrderId:      order.OrderID,
		UserId:       order.UserID,
		Channel:      order.Channel,
		Amount:       order.Amount,
		Status:       order.Status,
		ThirdTradeNo: order.ThirdTradeNo,
		ExpireAt:     timestamp(order.ExpireAt),
		CreatedAt:    timestamp(order.CreatedAt),
	}
	if order.PaidAt != nil {
		out.PaidAt = timestamp(*order.PaidAt)
	}
	return out
}

func convertRefund(refund *biz.Refund) *consolev1.Refund {
	return &consolev1.Refund{
		Id:                  refund.ID,
		RefundNo:            refund.RefundNo,
		OrderId:             refund.OrderID,
		OrderNo:             refund.OrderNo,
		UserId:              refund.UserID,
		ReasonCode:          refund.ReasonCode,
		ReasonText:          refund.ReasonText,
		RefundableAmount:    refund.RefundableAmount,
		Status:              refund.Status,
		ExpectedArrivalDays: refund.ExpectedArrivalDays,
		CreatedAt:           timestamp(refund.CreatedAt),
	}
}

func convertFeed(feed *biz.Feed) *consolev1.Feed {
	return &consolev1.Feed{
		Id:         feed.ID,
		Type:       feed.Type,
		UserId:     feed.UserID,
		Title:      feed.Title,
		Content:    feed.Content,
		Likes:      feed.Likes,
		ReplyCount: feed.ReplyCount,
		Views:      feed.Views,
		Status:     feed.Status,
		IsQuality:  feed.IsQuality,
		CreatedAt:  timestamp(feed.CreatedAt),
	}
}

// ListIdeas lists community ideas.
func (s *ConsoleService) ListIdeas(ctx context.Context, req *consolev1.ListIdeasRequest) (*consolev1.IdeaSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	ideas, err := s.uc.ListIdeas(ctx, biz.IdeaQuery{
		Keyword: req.Keyword,
		Status:  req.PublishStatus,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.IdeaSet{Ideas: make([]*consolev1.Idea, 0, len(ideas))}
	if len(ideas) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, idea := range ideas {
		set.Ideas = append(set.Ideas, convertIdea(idea))
	}
	return set, nil
}

// GetIdea returns one idea with its vote aggregation and recent votes.
func (s *ConsoleService) GetIdea(ctx context.Context, req *consolev1.GetIdeaRequest) (*consolev1.IdeaDetail, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	detail, err := s.uc.GetIdea(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &consolev1.IdeaDetail{
		Idea:         convertIdea(&detail.Idea),
		Description:  detail.Description,
		VotesTotal:   detail.VotesTotal,
		VotesSupport: detail.VotesSupport,
		VotesOppose:  detail.VotesOppose,
		RecentVotes:  make([]*consolev1.IdeaVote, 0, len(detail.RecentVotes)),
	}
	for _, vote := range detail.RecentVotes {
		out.RecentVotes = append(out.RecentVotes, convertIdeaVote(vote))
	}
	return out, nil
}

// ListGoods lists shop goods.
func (s *ConsoleService) ListGoods(ctx context.Context, req *consolev1.ListGoodsRequest) (*consolev1.GoodsSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	goods, err := s.uc.ListGoods(ctx, biz.GoodsQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.GoodsSet{Goods: make([]*consolev1.Goods, 0, len(goods))}
	if len(goods) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, item := range goods {
		set.Goods = append(set.Goods, convertGoods(item))
	}
	return set, nil
}

// GetGoods returns one goods item with its SKUs.
func (s *ConsoleService) GetGoods(ctx context.Context, req *consolev1.GetGoodsRequest) (*consolev1.GoodsDetail, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	detail, err := s.uc.GetGoods(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &consolev1.GoodsDetail{
		Goods:       convertGoods(&detail.Goods),
		Description: detail.Description,
		Skus:        make([]*consolev1.GoodsSku, 0, len(detail.Skus)),
	}
	for _, sku := range detail.Skus {
		out.Skus = append(out.Skus, convertGoodsSku(sku))
	}
	return out, nil
}

// ListGoodsOrders lists coin purchases.
func (s *ConsoleService) ListGoodsOrders(ctx context.Context, req *consolev1.ListGoodsOrdersRequest) (*consolev1.GoodsOrderSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	orders, err := s.uc.ListGoodsOrders(ctx, biz.GoodsOrderQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.GoodsOrderSet{Orders: make([]*consolev1.GoodsOrder, 0, len(orders))}
	if len(orders) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, order := range orders {
		set.Orders = append(set.Orders, convertGoodsOrder(order))
	}
	return set, nil
}

// GetOrder returns one unified order with its payments and refunds.
func (s *ConsoleService) GetOrder(ctx context.Context, req *consolev1.GetOrderRequest) (*consolev1.OrderDetail, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	detail, err := s.uc.GetOrder(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &consolev1.OrderDetail{
		Order:        convertConsoleOrder(&detail.Order),
		Title:        detail.Title,
		TotalAmount:  detail.TotalAmount,
		RefundAmount: detail.RefundAmount,
		PayOrders:    make([]*consolev1.PayOrder, 0, len(detail.PayOrders)),
		Refunds:      make([]*consolev1.Refund, 0, len(detail.Refunds)),
	}
	if detail.PayTime != nil {
		out.PayTime = timestamp(*detail.PayTime)
	}
	for _, order := range detail.PayOrders {
		out.PayOrders = append(out.PayOrders, convertPayOrder(order))
	}
	for _, refund := range detail.Refunds {
		out.Refunds = append(out.Refunds, convertRefund(refund))
	}
	return out, nil
}

// ListPayOrders lists payment attempts.
func (s *ConsoleService) ListPayOrders(ctx context.Context, req *consolev1.ListPayOrdersRequest) (*consolev1.PayOrderSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	orders, err := s.uc.ListPayOrders(ctx, biz.PayOrderQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.PayOrderSet{PayOrders: make([]*consolev1.PayOrder, 0, len(orders))}
	if len(orders) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, order := range orders {
		set.PayOrders = append(set.PayOrders, convertPayOrder(order))
	}
	return set, nil
}

// ListRefunds lists refund requests.
func (s *ConsoleService) ListRefunds(ctx context.Context, req *consolev1.ListRefundsRequest) (*consolev1.RefundSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	refunds, err := s.uc.ListRefunds(ctx, biz.RefundQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.RefundSet{Refunds: make([]*consolev1.Refund, 0, len(refunds))}
	if len(refunds) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, refund := range refunds {
		set.Refunds = append(set.Refunds, convertRefund(refund))
	}
	return set, nil
}

// ListFeeds lists community feeds.
func (s *ConsoleService) ListFeeds(ctx context.Context, req *consolev1.ListFeedsRequest) (*consolev1.FeedSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	feeds, err := s.uc.ListFeeds(ctx, biz.FeedQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.FeedSet{Feeds: make([]*consolev1.Feed, 0, len(feeds))}
	if len(feeds) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, feed := range feeds {
		set.Feeds = append(set.Feeds, convertFeed(feed))
	}
	return set, nil
}

// GetFeed returns one feed with its media and author snapshot.
func (s *ConsoleService) GetFeed(ctx context.Context, req *consolev1.GetFeedRequest) (*consolev1.FeedDetail, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	detail, err := s.uc.GetFeed(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &consolev1.FeedDetail{
		Feed:       convertFeed(&detail.Feed),
		Images:     detail.Images,
		Videos:     detail.Videos,
		Tags:       detail.Tags,
		AuthorName: detail.AuthorName,
	}, nil
}