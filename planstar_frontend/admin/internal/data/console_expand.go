package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"planstar/admin/internal/biz"
)

// Read-side expansion of the console: ideas, shop, payments and community.
// Every query is a whitelisted projection on a single business database; no
// query joins across databases.

const ideaColumns = `SELECT i.idea_id, i.user_id, i.title, i.summary, COALESCE(i.cover,''),
	       i.category_id, COALESCE(c.name,''), i.stage, i.publish_status, i.status,
	       i.support_count, i.oppose_count, i.coin_intent_sum, i.gmt_create`

const ideaFrom = `
	  FROM planstar_catalog.project_idea i
	  LEFT JOIN planstar_catalog.project_category c ON c.category_id = i.category_id`

const ideaSelect = ideaColumns + ideaFrom

func scanIdea(sc interface {
	Scan(dest ...any) error
}) (*biz.Idea, error) {
	idea := &biz.Idea{}
	if err := sc.Scan(&idea.ID, &idea.UserID, &idea.Title, &idea.Summary, &idea.Cover,
		&idea.CategoryID, &idea.CategoryName, &idea.Stage, &idea.PublishStatus, &idea.Status,
		&idea.SupportCount, &idea.OpposeCount, &idea.CoinIntentSum, &idea.CreatedAt); err != nil {
		return nil, err
	}
	return idea, nil
}

