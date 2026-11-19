package service

import (
	"net/http"
	"strconv"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

func (s *Service) banners(w http.ResponseWriter, r *http.Request) {
	items, err := s.catalog.Banners(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []*biz.Banner{}
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) hotActivities(w http.ResponseWriter, r *http.Request) {
	q := biz.ActivityQuery{
		Page:     httpx.IntQuery(r, "page", 1),
		PageSize: httpx.IntQuery(r, "pageSize", 20),
	}
	items, err := s.catalog.Activities(r.Context(), q, true)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []*biz.ShowSummary{}
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) activities(w http.ResponseWriter, r *http.Request) {
	q := biz.ActivityQuery{
		Page:       httpx.IntQuery(r, "page", 1),
		PageSize:   httpx.IntQuery(r, "pageSize", 20),
		CategoryID: httpx.Int64Query(r, "categoryId", 0),
		City:       r.URL.Query().Get("city"),
	}
	if q.City == "" {
		q.City = r.URL.Query().Get("cityId")
	}
	items, err := s.catalog.Activities(r.Context(), q, false)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []*biz.ShowSummary{}
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) activityDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	item, err := s.catalog.Activity(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}
