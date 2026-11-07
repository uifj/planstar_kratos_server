package service

import (
	"net/http"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

// chatThreads 返回当前用户的会话列表。
func (s *Service) chatThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.chat.Threads(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// chatMessages 返回会话消息分页（limit/before，兼容 pageSize）。
func (s *Service) chatMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	threadID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	limit := httpx.IntQuery(r, "limit", 0)
	if limit <= 0 {
		limit = httpx.IntQuery(r, "pageSize", 20)
	}
	items, err := s.chat.Messages(r.Context(), userID, threadID, limit,
		httpx.Int64Query(r, "before", 0))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// sendChatMessage 发送消息，返回落库后的消息。
func (s *Service) sendChatMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	threadID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	item, err := s.chat.SendMessage(r.Context(), userID, threadID,
		httpx.StrAny(body, "text", "content", "message"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// markChatRead 标记会话已读，返回未读数 0。
func (s *Service) markChatRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	threadID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	unread, err := s.chat.MarkRead(r.Context(), userID, threadID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"unreadCount": unread})
}