func (r *consoleRepo) ListIdeas(ctx context.Context, q biz.IdeaQuery) ([]*biz.Idea, error) {
	query := ideaSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(i.title LIKE ? OR i.summary LIKE ? OR CAST(i.idea_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Status != nil {
		where = append(where, "i.publish_status = ?")
		args = append(args, *q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY i.gmt_create DESC, i.idea_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ideas := make([]*biz.Idea, 0, q.Limit)
	for rows.Next() {
		idea, err := scanIdea(rows)
		if err != nil {
			return nil, err
		}
		ideas = append(ideas, idea)
	}
	return ideas, rows.Err()
}

const ideaVoteSelect = `SELECT id, idea_id, user_id, vote_type, coin_tier,
	       COALESCE(reason_code,''), COALESCE(reason_text,''), gmt_create
	  FROM planstar_catalog.project_idea_vote`

func scanIdeaVote(sc interface {
	Scan(dest ...any) error
}) (*biz.IdeaVote, error) {
	vote := &biz.IdeaVote{}
	if err := sc.Scan(&vote.ID, &vote.IdeaID, &vote.UserID, &vote.VoteType, &vote.CoinTier,
		&vote.ReasonCode, &vote.ReasonText, &vote.CreatedAt); err != nil {
		return nil, err
	}
	return vote, nil
}

func (r *consoleRepo) GetIdea(ctx context.Context, id int64) (*biz.IdeaDetail, error) {
	detail := &biz.IdeaDetail{}
	err := r.data.readDB.QueryRowContext(ctx,
		ideaColumns+", COALESCE(i.description,'')"+ideaFrom+" WHERE i.idea_id = ?", id).
		Scan(&detail.Idea.ID, &detail.Idea.UserID, &detail.Idea.Title, &detail.Idea.Summary,
			&detail.Idea.Cover, &detail.Idea.CategoryID, &detail.Idea.CategoryName,
			&detail.Idea.Stage, &detail.Idea.PublishStatus, &detail.Idea.Status,
			&detail.Idea.SupportCount, &detail.Idea.OpposeCount, &detail.Idea.CoinIntentSum,
			&detail.Idea.CreatedAt, &detail.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrIdeaNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(vote_type = 1),0), COALESCE(SUM(vote_type = 2),0)
		   FROM planstar_catalog.project_idea_vote WHERE idea_id = ?`, id).
		Scan(&detail.VotesTotal, &detail.VotesSupport, &detail.VotesOppose); err != nil {
		return nil, err
	}

	rows, err := r.data.readDB.QueryContext(ctx,
		ideaVoteSelect+" WHERE idea_id = ? ORDER BY gmt_create DESC, id DESC LIMIT 20", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		vote, err := scanIdeaVote(rows)
		if err != nil {
			return nil, err
		}
		detail.RecentVotes = append(detail.RecentVotes, vote)
	}
	return detail, rows.Err()
}

const goodsColumns = `SELECT goods_id, goods_sn, name, COALESCE(brief,''), COALESCE(cover,''),
	       COALESCE(category,''), status, sort, gmt_create`

const goodsFrom = `
	  FROM planstar_shop.goods`

const goodsSelect = goodsColumns + goodsFrom

func scanGoods(sc interface {
	Scan(dest ...any) error
}) (*biz.Goods, error) {
	goods := &biz.Goods{}
	if err := sc.Scan(&goods.ID, &goods.GoodsSN, &goods.Name, &goods.Brief, &goods.Cover,
		&goods.Category, &goods.Status, &goods.Sort, &goods.CreatedAt); err != nil {
		return nil, err
	}
	return goods, nil
}

func (r *consoleRepo) ListGoods(ctx context.Context, q biz.GoodsQuery) ([]*biz.Goods, error) {
	query := goodsSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(name LIKE ? OR goods_sn LIKE ? OR CAST(goods_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY sort ASC, goods_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goods := make([]*biz.Goods, 0, q.Limit)
	for rows.Next() {
		item, err := scanGoods(rows)
		if err != nil {
			return nil, err
		}
		goods = append(goods, item)
	}
	return goods, rows.Err()
}

func (r *consoleRepo) GetGoods(ctx context.Context, id int64) (*biz.GoodsDetail, error) {
	detail := &biz.GoodsDetail{}
	err := r.data.readDB.QueryRowContext(ctx,
		goodsColumns+", COALESCE(description,'')"+goodsFrom+" WHERE goods_id = ?", id).
		Scan(&detail.Goods.ID, &detail.Goods.GoodsSN, &detail.Goods.Name, &detail.Goods.Brief,
			&detail.Goods.Cover, &detail.Goods.Category, &detail.Goods.Status, &detail.Goods.Sort,
			&detail.Goods.CreatedAt, &detail.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrGoodsNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.data.readDB.QueryContext(ctx,
		`SELECT sku_id, goods_id, sku_name, price_coins, stock, sold, status
		   FROM planstar_shop.goods_sku WHERE goods_id = ? ORDER BY sku_id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		sku := &biz.GoodsSku{}
		if err := rows.Scan(&sku.ID, &sku.GoodsID, &sku.SkuName, &sku.PriceCoins,
			&sku.Stock, &sku.Sold, &sku.Status); err != nil {
			return nil, err
		}
		detail.Skus = append(detail.Skus, sku)
	}
	return detail, rows.Err()
}

func (r *consoleRepo) ListGoodsOrders(ctx context.Context, q biz.GoodsOrderQuery) ([]*biz.GoodsOrder, error) {
	query := `SELECT id, order_no, user_id, goods_id, sku_id, sku_name, price_coins, status, gmt_create
		  FROM planstar_shop.goods_order`
	where, args := make([]string, 0, 2), make([]any, 0, 3)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(order_no LIKE ? OR CAST(user_id AS CHAR) LIKE ?)")
		args = append(args, like, like)
	}
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY gmt_create DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*biz.GoodsOrder, 0, q.Limit)
	for rows.Next() {
		order := &biz.GoodsOrder{}
		if err := rows.Scan(&order.ID, &order.OrderNo, &order.UserID, &order.GoodsID,
			&order.SkuID, &order.SkuName, &order.PriceCoins, &order.Status, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

const payOrderSelect = `SELECT id, pay_no, order_number, order_id, user_id, channel, amount,
	       status, third_trade_no, paid_at, expire_at, gmt_create
	  FROM planstar_payment.pay_order`

func scanPayOrder(sc interface {
	Scan(dest ...any) error
}) (*biz.PayOrder, error) {
	order := &biz.PayOrder{}
	var amount float64
	var paidAt sql.NullTime
	if err := sc.Scan(&order.ID, &order.PayNo, &order.OrderNo, &order.OrderID, &order.UserID,
		&order.Channel, &amount, &order.Status, &order.ThirdTradeNo, &paidAt,
		&order.ExpireAt, &order.CreatedAt); err != nil {
		return nil, err
	}
	order.Amount = yuanToCents(amount)
	if paidAt.Valid {
		t := paidAt.Time
		order.PaidAt = &t
	}
	return order, nil
}

func (r *consoleRepo) ListPayOrders(ctx context.Context, q biz.PayOrderQuery) ([]*biz.PayOrder, error) {
	query := payOrderSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(pay_no LIKE ? OR order_number LIKE ? OR CAST(user_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY gmt_create DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*biz.PayOrder, 0, q.Limit)
	for rows.Next() {
		order, err := scanPayOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

const refundSelect = `SELECT id, refund_no, order_id, order_number, user_id, reason_code,
	       reason_text, refundable_amount, status, expected_arrival_days, gmt_create
	  FROM planstar_payment.order_refund`

func scanRefund(sc interface {
	Scan(dest ...any) error
}) (*biz.Refund, error) {
	refund := &biz.Refund{}
	var amount float64
	if err := sc.Scan(&refund.ID, &refund.RefundNo, &refund.OrderID, &refund.OrderNo,
		&refund.UserID, &refund.ReasonCode, &refund.ReasonText, &amount, &refund.Status,
		&refund.ExpectedArrivalDays, &refund.CreatedAt); err != nil {
		return nil, err
	}
	refund.RefundableAmount = yuanToCents(amount)
	return refund, nil
}

func (r *consoleRepo) ListRefunds(ctx context.Context, q biz.RefundQuery) ([]*biz.Refund, error) {
	query := refundSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(refund_no LIKE ? OR order_number LIKE ? OR CAST(user_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY gmt_create DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refunds := make([]*biz.Refund, 0, q.Limit)
	for rows.Next() {
		refund, err := scanRefund(rows)
		if err != nil {
			return nil, err
		}
		refunds = append(refunds, refund)
	}
	return refunds, rows.Err()
}

func (r *consoleRepo) GetOrder(ctx context.Context, id int64) (*biz.OrderDetail, error) {
	detail := &biz.OrderDetail{}
	var payAmount, totalAmount, refundAmount float64
	var payTime sql.NullTime
	err := r.data.readDB.QueryRowContext(ctx,
		`SELECT order_id, order_number, user_id, pay_amount, status, gmt_create,
		        COALESCE(title,''), total_amount, refund_amount, pay_time
		   FROM planstar_order.order_info WHERE order_id = ?`, id).
		Scan(&detail.Order.ID, &detail.Order.OrderNo, &detail.Order.UserID, &payAmount,
			&detail.Order.Status, &detail.Order.CreatedAt, &detail.Title,
			&totalAmount, &refundAmount, &payTime)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	detail.Order.Amount = yuanToCents(payAmount)
	detail.TotalAmount = yuanToCents(totalAmount)
	detail.RefundAmount = yuanToCents(refundAmount)
	if payTime.Valid {
		t := payTime.Time
		detail.PayTime = &t
	}

	payRows, err := r.data.readDB.QueryContext(ctx,
		payOrderSelect+" WHERE order_id = ? ORDER BY gmt_create DESC, id DESC LIMIT 50", id)
	if err != nil {
		return nil, err
	}
	defer payRows.Close()
	for payRows.Next() {
		order, err := scanPayOrder(payRows)
		if err != nil {
			return nil, err
		}
		detail.PayOrders = append(detail.PayOrders, order)
	}
	if err := payRows.Err(); err != nil {
		return nil, err
	}

	refundRows, err := r.data.readDB.QueryContext(ctx,
		refundSelect+" WHERE order_id = ? ORDER BY gmt_create DESC, id DESC LIMIT 50", id)
	if err != nil {
		return nil, err
	}
	defer refundRows.Close()
	for refundRows.Next() {
		refund, err := scanRefund(refundRows)
		if err != nil {
			return nil, err
		}
		detail.Refunds = append(detail.Refunds, refund)
	}
	return detail, refundRows.Err()
}

const feedColumns = `SELECT feed_id, type, user_id, COALESCE(title,''), COALESCE(content,''),
	       likes, reply_count, views, status, is_quality, gmt_create`

const feedFrom = `
	  FROM planstar_community.feed`

const feedSelect = feedColumns + feedFrom

func scanFeed(sc interface {
	Scan(dest ...any) error
}) (*biz.Feed, error) {
	feed := &biz.Feed{}
	if err := sc.Scan(&feed.ID, &feed.Type, &feed.UserID, &feed.Title, &feed.Content,
		&feed.Likes, &feed.ReplyCount, &feed.Views, &feed.Status, &feed.IsQuality,
		&feed.CreatedAt); err != nil {
		return nil, err
	}
	return feed, nil
}

func (r *consoleRepo) ListFeeds(ctx context.Context, q biz.FeedQuery) ([]*biz.Feed, error) {
	query := feedSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(title LIKE ? OR content LIKE ? OR CAST(feed_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Status != nil {
		// 1 = normal; 0 = hidden, which is any status other than 1.
		if *q.Status == 1 {
			where = append(where, "status = 1")
		} else {
			where = append(where, "status <> 1")
		}
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY gmt_create DESC, feed_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make([]*biz.Feed, 0, q.Limit)
	for rows.Next() {
		feed, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, rows.Err()
}

func (r *consoleRepo) GetFeed(ctx context.Context, id int64) (*biz.FeedDetail, error) {
	detail := &biz.FeedDetail{}
	var images, videos, tags, user string
	err := r.data.readDB.QueryRowContext(ctx,
		feedColumns+`, COALESCE(images,'[]'), COALESCE(videos,'[]'), COALESCE(tags,'[]'), COALESCE(user,'{}')`+feedFrom+`
		  WHERE feed_id = ?`, id).
		Scan(&detail.Feed.ID, &detail.Feed.Type, &detail.Feed.UserID, &detail.Feed.Title,
			&detail.Feed.Content, &detail.Feed.Likes, &detail.Feed.ReplyCount, &detail.Feed.Views,
			&detail.Feed.Status, &detail.Feed.IsQuality, &detail.Feed.CreatedAt,
			&images, &videos, &tags, &user)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrFeedNotFound
	}
	if err != nil {
		return nil, err
	}
	detail.Images = decodeStringSlice(images)
	detail.Videos = decodeStringSlice(videos)
	detail.Tags = decodeStringSlice(tags)
	var snapshot struct {
		UserName string `json:"userName"`
	}
	if err := json.Unmarshal([]byte(user), &snapshot); err == nil {
		detail.AuthorName = snapshot.UserName
	}
	return detail, nil
}