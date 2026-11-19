package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"

	"planstar/internal/biz"
)

// CatalogRepo 实现 biz.CatalogRepo，跨 planstar_catalog / planstar_ticket / planstar_content 查询。
type CatalogRepo struct {
	d      *Data
	syncMu sync.Mutex
}

func NewCatalogRepo(d *Data) *CatalogRepo { return &CatalogRepo{d: d} }

const activityBase = `SELECT a.activity_id, a.name, a.category_id,
	       COALESCE(NULLIF(a.category_name,''), ac.name, '') AS category_name,
	   a.city_id, COALESCE(NULLIF(a.city_name,''), c.city_name, '') AS city_name,
	   COALESCE(NULLIF(a.venue_name,''), v.name, '') AS venue_name,
	   COALESCE(v.address,'') AS venue_address,
	   a.low_price, a.high_price, a.star, a.review_num, a.want_watch_num, a.status, a.is_sold_out, a.is_proxy_buy,
	   COALESCE(a.poster, a.img, '') AS poster, COALESCE(a.duration,'') AS duration, LEFT(COALESCE(a.description,''), 500) AS descr,
	   COALESCE(a.display_tags,'[]') AS display_tags,
	   (SELECT MIN(e.start_time) FROM planstar_catalog.activity_event e WHERE e.activity_id = a.activity_id) AS first_time,
	   (SELECT GROUP_CONCAT(l.name ORDER BY l.id SEPARATOR ',') FROM planstar_catalog.activity_lineup l WHERE l.activity_id = a.activity_id) AS lineup
  FROM planstar_catalog.activity a
  LEFT JOIN planstar_catalog.activity_category ac ON ac.category_id = a.category_id
  LEFT JOIN planstar_catalog.city c ON c.city_id = a.city_id
  LEFT JOIN planstar_catalog.venue v ON v.venue_id = a.venue_id`

type activityRaw struct {
	ActivityID   int64
	Name         string
	CategoryID   int64
	CategoryName string
	CityID       int64
	CityName     string
	VenueName    string
	VenueAddress string
	LowPrice     float64
	HighPrice    float64
	Star         int
	ReviewNum    int
	WantWatchNum int
	Status       int
	IsSoldOut    int
	IsProxyBuy   int
	Poster       string
	Duration     string
	Descr        string
	DisplayTags  string
	FirstTime    sql.NullTime
	Lineup       sql.NullString
}

func scanActivity(sc interface {
	Scan(dest ...any) error
}) (*activityRaw, error) {
	raw := &activityRaw{}
	err := sc.Scan(&raw.ActivityID, &raw.Name, &raw.CategoryID, &raw.CategoryName,
		&raw.CityID, &raw.CityName, &raw.VenueName, &raw.VenueAddress,
		&raw.LowPrice, &raw.HighPrice, &raw.Star, &raw.ReviewNum, &raw.WantWatchNum,
		&raw.Status, &raw.IsSoldOut, &raw.IsProxyBuy,
		&raw.Poster, &raw.Duration, &raw.Descr, &raw.DisplayTags,
		&raw.FirstTime, &raw.Lineup)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (raw *activityRaw) summary() *biz.ShowSummary {
	lineup := splitLineup(raw.Lineup.String)
	artist := ""
	if len(lineup) > 0 {
		artist = strings.Join(lineup, " / ")
	}
	showTime := ""
	if raw.FirstTime.Valid {
		showTime = biz.FmtDT(raw.FirstTime.Time)
	}
	s := &biz.ShowSummary{
		ID: raw.ActivityID, ActivityID: raw.ActivityID,
		Title: raw.Name, Name: raw.Name,
		Artist: artist, Star: artist,
		CategoryID: raw.CategoryID, Category: raw.CategoryID,
		CategoryName: raw.CategoryName,
		City:         raw.CityName, Venue: raw.VenueName, VenueName: raw.VenueName,
		VenueAddress:   raw.VenueAddress,
		FirstSessionAt: showTime, ShowTime: showTime,
		PriceFrom: biz.Fen(raw.LowPrice), MinPrice: biz.Fen(raw.LowPrice),
		ListPriceFrom: biz.Fen(raw.HighPrice), OriginalPrice: biz.Fen(raw.HighPrice),
		DiscountPermille: 1000,
		Discount:         1000,
		Tags:             parseTagNames(raw.DisplayTags),
		WishCount:        raw.WantWatchNum, WannaSeeCount: raw.WantWatchNum,
		SoldOut: raw.IsSoldOut > 0, Status: raw.IsSoldOut == 0,
		ProxyBuy: raw.IsProxyBuy > 0,
		Rating:   float64(raw.Star), RatingCount: raw.ReviewNum,
		Duration: raw.Duration, Intro: raw.Descr, Description: raw.Descr,
		Poster:         raw.Poster,
		Sessions:       []biz.SessionSummary{},
		Tiers:          []biz.TierSummary{},
		Lineup:         lineup,
		PurchaseNotice: []string{},
	}
	return s
}

func splitLineup(v string) []string {
	out := make([]string, 0, 4)
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseTagNames(raw string) []string {
	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t.Name != "" {
			out = append(out, t.Name)
		}
	}
	return out
}

func (r *CatalogRepo) Banners(ctx context.Context) ([]*biz.Banner, error) {
	db, err := r.d.DB("content")
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT banner_id, COALESCE(title,''), COALESCE(poster,''), COALESCE(url,'' ),
		        COALESCE(jump_schema,''), COALESCE(sort,0)
		   FROM banner WHERE status = 1
		ORDER BY sort ASC, banner_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.Banner, 0, 8)
	for rows.Next() {
		b := &biz.Banner{}
		var jump string
		if err := rows.Scan(&b.ID, &b.Title, &b.Poster, &b.URL, &jump, &b.Sort); err != nil {
			return nil, err
		}
		b.Image = b.Poster
		b.ShowID = parseShowID(jump)
		items = append(items, b)
	}
	return items, rows.Err()
}

