package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"planstar/internal/biz"
)

// quoteKeyPrefix 是 Redis 中报价快照的键前缀。
const quoteKeyPrefix = "planstar:quote:"

// OrderRepo 实现 biz.OrderRepo，落 planstar_order.order_info。
// 同时实现 biz.QuoteStore（Redis / 进程内 TTL）与 biz.CatalogReader（catalog / ticket 库）。
type OrderRepo struct {
	d *Data

	quoteMu sync.Mutex
	quotes  map[string]*biz.QuoteSnapshot
}

func NewOrderRepo(d *Data) *OrderRepo {
	return &OrderRepo{d: d, quotes: make(map[string]*biz.QuoteSnapshot)}
}

func (r *OrderRepo) Orders(ctx context.Context, userID int64, status, orderType, pageIndex, pageSize int) ([]*biz.TicketOrder, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	where := "user_id = ?"
	args := []any{userID}
	if status > 0 {
		where += " AND status = ?"
		args = append(args, status)
	}
	if orderType > 0 {
		where += " AND order_type = ?"
		args = append(args, orderType)
	}
	args = append(args, pageSize, (pageIndex-1)*pageSize)

	rows, err := db.QueryContext(ctx,
		`SELECT order_id, order_number, COALESCE(activity_id,0), COALESCE(activity_event_id,0),
		        COALESCE(order_type,0), COALESCE(status,0), COALESCE(title,''), COALESCE(poster,''),
		        COALESCE(pay_amount,0), gmt_create
		   FROM order_info WHERE `+where+`
		ORDER BY order_id DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.TicketOrder, 0, pageSize)
	for rows.Next() {
		o := &biz.TicketOrder{}
		var payAmount float64
		var created time.Time
		if err := rows.Scan(&o.OrderID, &o.OrderNumber, &o.ShowID, &o.SessionID, &o.OrderType, &o.Status,
			&o.Title, &o.Poster, &payAmount, &created); err != nil {
			return nil, err
		}
		o.ID = o.OrderID
		o.BuyType = 1
		o.CreateTime = biz.FmtDT(created)
		o.PayAmount = biz.Fen(payAmount)
		items = append(items, o)
	}
	return items, rows.Err()
}

func (r *OrderRepo) OrderByNumber(ctx context.Context, userID int64, orderNumber string) (*biz.TicketOrder, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	o := &biz.TicketOrder{}
	var payAmount float64
	var created time.Time
	err = db.QueryRowContext(ctx,
		`SELECT order_id, order_number, COALESCE(activity_id,0), COALESCE(activity_event_id,0),
		        COALESCE(order_type,0), COALESCE(status,0), COALESCE(title,''), COALESCE(poster,''),
		        COALESCE(pay_amount,0), gmt_create
		   FROM order_info WHERE order_number = ? AND user_id = ? LIMIT 1`, orderNumber, userID).
		Scan(&o.OrderID, &o.OrderNumber, &o.ShowID, &o.SessionID, &o.OrderType, &o.Status,
			&o.Title, &o.Poster, &payAmount, &created)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	o.ID = o.OrderID
	o.BuyType = 1
	o.CreateTime = biz.FmtDT(created)
	o.PayAmount = biz.Fen(payAmount)
	return o, nil
}

// GetActivitySnapshot 从 catalog / ticket 库读取报价所需的权威数据。
func (r *OrderRepo) GetActivitySnapshot(ctx context.Context, activityID int64) (*biz.ActivitySnapshot, error) {
	if activityID <= 0 {
		return nil, biz.ErrBadRequest
	}
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	snap := &biz.ActivitySnapshot{ActivityID: activityID}
	var (
		lowPrice      float64
		status        int
		publishStatus int
		soldOut       int
	)
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(name,''), COALESCE(poster, img, ''), low_price, status, publish_status, is_sold_out
		   FROM activity WHERE activity_id = ?`, activityID).
		Scan(&snap.Title, &snap.Poster, &lowPrice, &status, &publishStatus, &soldOut)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	snap.SoldOut = soldOut > 0 || status != 1 || publishStatus != 1

	// 优先使用回报档位/票组（legacy planstar_ticket 库）的在售价格，取最低价 SKU。
	// SKU 级定价待接入：目前按企划聚合最低回报档位，未细分线下回报场次与档位售卖策略。
	if tdb, terr := r.d.DB("ticket"); terr == nil {
		var categoryID int64
		var salePrice float64
		qerr := tdb.QueryRowContext(ctx,
			`SELECT tc.ticket_category_id, COALESCE(MIN(tg.sale_price), tc.low_price, 0)
			   FROM ticket_category tc
			   LEFT JOIN ticket_group tg
			          ON tg.ticket_category_id = tc.ticket_category_id AND tg.is_saling = 1
			  WHERE tc.activity_id = ? AND tc.status = 1
			  GROUP BY tc.ticket_category_id, tc.low_price
			  ORDER BY 2 ASC, tc.ticket_category_id ASC
			  LIMIT 1`, activityID).Scan(&categoryID, &salePrice)
		if qerr == nil && salePrice > 0 {
			snap.TicketCategoryID = categoryID
			snap.UnitPrice = biz.Fen(salePrice)
		}
	}
	if snap.UnitPrice <= 0 {
		// SKU 级定价待接入：回报档位表无可用售价时回退 catalog.activity.low_price。
		snap.UnitPrice = biz.Fen(lowPrice)
	}
	return snap, nil
}

