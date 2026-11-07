package service

import (
	"net/http"
	"strconv"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

// pathID 解析路径参数为 int64；非法或非正数返回 false。
func pathID(r *http.Request, key string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (s *Service) hotProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.project.Hot(r.Context(),
		httpx.IntQuery(r, "page", 1),
		httpx.IntQuery(r, "pageSize", 20))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) projectList(w http.ResponseWriter, r *http.Request) {
	q := biz.ProjectQuery{
		Page:          httpx.IntQuery(r, "page", 1),
		PageSize:      httpx.IntQuery(r, "pageSize", 20),
		CategoryID:    httpx.Int64Query(r, "categoryId", 0),
		FundingStatus: r.URL.Query().Get("fundingStatus"),
		Sort:          r.URL.Query().Get("sort"),
	}
	items, err := s.project.List(r.Context(), q)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) projectDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	item, err := s.project.Detail(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) projectRewardTiers(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrNotFound)
		return
	}
	items, err := s.project.RewardTiers(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) projectUpdates(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrNotFound)
		return
	}
	userID, err := s.currentUser(r)
	if err != nil {
		userID = 0
	}
	items, err := s.project.Updates(r.Context(), id, userID,
		httpx.IntQuery(r, "page", 1),
		httpx.IntQuery(r, "pageSize", 20))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) createPledge(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.PledgeInput{
		RewardTierID: httpx.Int64Any(body, "rewardTierId", "reward_tier_id", "tierId"),
		Quantity:     int(httpx.Int64Any(body, "quantity", "count")),
		IsAnonymous:  httpx.BoolAny(body, "isAnonymous", "anonymous"),
		Message:      httpx.StrAny(body, "message", "supportMessage"),
		AddressID:    httpx.Int64Any(body, "addressId", "address_id"),
	}
	item, err := s.project.CreatePledge(r.Context(), userID, projectID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) pledges(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.project.Pledges(r.Context(), userID, r.URL.Query().Get("status"),
		httpx.IntQuery(r, "page", 1),
		httpx.IntQuery(r, "pageSize", 20))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) pledgeDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrNotFound)
		return
	}
	item, err := s.project.Pledge(r.Context(), userID, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) creatorPage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrNotFound)
		return
	}
	item, err := s.project.Creator(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) projectFaqs(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrNotFound)
		return
	}
	items, err := s.project.Faqs(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) projectReviews(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrNotFound)
		return
	}
	page, err := s.project.Reviews(r.Context(), id,
		r.URL.Query().Get("sort"),
		boolQuery(r, "onlyShow"),
		httpx.IntQuery(r, "page", 1),
		httpx.IntQuery(r, "pageSize", 20))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

func (s *Service) createReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.ReviewInput{
		PledgeID:    httpx.Int64Any(body, "pledgeId", "pledge_id"),
		Rating:      int(httpx.Int64Any(body, "rating")),
		Content:     httpx.StrAny(body, "content"),
		Images:      stringList(body, "images"),
		IsShow:      httpx.BoolAny(body, "isShow", "show"),
		IsAnonymous: httpx.BoolAny(body, "isAnonymous", "anonymous"),
	}
	item, err := s.project.CreateReview(r.Context(), userID, projectID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) createReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.ReportInput{
		TargetType:  httpx.StrAny(body, "targetType", "target_type"),
		TargetID:    httpx.Int64Any(body, "targetId", "target_id"),
		ReasonCode:  httpx.StrAny(body, "reasonCode", "reason_code"),
		Description: httpx.StrAny(body, "description"),
		Evidence:    stringList(body, "evidence"),
	}
	item, err := s.project.CreateReport(r.Context(), userID, projectID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) rankings(w http.ResponseWriter, r *http.Request) {
	item, err := s.project.Rankings(r.Context(),
		r.URL.Query().Get("type"),
		httpx.Int64Query(r, "categoryId", 0),
		httpx.IntQuery(r, "limit", 10))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// boolQuery 兼容 true/1 两种布尔 query 写法。
func boolQuery(r *http.Request, key string) bool {
	v := r.URL.Query().Get(key)
	return v == "1" || v == "true"
}

// stringList 从通用请求体取字符串数组（非字符串项忽略）。
func stringList(m map[string]any, key string) []string {
	switch v := m[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
