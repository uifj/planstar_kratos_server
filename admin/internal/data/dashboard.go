package data

import (
	"context"
	"time"

	"planstar/admin/internal/biz"
)

// trendDays is the window of every dashboard trend series, including today.
const trendDays = 14

// Dashboard aggregates read-only metrics. Every query is a whitelisted
// projection; no query joins across databases.
func (r *consoleRepo) Dashboard(ctx context.Context) (*biz.Dashboard, error) {
	d := &biz.Dashboard{}
	var err error

	if d.UsersTotal, err = r.count(ctx, `SELECT COUNT(*) FROM planstar_user.user_info`); err != nil {
		return nil, err
	}

	rows, err := r.data.readDB.QueryContext(ctx,
		`SELECT publish_status, COUNT(*) FROM planstar_catalog.project GROUP BY publish_status ORDER BY publish_status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var bucket biz.StatusCount
		if err := rows.Scan(&bucket.Status, &bucket.Count); err != nil {
			return nil, err
		}
		d.ProjectsByPublishStatus = append(d.ProjectsByPublishStatus, bucket)
		d.ProjectsTotal += bucket.Count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if d.ProjectsOngoing, err = r.count(ctx,
		`SELECT COUNT(*) FROM planstar_catalog.project WHERE funding_status = 'ongoing'`); err != nil {
		return nil, err
	}
	var raised float64
	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(raised_amount),0) FROM planstar_catalog.project WHERE publish_status = 1`).Scan(&raised); err != nil {
		return nil, err
	}
	d.RaisedTotal = yuanToCents(raised)

	var pledged float64
	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(support_amount),0) FROM planstar_order.pledge
		  WHERE pledge_status IN ('paid','confirmed','shipped','delivered')`).Scan(&d.PledgesPaid, &pledged); err != nil {
		return nil, err
	}
	d.PledgedAmount = yuanToCents(pledged)

	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(publish_status = 1),0) FROM planstar_catalog.project_idea`).Scan(&d.IdeasTotal, &d.IdeasPublished); err != nil {
		return nil, err
	}
	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(vote_type = 1),0), COALESCE(SUM(vote_type = 2),0)
		   FROM planstar_catalog.project_idea_vote`).Scan(&d.VotesTotal, &d.VotesSupport, &d.VotesOppose); err != nil {
		return nil, err
	}

	var gmv float64
	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(pay_time IS NOT NULL),0),
		        COALESCE(SUM(CASE WHEN pay_time IS NOT NULL THEN pay_amount ELSE 0 END),0)
		   FROM planstar_order.order_info`).Scan(&d.OrdersTotal, &d.OrdersPaid, &gmv); err != nil {
		return nil, err
	}
	d.GMV = yuanToCents(gmv)

	if d.RefundsPending, err = r.count(ctx,
		`SELECT COUNT(*) FROM planstar_payment.order_refund WHERE status IN ('applied','processing')`); err != nil {
		return nil, err
	}
	if d.ReportsPending, err = r.count(ctx,
		`SELECT COUNT(*) FROM planstar_catalog.project_report WHERE status = 'pending'`); err != nil {
		return nil, err
	}
	if d.GoodsOnSale, err = r.count(ctx, `SELECT COUNT(*) FROM planstar_shop.goods WHERE status = 1`); err != nil {
		return nil, err
	}
	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(price_coins),0) FROM planstar_shop.goods_order
		  WHERE status IN ('paid','completed')`).Scan(&d.GoodsOrdersPaid, &d.CoinsSpent); err != nil {
		return nil, err
	}
	if err := r.data.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(status <> 1),0) FROM planstar_community.feed`).Scan(&d.FeedsTotal, &d.FeedsHidden); err != nil {
		return nil, err
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(trendDays - 1))
	series := []struct {
		name  string
		query string
	}{
		{"orders", `SELECT DATE_FORMAT(gmt_create,'%Y-%m-%d'), COUNT(*) FROM planstar_order.order_info WHERE gmt_create >= ? GROUP BY 1`},
		{"pledges", `SELECT DATE_FORMAT(gmt_create,'%Y-%m-%d'), COUNT(*) FROM planstar_order.pledge WHERE gmt_create >= ? GROUP BY 1`},
		{"votes", `SELECT DATE_FORMAT(gmt_create,'%Y-%m-%d'), COUNT(*) FROM planstar_catalog.project_idea_vote WHERE gmt_create >= ? GROUP BY 1`},
		{"new_users", `SELECT DATE_FORMAT(gmt_create,'%Y-%m-%d'), COUNT(*) FROM planstar_user.login_session WHERE is_new_user = 1 AND is_success = 1 AND gmt_create >= ? GROUP BY 1`},
	}
	for _, s := range series {
		points, err := r.dailyCounts(ctx, s.query, start)
		if err != nil {
			return nil, err
		}
		d.Trends = append(d.Trends, biz.TrendSeries{Name: s.name, Points: points})
	}
	return d, nil
}

func (r *consoleRepo) count(ctx context.Context, query string, args ...any) (int64, error) {
	var n int64
	err := r.data.readDB.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

// dailyCounts runs one grouped query and fills missing days with zero so every
// series carries the same 14-day skeleton.
func (r *consoleRepo) dailyCounts(ctx context.Context, query string, start time.Time) ([]biz.DailyCount, error) {
	rows, err := r.data.readDB.QueryContext(ctx, query, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int64, trendDays)
	for rows.Next() {
		var date string
		var n int64
		if err := rows.Scan(&date, &n); err != nil {
			return nil, err
		}
		counts[date] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	points := make([]biz.DailyCount, 0, trendDays)
	for i := 0; i < trendDays; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		points = append(points, biz.DailyCount{Date: date, Count: counts[date]})
	}
	return points, nil
}