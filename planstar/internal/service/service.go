package service

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"

	"planstar/internal/biz"
	"planstar/internal/conf"
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
	project *biz.ProjectUsecase
	// projectIdea 是「企划创意」叠卡流用例（契约 7.9）。
	projectIdea *biz.ProjectIdeaUsecase
	payment     *biz.PaymentUsecase
	wallet      *biz.WalletUsecase
	shop        *biz.ShopUsecase
	chat        *biz.ChatUsecase
	log         *slog.Logger
	// authCfg 保存认证风控配置（由 conf.Loaded 注入，零值时使用安全默认值）。
	authCfg conf.Auth
	// limit 是发码/登录失败的进程内限流器。
	limit *authLimiter
	// ideaVoteLimit 是企划创意投票的进程内频控（单用户 60 次/小时、单 IP 300 次/小时）。
	ideaVoteLimit *ideaVoteLimiter
}

func New(
	auth *biz.AuthUsecase,
	catalog *biz.CatalogUsecase,
	feed *biz.FeedUsecase,
	message *biz.MessageUsecase,
	order *biz.OrderUsecase,
	user *biz.UserUsecase,
	project *biz.ProjectUsecase,
	projectIdea *biz.ProjectIdeaUsecase,
	payment *biz.PaymentUsecase,
	wallet *biz.WalletUsecase,
	shop *biz.ShopUsecase,
	chat *biz.ChatUsecase,
	log *slog.Logger,
) *Service {
	cfg := conf.Loaded()
	return &Service{
		auth: auth, catalog: catalog, feed: feed, message: message, order: order, user: user,
		project: project, projectIdea: projectIdea,
		payment: payment, wallet: wallet, shop: shop, chat: chat, log: log,
		authCfg: cfg.Auth, limit: newAuthLimiter(), ideaVoteLimit: newIdeaVoteLimiter(),
	}
}

// route 是注册表的一项：pattern 与 handler 一一对应。
type route struct {
	pattern string
	handler http.HandlerFunc
}

// routes 返回全部端点注册表，Register 与 Routes 共用同一份数据。
func (s *Service) routes() []route {
	return []route{
		{"GET /v1/user/captcha", s.captchaImage},
		{"POST /v3/user/sendLoginCode", s.sendLoginCode},
		{"POST /v1/user/login", s.login},

		{"GET /v1/home/banners", s.banners},
		{"GET /v1/activities/hot", s.hotActivities},
		{"GET /v1/activities", s.activities},
		{"GET /v3/activities/{id}", s.activityDetail},

		{"GET /v4/feed/hot", s.hotFeeds},
		{"GET /v1/feed", s.feeds},
		{"GET /v1/tweets/{id}", s.feedDetail},
		{"GET /v1/userReply", s.feedComments},

		{"GET /v1/messages", s.messages},
		{"GET /v1/messages/unreadCount", s.unreadCount},
		{"POST /v1/messages/read", s.markRead},

		{"GET /v1/chat/threads", s.chatThreads},
		{"GET /v1/chat/threads/{id}/messages", s.chatMessages},
		{"POST /v1/chat/threads/{id}/messages", s.sendChatMessage},
		{"POST /v1/chat/threads/{id}/read", s.markChatRead},

		{"GET /v1/order/list", s.orders},
		{"GET /v1/order/detail", s.orderDetail},
		{"POST /v1/unifiedOrder/confirm", s.confirmOrder},
		{"POST /v3/unifiedOrder/makeOrder", s.makeOrder},
		{"POST /v1/unifiedPay/pay", s.pay},
		{"GET /v1/unifiedPay/payDetail", s.payDetail},
		{"POST /v1/order/{id}/refund", s.createRefund},
		{"GET /v1/order/{id}/refund", s.orderRefund},

		{"GET /v1/user", s.profile},
		// 客户端历史上请求带尾斜杠，保留为 GET /v1/user 的别名。
		{"GET /v1/user/", s.profile},
		{"GET /v1/myaddress", s.addresses},
		{"POST /v1/myaddress", s.addAddress},
		{"DELETE /v1/myaddress/{id}", s.deleteAddress},
		{"POST /v1/myaddress/default/{id}", s.setDefaultAddress},

		// 星币钱包与每日签到（契约 7.8）。
		{"GET /v1/user/signIn", s.signInStatus},
		{"POST /v1/user/signIn", s.signIn},
		{"GET /v1/moneyaccount", s.walletAccount},
		{"GET /v1/moneyaccount/sequences", s.walletSequences},

		// 星币商城（契约 7.9）。
		{"GET /v1/shop/goods", s.shopGoods},
		{"GET /v1/shop/goods/{id}", s.shopGoodsDetail},
		{"POST /v1/shop/goods/{id}/skus/{skuId}/redeem", s.redeemGoods},

		// 企划域（契约 7.7）。
		{"GET /v1/projects/hot", s.hotProjects},
		{"GET /v1/projects", s.projectList},
		{"GET /v1/projects/{id}", s.projectDetail},
		{"GET /v1/projects/{id}/reward-tiers", s.projectRewardTiers},
		{"GET /v1/projects/{id}/updates", s.projectUpdates},
		{"POST /v1/projects/{id}/pledges", s.createPledge},
		{"GET /v1/pledges", s.pledges},
		{"GET /v1/pledges/{id}", s.pledgeDetail},
		{"GET /v1/creators/{id}", s.creatorPage},
		{"GET /v1/projects/{id}/faqs", s.projectFaqs},
		{"GET /v1/projects/{id}/reviews", s.projectReviews},
		{"POST /v1/projects/{id}/reviews", s.createReview},
		{"POST /v1/projects/{id}/reports", s.createReport},
		{"GET /v1/rankings", s.rankings},

		// 企划创意叠卡流（契约 7.9）。
		{"GET /v1/projectIdeas/feed", s.projectIdeaFeed},
		{"GET /v1/projectIdeas/{ideaId}", s.projectIdeaDetail},
		{"POST /v1/projectIdeas/{ideaId}/vote", s.projectIdeaVote},
	}
}

