package service

import (
	"net/http"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

// projectIdeaFeed 返回叠卡流；游客可访问，有会话时带 viewerID 回填 myVote。
func (s *Service) projectIdeaFeed(w http.ResponseWriter, r *http.Request) {
	viewerID, err := s.currentUser(r)
	if err != nil {
		viewerID = 0
	}
	page, err := s.projectIdea.Feed(r.Context(),
		r.URL.Query().Get("cursor"),
		httpx.IntQuery(r, "size", 10),
		viewerID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

// projectIdeaDetail 返回创意详情；游客可访问，有会话时返回 myVote。
func (s *Service) projectIdeaDetail(w http.ResponseWriter, r *http.Request) {
	ideaID, ok := pathID(r, "ideaId")
	if !ok {
		s.fail(w, r, biz.ErrIdeaNotFound)
		return
	}
	viewerID, err := s.currentUser(r)
	if err != nil {
		viewerID = 0
	}
	idea, err := s.projectIdea.Detail(r.Context(), ideaID, viewerID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, idea)
}

// projectIdeaVote 提交支持/不支持表态，必须登录。
func (s *Service) projectIdeaVote(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.ideaVoteLimit.allow(userID, clientIP(r)) {
		s.fail(w, r, biz.ErrRateLimited)
		return
	}
	ideaID, ok := pathID(r, "ideaId")
	if !ok {
		s.fail(w, r, biz.ErrIdeaNotFound)
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	vote := &biz.IdeaVote{
		VoteType:   int(httpx.Int64Any(body, "voteType", "vote_type")),
		CoinTier:   int(httpx.Int64Any(body, "coinTier", "coin_tier")),
		ReasonCode: httpx.StrAny(body, "reasonCode", "reason_code"),
		ReasonText: httpx.StrAny(body, "reasonText", "reason_text"),
	}
	result, err := s.projectIdea.Vote(r.Context(), ideaID, userID, vote)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
