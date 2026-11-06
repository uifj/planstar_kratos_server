package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/olivere/elastic/v7"
)

// errNoElastic 表示未配置 Elasticsearch，调用方应回退 MySQL。
var errNoElastic = errors.New("elasticsearch not configured")

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

// EnsureIndex 幂等创建索引（已存在则跳过）。
func (r *CatalogRepo) EnsureIndex(ctx context.Context) error {
	es := r.d.Elastic()
	if es == nil {
		return nil
	}
	exists, err := es.IndexExists(r.d.ElasticIndex()).Do(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = es.CreateIndex(r.d.ElasticIndex()).BodyString(esActivityMapping).Do(ctx)
	if err != nil && !elastic.IsStatusCode(err, 400) {
		return err
	}
	return nil
}

// SyncActivities 全量重建演出索引（数据量小，用 bulk 覆盖写）。
func (r *CatalogRepo) SyncActivities(ctx context.Context) error {
	es := r.d.Elastic()
	if es == nil {
		return nil
	}
	db, err := r.d.DB("catalog")
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT a.activity_id, a.name, COALESCE(NULLIF(a.city_name,''), c.city_name, ''),
		        a.category_id, COALESCE(NULLIF(a.category_name,''), ac.name, ''),
		        COALESCE(NULLIF(a.venue_name,''), v.name, ''),
		        a.low_price, a.want_watch_num, a.rank_weight, a.star, a.status, a.publish_status,
		        COALESCE(a.poster, a.img, ''),
		        DATE_FORMAT((SELECT MIN(e.start_time) FROM planstar_catalog.activity_event e WHERE e.activity_id = a.activity_id), '%Y-%m-%d %H:%i:%s')
		   FROM planstar_catalog.activity a
		   LEFT JOIN planstar_catalog.activity_category ac ON ac.category_id = a.category_id
		   LEFT JOIN planstar_catalog.city c ON c.city_id = a.city_id
		   LEFT JOIN planstar_catalog.venue v ON v.venue_id = a.venue_id
		  WHERE a.status = 1 AND a.publish_status = 1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	bulk := es.Bulk().Index(r.d.ElasticIndex())
	count := 0
	for rows.Next() {
		var doc esActivityDoc
		var firstTime *string
		if err := rows.Scan(&doc.ActivityID, &doc.Name, &doc.CityName, &doc.CategoryID, &doc.CategoryName,
			&doc.VenueName, &doc.LowPrice, &doc.WantWatchNum, &doc.RankWeight, &doc.Star,
			&doc.Status, &doc.PublishStatus, &doc.Poster, &firstTime); err != nil {
			return err
		}
		if firstTime != nil {
			doc.FirstTime = *firstTime
		}
		bulk.Add(elastic.NewBulkIndexRequest().Id(fmt.Sprint(doc.ActivityID)).Doc(doc))
		count++
		if count%500 == 0 {
			if _, err := bulk.Do(ctx); err != nil {
				return err
			}
			bulk = es.Bulk().Index(r.d.ElasticIndex())
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
	r.d.log.InfoContext(ctx, "elasticsearch activities synced", "docs", count, "index", r.d.ElasticIndex())
	return nil
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

// RunSync 周期同步 ES 索引，随 kratos.App 生命周期启停。
func (r *CatalogRepo) RunSync(ctx context.Context, interval time.Duration, log *slog.Logger) {
	if r.d.Elastic() == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		syncCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		if err := r.SyncActivities(syncCtx); err != nil {
			log.WarnContext(syncCtx, "elasticsearch sync failed", "err", err)
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