// SaveQuote 持久化报价快照：优先 Redis，未配置或写入失败时退化为进程内 TTL 缓存。
func (r *OrderRepo) SaveQuote(ctx context.Context, snap *biz.QuoteSnapshot) error {
	if snap == nil || snap.QuoteID == "" {
		return biz.ErrBadRequest
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	ttl := time.Until(time.UnixMilli(snap.ExpiresAt))
	if ttl <= 0 {
		ttl = biz.QuoteTTLFromEnv()
	}
	if rdb := r.d.Redis(); rdb != nil {
		if err := rdb.Set(ctx, quoteKeyPrefix+snap.QuoteID, b, ttl).Err(); err == nil {
			return nil
		} else if r.d.log != nil {
			r.d.log.WarnContext(ctx, "redis save quote failed, fallback to memory", "err", err)
		}
	}
	// 单机语义：进程内 map 仅对当前实例可见，多实例部署必须使用 Redis。
	r.quoteMu.Lock()
	defer r.quoteMu.Unlock()
	now := time.Now().UnixMilli()
	for k, v := range r.quotes {
		if v.ExpiresAt <= now {
			delete(r.quotes, k)
		}
	}
	r.quotes[snap.QuoteID] = snap
	return nil
}

// GetQuote 读取报价快照；不存在或已过期统一返回 biz.ErrQuoteExpired。
func (r *OrderRepo) GetQuote(ctx context.Context, quoteID string) (*biz.QuoteSnapshot, error) {
	if quoteID == "" {
		return nil, biz.ErrQuoteExpired
	}
	if rdb := r.d.Redis(); rdb != nil {
		b, err := rdb.Get(ctx, quoteKeyPrefix+quoteID).Bytes()
		switch {
		case err == nil:
			snap := &biz.QuoteSnapshot{}
			if json.Unmarshal(b, snap) != nil {
				return nil, biz.ErrQuoteExpired
			}
			return snap, nil
		case errors.Is(err, redis.Nil):
			// Redis 无此键时继续查进程内缓存（Redis 写入失败后的降级路径）。
		default:
			if r.d.log != nil {
				r.d.log.WarnContext(ctx, "redis get quote failed, fallback to memory", "err", err)
			}
		}
	}
	r.quoteMu.Lock()
	defer r.quoteMu.Unlock()
	snap, ok := r.quotes[quoteID]
	if !ok {
		return nil, biz.ErrQuoteExpired
	}
	if snap.ExpiresAt > 0 && snap.ExpiresAt <= time.Now().UnixMilli() {
		delete(r.quotes, quoteID)
		return nil, biz.ErrQuoteExpired
	}
	return snap, nil
}

// CreateOrder 单事务完成：幂等校验 → 库存扣减 → 写订单 → 写幂等记录 → 提交。
// 库存与订单在同一 MySQL 实例上以跨 schema DML 保证原子性（planstar_ticket.ticket_category）。
func (r *OrderRepo) CreateOrder(ctx context.Context, userID int64, in *biz.OrderInput, quote *biz.QuoteSnapshot) (*biz.TicketOrder, error) {
	if quote == nil {
		return nil, biz.ErrPriceChanged
	}
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// 幂等：同 key 已存在时直接返回已有订单，不重复扣库存。
	if in.IdempotencyKey != "" {
		existing, err := orderByIdemKey(ctx, tx, userID, in.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	// 库存扣减：影响行数为 0 说明库存不足，整单回滚。
	if quote.TicketCategoryID > 0 {
		res, err := tx.ExecContext(ctx,
			`UPDATE planstar_ticket.ticket_category
			    SET stock = stock - ?, gmt_modified = NOW()
			  WHERE ticket_category_id = ? AND stock >= ?`,
			quote.Quantity, quote.TicketCategoryID, quote.Quantity)
		if err != nil {
			return nil, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected == 0 {
			return nil, biz.ErrSoldOut
		}
	}

	now := time.Now()
	var (
		orderID     int64
		orderNumber string
	)
	for attempt := 0; attempt < 3; attempt++ {
		orderID = newOrderID()
		orderNumber = newOrderNumber(now)
		_, err = tx.ExecContext(ctx,
			`INSERT INTO order_info
			   (order_id, order_number, user_id, activity_id, activity_event_id, order_type, status,
			    title, poster, pay_amount, total_amount, name, phone, user_address_id, gmt_create, gmt_modified)
			 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			orderID, orderNumber, userID, in.ActivityID, in.ActivityEventID, in.OrderType,
			quote.Title, quote.Poster, fenToYuan(quote.TotalAmount), fenToYuan(quote.TotalAmount),
			in.Name, in.Phone, in.AddressID)
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

	if in.IdempotencyKey != "" {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO order_idempotency (idem_key, user_id, order_id, order_number, gmt_create)
			 VALUES (?, ?, ?, ?, NOW())`,
			in.IdempotencyKey, userID, orderID, orderNumber)
		if err != nil {
			// 并发同 key：回滚后返回先落库的订单。
			if isDuplicateKey(err) {
				_ = tx.Rollback()
				existing, qerr := orderByIdemKey(ctx, db, userID, in.IdempotencyKey)
				if qerr == nil && existing != nil {
					return existing, nil
				}
			}
			return nil, err
		}
	}

	// 报价快照一次性消费：同一 quoteId 只能落一单。
	// 防止 15 分钟报价有效期内换一个客户端幂等键重复下单 / 重复扣库存。
	// 冲突时回滚整个事务（库存扣减一并撤销）并返回首次创建的订单。
	if quote.QuoteID != "" {
		quoteKey := "quote:" + quote.QuoteID
		_, err = tx.ExecContext(ctx,
			`INSERT INTO order_idempotency (idem_key, user_id, order_id, order_number, gmt_create)
			 VALUES (?, ?, ?, ?, NOW())`,
			quoteKey, userID, orderID, orderNumber)
		if err != nil {
			if isDuplicateKey(err) {
				_ = tx.Rollback()
				existing, qerr := orderByIdemKey(ctx, db, userID, quoteKey)
				if qerr == nil && existing != nil {
					return existing, nil
				}
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &biz.TicketOrder{
		OrderID: orderID, ID: orderID, OrderNumber: orderNumber,
		ShowID: in.ActivityID, SessionID: in.ActivityEventID,
		CreateTime: biz.FmtDT(now), OrderType: in.OrderType, BuyType: 1, Status: 1,
		Title: quote.Title, Poster: quote.Poster, PayAmount: quote.TotalAmount,
	}, nil
}

// rowQuerier 同时适配 *sql.DB 与 *sql.Tx。
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// orderByIdemKey 按 (user_id, idem_key) 查已有订单；不存在返回 (nil, nil)。
func orderByIdemKey(ctx context.Context, q rowQuerier, userID int64, key string) (*biz.TicketOrder, error) {
	var orderID int64
	err := q.QueryRowContext(ctx,
		`SELECT order_id FROM order_idempotency WHERE user_id = ? AND idem_key = ? LIMIT 1`,
		userID, key).Scan(&orderID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return orderByID(ctx, q, userID, orderID)
}

// orderByID 读取订单详情（仅限当前用户）。
func orderByID(ctx context.Context, q rowQuerier, userID, orderID int64) (*biz.TicketOrder, error) {
	o := &biz.TicketOrder{}
	var payAmount float64
	var created time.Time
	err := q.QueryRowContext(ctx,
		`SELECT order_id, order_number, COALESCE(activity_id,0), COALESCE(activity_event_id,0),
		        COALESCE(order_type,0), COALESCE(status,0), COALESCE(title,''), COALESCE(poster,''),
		        COALESCE(pay_amount,0), gmt_create
		   FROM order_info WHERE order_id = ? AND user_id = ? LIMIT 1`, orderID, userID).
		Scan(&o.OrderID, &o.OrderNumber, &o.ShowID, &o.SessionID, &o.OrderType, &o.Status,
			&o.Title, &o.Poster, &payAmount, &created)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	o.ID = o.OrderID
	o.BuyType = 1
	o.CreateTime = biz.FmtDT(created)
	o.PayAmount = biz.Fen(payAmount)
	return o, nil
}

// isDuplicateKey 判断 MySQL 唯一键冲突（错误码 1062）。
func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// newOrderID 生成应用侧唯一订单 ID（毫秒时间戳 + 20 位随机），替代并发不安全的 MAX+1。
func newOrderID() int64 {
	return (time.Now().UnixMilli() << 20) | rand.Int63n(1<<20)
}

// newOrderNumber 生成 PN + 毫秒时间戳 + 6 位随机数的订单号（配合唯一索引保证唯一）。
func newOrderNumber(now time.Time) string {
	return fmt.Sprintf("PN%d%06d", now.UnixMilli(), rand.Intn(1000000))
}

func fenToYuan(v int64) float64 { return float64(v) / 100 }
