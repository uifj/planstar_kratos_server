package biz

import "context"

// StatusCount is one bucket of a grouped count.
type StatusCount struct {
	Status int32
	Count  int64
}

// DailyCount is one day of a trend series (date formatted YYYY-MM-DD).
type DailyCount struct {
	Date  string
	Count int64
}

// TrendSeries is a named daily series covering the last 14 days.
type TrendSeries struct {
	Name   string
	Points []DailyCount
}

// Dashboard aggregates read-only business metrics across the Planstar
// databases for the console overview page. Amounts are in cents.
type Dashboard struct {
	UsersTotal              int64
	ProjectsTotal           int64
	ProjectsByPublishStatus []StatusCount
	ProjectsOngoing         int64
	RaisedTotal             int64
	PledgesPaid             int64
	PledgedAmount           int64
	IdeasTotal              int64
	IdeasPublished          int64
	VotesTotal              int64
	VotesSupport            int64
	VotesOppose             int64
	OrdersTotal             int64
	OrdersPaid              int64
	GMV                     int64
	RefundsPending          int64
	ReportsPending          int64
	GoodsOnSale             int64
	GoodsOrdersPaid         int64
	CoinsSpent              int64
	FeedsTotal              int64
	FeedsHidden             int64
	Trends                  []TrendSeries
}

// Dashboard reads the aggregated overview metrics.
func (uc *ConsoleUsecase) Dashboard(ctx context.Context) (*Dashboard, error) {
	return uc.repo.Dashboard(ctx)
}