package service

import (
	"net/http"
	"strconv"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

func (s *Service) hotFeeds(w http.ResponseWriter, r *http.Request) {
	s.listFeeds(w, r, true)
}

func (s *Service) feeds(w http.ResponseWriter, r *http.Request) {
	s.listFeeds(w, r, false)
}

func (s *Service) listFeeds(w http.ResponseWriter, r *http.Request, hot bool) {
	q := biz.FeedQuery{
		Page:        httpx.IntQuery(r, "page", 1),
		PageSize:    httpx.IntQuery(r, "pageSize", 20),
		NextPageKey: httpx.Int64Query(r, "nextPageKey", 0),
	}
	page, err := s.feed.Feeds(r.Context(), q, hot)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

func (s *Service) feedDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	item, err := s.feed.Feed(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) feedComments(w http.ResponseWriter, r *http.Request) {
	tweetID := httpx.Int64Query(r, "tweetId", 0)
	items, err := s.feed.Comments(r.Context(), tweetID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}