// parseShowID 从跳转 scheme 中提取企划 ID（planstar://activity/1001 或 ...activityId=1001）。
func parseShowID(schema string) int64 {
	if schema == "" {
		return 0
	}
	idx := strings.Index(schema, "activityId=")
	if idx >= 0 {
		rest := schema[idx+len("activityId="):]
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if v, err := strconv.ParseInt(rest[:end], 10, 64); err == nil {
			return v
		}
	}
	idx = strings.LastIndex(schema, "activity/")
	if idx >= 0 {
		rest := schema[idx+len("activity/"):]
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if v, err := strconv.ParseInt(rest[:end], 10, 64); err == nil {
			return v
		}
	}
	return 0
}

func (r *CatalogRepo) Activities(ctx context.Context, q biz.ActivityQuery, hot bool) ([]*biz.ShowSummary, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	where := "a.status = 1 AND a.publish_status = 1"
	args := make([]any, 0, 6)
	if q.CategoryID > 0 {
		where += " AND a.category_id = ?"
		args = append(args, q.CategoryID)
	}
	if q.City != "" {
		if id, perr := strconv.ParseInt(q.City, 10, 64); perr == nil {
			where += " AND a.city_id = ?"
			args = append(args, id)
		} else {
			where += " AND (a.city_name = ? OR c.city_name = ?)"
			args = append(args, q.City, q.City)
		}
	}

	// 热门榜优先走 Elasticsearch 读模型；ES 不可用或未同步时自动回退 MySQL 排序。
	if hot && q.CategoryID == 0 && q.City == "" {
		if items, err := r.hotFromES(ctx, q); err == nil {
			return items, nil
		} else {
			catalogESFallbackTotal.Inc()
			if !errors.Is(err, errNoElastic) {
				r.d.log.WarnContext(ctx, "elasticsearch hot query failed, fallback to mysql", "err", err)
			}
		}
	}

	order := "a.rank_weight DESC, a.activity_id DESC"
	if hot {
		order = "a.want_watch_num DESC, a.rank_weight DESC, a.activity_id DESC"
	}
	offset := (q.Page - 1) * q.PageSize
	args = append(args, q.PageSize, offset)

	rows, err := db.QueryContext(ctx, activityBase+" WHERE "+where+" ORDER BY "+order+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ShowSummary, 0, q.PageSize)
	for rows.Next() {
		raw, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, raw.summary())
	}
	return items, rows.Err()
}

