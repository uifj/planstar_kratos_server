package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/prometheus/client_golang/prometheus"
)

// errNoElastic 表示未配置 Elasticsearch，调用方应回退 MySQL。
var errNoElastic = errors.New("elasticsearch not configured")

// catalogESFallbackTotal 统计 ES 查询失败回退 MySQL 的次数。
// 使用默认注册表并容忍重复注册，避免多包初始化时 panic。
var catalogESFallbackTotal = func() prometheus.Counter {
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "planstar",
		Subsystem: "catalog",
		Name:      "es_fallback_total",
		Help:      "目录查询因 ES 不可用/失败回退 MySQL 的次数",
	})
	if err := prometheus.Register(c); err != nil {
		var already prometheus.AlreadyRegisteredError
		if errors.As(err, &already) {
			if existing, ok := already.ExistingCollector.(prometheus.Counter); ok {
				return existing
			}
		}
	}
	return c
}()

// RegisterCatalogMetrics 把目录降级计数注册到指定注册表（/metrics 使用独立 Registry）。
// 同一 Collector 可注册到多个 Registry，因此不影响默认注册表中的同名指标。
func RegisterCatalogMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	if err := reg.Register(catalogESFallbackTotal); err != nil {
		var already prometheus.AlreadyRegisteredError
		if !errors.As(err, &already) {
			return
		}
	}
}

// esActivityMapping 是演出库读模型的索引结构。
const esActivityMapping = `{
  "settings": {"number_of_shards": 1, "number_of_replicas": 0},
  "mappings": {
    "properties": {
      "activity_id":    {"type": "long"},
      "name":           {"type": "text", "fields": {"keyword": {"type": "keyword"}}},
      "city_name":      {"type": "keyword"},
      "category_id":    {"type": "long"},
      "category_name":  {"type": "keyword"},
      "venue_name":     {"type": "keyword"},
      "low_price":      {"type": "double"},
      "want_watch_num": {"type": "long"},
      "rank_weight":    {"type": "double"},
      "star":           {"type": "integer"},
      "status":         {"type": "integer"},
      "publish_status": {"type": "integer"},
      "poster":         {"type": "keyword", "index": false},
      "first_time":     {"type": "date", "format": "yyyy-MM-dd HH:mm:ss"}
    }
  }
}`

// esActivitySelect 是读模型文档的公共查询；调用方追加 WHERE 条件。
const esActivitySelect = `SELECT a.activity_id, a.name, COALESCE(NULLIF(a.city_name,''), c.city_name, ''),
	        a.category_id, COALESCE(NULLIF(a.category_name,''), ac.name, ''),
	        COALESCE(NULLIF(a.venue_name,''), v.name, ''),
	        a.low_price, a.want_watch_num, a.rank_weight, a.star, a.status, a.publish_status,
	        COALESCE(a.poster, a.img, ''),
	        DATE_FORMAT((SELECT MIN(e.start_time) FROM planstar_catalog.activity_event e WHERE e.activity_id = a.activity_id), '%Y-%m-%d %H:%i:%s')
	   FROM planstar_catalog.activity a
	   LEFT JOIN planstar_catalog.activity_category ac ON ac.category_id = a.category_id
	   LEFT JOIN planstar_catalog.city c ON c.city_id = a.city_id
	   LEFT JOIN planstar_catalog.venue v ON v.venue_id = a.venue_id`

