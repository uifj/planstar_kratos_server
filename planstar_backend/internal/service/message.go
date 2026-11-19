package service

import (
	"net/http"

	"planstar/internal/pkg/httpx"
)

func (s *Service) messages(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.message.Messages(r.Context(), userID,
		httpx.IntQuery(r, "type", 0),
		httpx.IntQuery(r, "pageIndex", 1),
		httpx.IntQuery(r, "pageSize", 20))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) unreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	n, err := s.message.UnreadCount(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, n)
}

func (s *Service) markRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.message.MarkRead(r.Context(), userID, httpx.Int64Any(body, "messageId", "id")); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{})
}