// Register 注册全部端点。路径与客户端契约逐字一致（含 v1/v3/v4 版本错配）。
func (s *Service) Register(mux *http.ServeMux) {
	for _, rt := range s.routes() {
		mux.HandleFunc(rt.pattern, rt.handler)
	}
}

// Routes 返回已注册路由清单（"METHOD /path"），供契约一致性测试复用。
func (s *Service) Routes() []string {
	rs := s.routes()
	out := make([]string, 0, len(rs))
	for _, rt := range rs {
		out = append(out, rt.pattern)
	}
	sort.Strings(out)
	return out
}

// currentUser 由 Cookie psSession 解析登录用户（Redis 缓存优先，回源 MySQL）。
func (s *Service) currentUser(r *http.Request) (int64, error) {
	return s.auth.ResolveSession(r.Context(), session.FromRequest(r))
}

func (s *Service) requireUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, err := s.currentUser(r)
	if err != nil {
		s.fail(w, r, err)
		return 0, false
	}
	return userID, true
}

// fail 把领域错误映射为统一 JSON 错误体；内部错误只落日志，不回显细节。
func (s *Service) fail(w http.ResponseWriter, r *http.Request, err error) {
	observability.RecordError(r.Context(), err)
	switch {
	case errors.Is(err, biz.ErrUnauthorized):
		httpx.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "未登录")
	case errors.Is(err, biz.ErrForbidden):
		httpx.Error(w, r, http.StatusForbidden, "FORBIDDEN", "无权限")
	case errors.Is(err, biz.ErrInsufficientCoins):
		httpx.Error(w, r, http.StatusBadRequest, "INSUFFICIENT_COINS", "星币余额不足")
	case errors.Is(err, biz.ErrBadRequest):
		httpx.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "参数错误")
	case errors.Is(err, biz.ErrNotFound):
		httpx.Error(w, r, http.StatusNotFound, "NOT_FOUND", "资源不存在")
	case errors.Is(err, biz.ErrPriceChanged):
		httpx.Error(w, r, http.StatusConflict, "PRICE_CHANGED", "价格已变动，请重新确认")
	case errors.Is(err, biz.ErrSoldOut):
		httpx.Error(w, r, http.StatusConflict, "SOLD_OUT", "库存不足")
	case errors.Is(err, biz.ErrLimitExceeded):
		httpx.Error(w, r, http.StatusConflict, "LIMIT_EXCEEDED", "超过限购数量")
	case errors.Is(err, biz.ErrQuoteExpired):
		httpx.Error(w, r, http.StatusConflict, "QUOTE_EXPIRED", "报价已过期，请重新确认")
	case errors.Is(err, biz.ErrRefundNotAllowed):
		httpx.Error(w, r, http.StatusConflict, "REFUND_NOT_ALLOWED", "当前订单状态不支持退款")
	case errors.Is(err, biz.ErrConflict):
		httpx.Error(w, r, http.StatusConflict, "CONFLICT", "资源冲突")
	case errors.Is(err, biz.ErrRateLimited):
		httpx.Error(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "请求过于频繁")
	case errors.Is(err, biz.ErrIdeaNotFound):
		httpx.Error(w, r, http.StatusNotFound, "IDEA_NOT_FOUND", "创意不存在")
	case errors.Is(err, biz.ErrIdeaSelfVote):
		httpx.Error(w, r, http.StatusForbidden, "IDEA_SELF_VOTE_FORBIDDEN", "不能给自己的企划投票")
	case errors.Is(err, biz.ErrIdeaInvalidTier):
		httpx.Error(w, r, http.StatusBadRequest, "INVALID_COIN_TIER", "意愿档位不合法")
	case errors.Is(err, biz.ErrIdeaReasonRequired):
		httpx.Error(w, r, http.StatusBadRequest, "REASON_REQUIRED", "请选择不支持理由")
	case errors.Is(err, biz.ErrIdeaInvalidReason):
		httpx.Error(w, r, http.StatusBadRequest, "INVALID_REASON_CODE", "不支持理由不合法")
	case errors.Is(err, httpx.ErrBodyTooLarge):
		httpx.Error(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "请求体过大")
	default:
		s.log.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
		httpx.Error(w, r, http.StatusInternalServerError, "INTERNAL", "服务异常")
	}
}