// hotFromES 按 ES 排序取 ID，再回 MySQL 补全详情并保持 ES 顺序。
func (r *CatalogRepo) hotFromES(ctx context.Context, q biz.ActivityQuery) ([]*biz.ShowSummary, error) {
	offset := (q.Page - 1) * q.PageSize
	ids, err := r.HotIDs(ctx, offset, q.PageSize)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*biz.ShowSummary{}, nil
	}
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := db.QueryContext(ctx,
		activityBase+" WHERE a.activity_id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int64]*biz.ShowSummary, len(ids))
	for rows.Next() {
		raw, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		byID[raw.ActivityID] = raw.summary()
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]*biz.ShowSummary, 0, len(ids))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			items = append(items, s)
		}
	}
	return items, nil
}

func (r *CatalogRepo) Activity(ctx context.Context, id int64) (*biz.ShowSummary, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	raw, err := scanActivity(db.QueryRowContext(ctx, activityBase+" WHERE a.activity_id = ?", id))
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, biz.ErrNotFound
	}
	s := raw.summary()

	// 场次
	rows, err := db.QueryContext(ctx,
		`SELECT activity_event_id, start_time, low_price, status
		   FROM activity_event WHERE activity_id = ? ORDER BY start_time ASC`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var (
			eventID  int64
			start    sql.NullTime
			lowPrice sql.NullFloat64
			status   int
		)
		if err := rows.Scan(&eventID, &start, &lowPrice, &status); err != nil {
			rows.Close()
			return nil, err
		}
		item := biz.SessionSummary{SoldOut: status != 1}
		if start.Valid {
			item.StartsAt = biz.FmtDT(start.Time)
			item.ShowTime = item.StartsAt
		}
		if lowPrice.Valid {
			item.TierSummary = "¥" + strconv.FormatInt(biz.Fen(lowPrice.Float64)/100, 10) + " 起"
		}
		s.Sessions = append(s.Sessions, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 回报档位（按企划聚合，同名档位取最低价；读 legacy planstar_ticket.ticket_category / ticket_group）
	trows, err := db.QueryContext(ctx,
		`SELECT tc.specification, COALESCE(tc.low_price, tc.origin_price, 0), COALESCE(tc.is_e_ticket,0),
		        COALESCE((SELECT MIN(tg.sale_price) FROM planstar_ticket.ticket_group tg
		                   WHERE tg.ticket_category_id = tc.ticket_category_id AND tg.is_saling = 1), tc.low_price, 0) AS sale_price,
		        COALESCE((SELECT SUM(GREATEST(tg.count - tg.sold, 0)) FROM planstar_ticket.ticket_group tg
		                   WHERE tg.ticket_category_id = tc.ticket_category_id AND tg.is_saling = 1), 0) AS remaining
		   FROM planstar_ticket.ticket_category tc
		  WHERE tc.activity_id = ? AND tc.status = 1
		ORDER BY sale_price ASC, tc.ticket_category_id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	seen := map[string]int{}
	for trows.Next() {
		var (
			spec      string
			lowPrice  float64
			isE       int
			salePrice float64
			remaining float64
		)
		if err := trows.Scan(&spec, &lowPrice, &isE, &salePrice, &remaining); err != nil {
			return nil, err
		}
		if idx, ok := seen[spec]; ok {
			s.Tiers[idx].Remaining += int(remaining)
			continue
		}
		note := ""
		if isE > 0 {
			note = "电子票"
		}
		seen[spec] = len(s.Tiers)
		s.Tiers = append(s.Tiers, biz.TierSummary{
			Name: spec, TierName: spec, Price: biz.Fen(salePrice), Note: note, Remaining: int(remaining),
		})
	}
	if err := trows.Err(); err != nil {
		return nil, err
	}

	// 介绍与购票须知
	var intro string
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(detail_desc,'') FROM activity_detail WHERE activity_id = ? AND type = 1 ORDER BY id LIMIT 1`, id).
		Scan(&intro); err == nil && intro != "" {
		s.Intro = intro
		s.Description = intro
	}
	var notice string
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(detail_desc,'') FROM activity_detail WHERE activity_id = ? AND type IN (5,2)
		 ORDER BY FIELD(type,5,2), id LIMIT 1`, id).Scan(&notice); err == nil && notice != "" {
		s.PurchaseNotice = splitNotice(notice)
	}
	return s, nil
}

func splitNotice(v string) []string {
	v = strings.ReplaceAll(v, "\r\n", "\n")
	v = strings.ReplaceAll(v, "；", "\n")
	out := make([]string, 0, 8)
	for _, p := range strings.Split(v, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