type esActivityDoc struct {
	ActivityID    int64   `json:"activity_id"`
	Name          string  `json:"name"`
	CityName      string  `json:"city_name"`
	CategoryID    int64   `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	VenueName     string  `json:"venue_name"`
	LowPrice      float64 `json:"low_price"`
	WantWatchNum  int     `json:"want_watch_num"`
	RankWeight    float64 `json:"rank_weight"`
	Star          int     `json:"star"`
	Status        int     `json:"status"`
	PublishStatus int     `json:"publish_status"`
	Poster        string  `json:"poster"`
	FirstTime     string  `json:"first_time"`
}

// EnsureIndex 迁移旧版「具体索引即查询名」结构，并确保别名模型可用。
// 新索引由 SyncActivities 创建，成功后原子切换别名；此处不做全量写入。
func (r *CatalogRepo) EnsureIndex(ctx context.Context) error {
	es := r.d.Elastic()
	if es == nil {
		return nil
	}
	// 旧版本直接用 planstar-activities 作为具体索引，与别名同名；读模型可重建，
	// 这里删除遗留索引，随后由一次全量同步重建并挂载别名（期间查询回退 MySQL）。
	res, err := es.Aliases().Do(ctx)
	if err != nil {
		return err
	}
	alias := r.d.ElasticIndex()
	if _, legacy := res.Indices[alias]; legacy {
		if _, err := es.DeleteIndex(alias).Do(ctx); err != nil {
			return err
		}
		r.d.log.WarnContext(ctx, "legacy elasticsearch index removed for alias migration", "index", alias)
	}
	return nil
}

// SyncActivities 全量重建：写入新索引 → 刷新 → 原子切换别名 → 删除旧索引。
// 任一步失败都保留旧索引并清理新索引，查询不中断。
func (r *CatalogRepo) SyncActivities(ctx context.Context) error {
	es := r.d.Elastic()
	if es == nil {
		return nil
	}
	r.syncMu.Lock()
	defer r.syncMu.Unlock()

	alias := r.d.ElasticIndex()
	newIndex := fmt.Sprintf("%s-%s", alias, time.Now().UTC().Format("20060102150405"))
	if _, err := es.CreateIndex(newIndex).BodyString(esActivityMapping).Do(ctx); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = es.DeleteIndex(newIndex).Do(context.WithoutCancel(ctx))
		}
	}()

	db, err := r.d.DB("catalog")
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, esActivitySelect+" WHERE a.status = 1 AND a.publish_status = 1")
	if err != nil {
		return err
	}
	defer rows.Close()

	bulk := es.Bulk().Index(newIndex)
	count := 0
	for rows.Next() {
		doc, err := scanESActivity(rows)
		if err != nil {
			return err
		}
		bulk.Add(elastic.NewBulkIndexRequest().Id(fmt.Sprint(doc.ActivityID)).Doc(doc))
		count++
		if count%500 == 0 {
			if _, err := bulk.Do(ctx); err != nil {
				return err
			}
			bulk = es.Bulk().Index(newIndex)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count%500 != 0 || count == 0 {
		if _, err := bulk.Do(ctx); err != nil {
			return err
		}
	}

	// 刷新后再切别名，保证切换后立即可查。
	if _, err := es.Refresh(newIndex).Do(ctx); err != nil {
		return err
	}

	oldIndices := r.aliasedIndices(ctx, alias)
	svc := es.Alias().Add(newIndex, alias)
	for _, old := range oldIndices {
		if old != newIndex {
			svc = svc.Remove(old, alias)
		}
	}
	if _, err := svc.Do(ctx); err != nil {
		return err
	}
	committed = true

	// 别名切换成功后清理旧索引；失败不影响读模型可用性。
	for _, old := range oldIndices {
		if old == newIndex {
			continue
		}
		if _, err := es.DeleteIndex(old).Do(ctx); err != nil {
			r.d.log.WarnContext(ctx, "delete old elasticsearch index failed", "index", old, "err", err)
		}
	}
	r.d.log.InfoContext(ctx, "elasticsearch activities synced", "docs", count, "index", newIndex, "alias", alias)
	return nil
}

// aliasedIndices 返回当前别名指向的具体索引；别名不存在时返回空。
func (r *CatalogRepo) aliasedIndices(ctx context.Context, alias string) []string {
	es := r.d.Elastic()
	if es == nil {
		return nil
	}
	res, err := es.Aliases().Index(alias).Do(ctx)
	if err != nil {
		if !elastic.IsNotFound(err) {
			r.d.log.WarnContext(ctx, "get elasticsearch alias failed", "alias", alias, "err", err)
		}
		return nil
	}
	return res.IndicesByAlias(alias)
}

// HotIDs 用 ES 计算热门榜单的活动 ID（按想看数、权重降序）。
// 返回 error 时调用方应回退 MySQL 排序，保证 ES 抖动不影响可用性。
func (r *CatalogRepo) HotIDs(ctx context.Context, offset, limit int) ([]int64, error) {
	es := r.d.Elastic()
	if es == nil {
		return nil, errNoElastic
	}
	ctx, cancel := context.WithTimeout(ctx, r.d.ElasticTimeout())
	defer cancel()

	res, err := es.Search().
		Index(r.d.ElasticIndex()).
		Query(elastic.NewTermQuery("status", 1)).
		Sort("want_watch_num", false).
		Sort("rank_weight", false).
		Sort("activity_id", false).
		From(offset).Size(limit).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var doc esActivityDoc
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		if doc.ActivityID > 0 {
			ids = append(ids, doc.ActivityID)
		}
	}
	return ids, nil
}

// UpsertActivity 增量写入/更新单个演出文档，供目录变更事件消费调用（本次不接入 MQ）。
// 演出下架或不存在时转为删除文档。
func (r *CatalogRepo) UpsertActivity(ctx context.Context, activityID int64) error {
	es := r.d.Elastic()
	if es == nil || activityID <= 0 {
		return nil
	}
	db, err := r.d.DB("catalog")
	if err != nil {
		return err
	}
	doc, err := scanESActivity(db.QueryRowContext(ctx,
		esActivitySelect+" WHERE a.activity_id = ? AND a.status = 1 AND a.publish_status = 1", activityID))
	if err == sql.ErrNoRows {
		return r.DeleteActivity(ctx, activityID)
	}
	if err != nil {
		return err
	}
	_, err = es.Index().
		Index(r.d.ElasticIndex()).
		Id(fmt.Sprint(doc.ActivityID)).
		BodyJson(doc).
		Do(ctx)
	return err
}

// DeleteActivity 增量删除单个演出文档，供目录变更事件消费调用（本次不接入 MQ）。
func (r *CatalogRepo) DeleteActivity(ctx context.Context, activityID int64) error {
	es := r.d.Elastic()
	if es == nil || activityID <= 0 {
		return nil
	}
	_, err := es.Delete().Index(r.d.ElasticIndex()).Id(fmt.Sprint(activityID)).Do(ctx)
	if err != nil && elastic.IsNotFound(err) {
		return nil
	}
	return err
}

// RunSync 启动时先做一次全量同步，之后按 interval 周期兜底。
// interval <= 0 表示关闭周期同步（仍保留启动时一次同步）。
// 另可用 PLANSTAR_ES_SYNC_INTERVAL_SECONDS=0 显式关闭（conf 会把 0 归一化为默认 300s）。
func (r *CatalogRepo) RunSync(ctx context.Context, interval time.Duration, log *slog.Logger) {
	if r.d.Elastic() == nil {
		return
	}
	if v := os.Getenv("PLANSTAR_ES_SYNC_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			interval = time.Duration(n) * time.Second
		}
	}
	syncOnce := func() {
		syncCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		if err := r.SyncActivities(syncCtx); err != nil {
			log.WarnContext(syncCtx, "elasticsearch sync failed", "err", err)
		}
		cancel()
	}
	syncOnce()
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncOnce()
		}
	}
}

// scanESActivity 把一行查询结果转成读模型文档。
func scanESActivity(sc interface {
	Scan(dest ...any) error
}) (*esActivityDoc, error) {
	doc := &esActivityDoc{}
	var firstTime *string
	err := sc.Scan(&doc.ActivityID, &doc.Name, &doc.CityName, &doc.CategoryID, &doc.CategoryName,
		&doc.VenueName, &doc.LowPrice, &doc.WantWatchNum, &doc.RankWeight, &doc.Star,
		&doc.Status, &doc.PublishStatus, &doc.Poster, &firstTime)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if firstTime != nil {
		doc.FirstTime = *firstTime
	}
	return doc, nil
}
