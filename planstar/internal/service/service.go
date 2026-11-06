package service

import (
	"errors"
	"log/slog"
	"net/http"

	"planstar/internal/biz"
	"planstar/internal/observability"
	"planstar/internal/pkg/httpx"
	"planstar/internal/pkg/session"
)

// Service 聚合各域用例并注册全部 HTTP 端点（Transport 适配层，DTO ↔ DO）。
type Service struct {
	auth    *biz.AuthUsecase
	catalog *biz.CatalogUsecase
	feed    *biz.FeedUsecase
	message *biz.MessageUsecase
	order   *biz.OrderUsecase
	user    *biz.UserUsecase
	log     *slog.Logger
}

func New(
	auth *biz.AuthUsecase,
	catalog *biz.CatalogUsecase,
	feed *biz.FeedUsecase,
	message *biz.MessageUsecase,
	order *biz.OrderUsecase,
	user *biz.UserUsecase,
	log *slog.Logger,
) *Service {
	return &Service{
		auth: auth, catalog: catalog, feed: feed, message: message, order: order, user: user,
		log: log,
	}
}

// Register 注册全部端点。路径与客户端契约逐字一致（含 v1/v3/v4 版本错配）。
func (s *Service) Register(mux *http.ServeMux) {
	s.registerAuth(mux)
	s.registerCatalog(mux)
	s.registerFeed(mux)
	s.registerMessage(mux)
	s.registerOrder(mux)
	s.registerUser(mux)
}

func (s *Service) registerAuth(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/user/captcha", s.captchaImage)
	mux.HandleFunc("POST /v3/user/sendLoginCode", s.sendLoginCode)
	mux.HandleFunc("POST /v1/user/login", s.login)
}

func (s *Service) registerCatalog(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/home/banners", s.banners)
	mux.HandleFunc("GET /v1/activities/hot", s.hotActivities)
	mux.HandleFunc("GET /v1/activities", s.activities)
	mux.HandleFunc("GET /v3/activities/{id}", s.activityDetail)
}

func (s *Service) registerFeed(mux *http.ServeMux) {
	mux.HandleFunc("GET /v4/feed/hot", s.hotFeeds)
	mux.HandleFunc("GET /v1/feed", s.feeds)
	mux.HandleFunc("GET /v1/tweets/{id}", s.feedDetail)
	mux.HandleFunc("GET /v1/userReply", s.feedComments)
}

func (s *Service) registerMessage(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/messages", s.messages)
	mux.HandleFunc("GET /v1/messages/unreadCount", s.unreadCount)
	mux.HandleFunc("POST /v1/messages/read", s.markRead)
}

func (s *Service) registerOrder(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/order/list", s.orders)
	mux.HandleFunc("GET /v1/order/detail", s.orderDetail)
	mux.HandleFunc("POST /v1/unifiedOrder/confirm", s.confirmOrder)
	mux.HandleFunc("POST /v3/unifiedOrder/makeOrder", s.makeOrder)
}

func (s *Service) registerUser(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/user", s.profile)
	mux.HandleFunc("GET /v1/user/", s.profile)
	mux.HandleFunc("GET /v1/myaddress", s.addresses)
	mux.HandleFunc("POST /v1/myaddress", s.addAddress)
	mux.HandleFunc("DELETE /v1/myaddress/{id}", s.deleteAddress)
	mux.HandleFunc("POST /v1/myaddress/default/{id}", s.setDefaultAddress)
}

// currentUser 由 Cookie psSession 解析登录用户（Redis 缓存优先，回源 MySQL）。
func (s *Service) currentUser(r *http.Request) (int64, error) {
	return s.auth.ResolveSession(r.Context(), session.FromRequest(r))
}

func (s *Service) requireUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, err := s.currentUser(r)
	if err != nil {
		if errors.Is(err, biz.ErrUnauthorized) {
			httpx.Plain(w, http.StatusUnauthorized, "未登录")
			return 0, false
		}
		s.fail(w, r, err)
		return 0, false
	}
	return userID, true
}

func (s *Service) fail(w http.ResponseWriter, r *http.Request, err error) {
	observability.RecordError(r.Context(), err)
	switch {
	case errors.Is(err, biz.ErrUnauthorized):
		httpx.Plain(w, http.StatusUnauthorized, "未登录")
	case errors.Is(err, biz.ErrBadRequest):
		httpx.Plain(w, http.StatusBadRequest, "参数错误")
	case errors.Is(err, biz.ErrNotFound):
		httpx.Plain(w, http.StatusNotFound, "资源不存在")
	default:
		s.log.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
		httpx.Plain(w, http.StatusInternalServerError, "服务异常")
	}
}
