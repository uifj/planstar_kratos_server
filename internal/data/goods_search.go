package data

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/olivere/elastic/v7"
)

// goodsMapping pins the fields the read path filters and sorts on. keyword
// fields use exact term queries; name/goods_sn/sku_codes are searched with a
// multi_match so a free-text keyword can hit any of them.
const goodsMapping = `{
  "settings": {"number_of_shards": 1, "number_of_replicas": 0},
  "mappings": {
    "properties": {
      "id":            {"type": "keyword"},
      "name":          {"type": "text"},
      "goods_sn":      {"type": "text", "fields": {"raw": {"type": "keyword"}}},
      "market_price":  {"type": "long"},
      "on_sale":       {"type": "boolean"},
      "is_new":        {"type": "boolean"},
      "is_hot":        {"type": "boolean"},
      "category_id":   {"type": "keyword"},
      "brand_id":      {"type": "keyword"},
      "category_name": {"type": "keyword"},
      "brand_name":    {"type": "keyword"},
      "sold_num":      {"type": "long"},
      "sku_codes":     {"type": "keyword"},
      "sku_ids":       {"type": "keyword"}
    }
  }
}`

type goodsSearch struct {
	es    *elastic.Client
	index string
}

func NewGoodsSearch(d *Data) biz.GoodsSearch {
	return &goodsSearch{es: d.es, index: d.esIndex}
}

func (s *goodsSearch) EnsureIndex(ctx context.Context) error {
	exists, err := s.es.IndexExists(s.index).Do(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = s.es.CreateIndex(s.index).BodyString(goodsMapping).Do(ctx)
	return err
}

func (s *goodsSearch) Index(ctx context.Context, doc *biz.GoodsDocument) error {
	if doc == nil || doc.ID == "" {
		return nil
	}
	_, err := s.es.Index().
		Index(s.index).Id(doc.ID).BodyJson(doc).Refresh("true").Do(ctx)
	return err
}

func (s *goodsSearch) Search(ctx context.Context, q biz.GoodsQuery) ([]string, int32, error) {
	page, pageSize := q.Page, q.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	bq := elastic.NewBoolQuery()
	if kw := strings.TrimSpace(q.Keywords); kw != "" {
		bq.Must(elastic.NewMultiMatchQuery(kw, "name", "goods_sn", "sku_codes", "category_name", "brand_name"))
	} else {
		bq.Must(elastic.NewMatchAllQuery())
	}
	if q.OnlyOnSale {
		bq.Filter(elastic.NewTermQuery("on_sale", true))
	}
	if q.CategoryID != "" {
		bq.Filter(elastic.NewTermQuery("category_id", q.CategoryID))
	}
	if q.BrandID != "" {
		bq.Filter(elastic.NewTermQuery("brand_id", q.BrandID))
	}
	if q.SkuCode != "" {
		bq.Filter(elastic.NewTermQuery("sku_codes", q.SkuCode))
	}

	// Sorting belongs in SortBy, not in a ShouldQuery (the source project put
	// a sort clause into the bool query, which silently did nothing).
	res, err := s.es.Search().
		Index(s.index).
		Query(bq).
		From((page - 1) * pageSize).Size(pageSize).
		SortBy(
			elastic.NewFieldSort("sold_num").Desc().Missing(0),
			elastic.NewFieldSort("id").Desc(),
		).Do(ctx)
	if err != nil {
		return nil, 0, err
	}
	if res == nil || res.Hits == nil {
		return []string{}, 0, nil
	}
	ids := make([]string, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		ids = append(ids, hit.Id)
	}
	return ids, int32(res.TotalHits()), nil
}

// UpdateSoldNum increments sold_num on the goods document(s) that own the
// given SKU. sku_ids is indexed as a keyword field for exactly this lookup.
func (s *goodsSearch) UpdateSoldNum(ctx context.Context, skuID string, delta int64) error {
	script := elastic.NewScriptInline(
		"ctx._source.sold_num = (ctx._source.sold_num ?: 0) + params.delta",
	).Param("delta", delta)
	_, err := s.es.UpdateByQuery(s.index).
		Query(elastic.NewTermQuery("sku_ids", skuID)).
		Script(script).
		ProceedOnVersionConflict().
		Do(ctx)
	return err
}
