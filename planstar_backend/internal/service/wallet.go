package service

import (
	"net/http"

	"planstar/internal/pkg/httpx"
)

// signInStatus 返回当日签到状态（只读，不产生签到）。
func (s *Service) signInStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	status, err := s.wallet.Status(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, status)
}

// signIn 执行每日签到；重复调用返回 alreadySigned 且不重复入账。
func (s *Service) signIn(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	status, err := s.wallet.SignIn(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, status)
}

// walletAccount 返回钱包账户；首次访问时按用户统计余额开户。
func (s *Service) walletAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	item, err := s.wallet.Account(r.Context(), userID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// walletSequences 返回钱包流水（默认最近 50 条）。
func (s *Service) walletSequences(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.wallet.Sequences(r.Context(), userID, httpx.IntQuery(r, "limit", 0))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}
