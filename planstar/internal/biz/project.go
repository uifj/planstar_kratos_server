package biz

import (
	"context"
	"math"
	"strings"
	"time"
)

// 企划域分页与输入上限（与 API_CONTRACT 7.7 对齐）。
const (
	maxProjectPageSize   = 100
	maxPledgeQuantity    = 99
	maxSupportMessage    = 255
	maxReviewContent     = 1024
	maxReportDescription = 1024
	rankingDefaultLimit  = 10
	rankingMaxLimit      = 50
)

// projectSorts 是 /v1/projects 允许的排序枚举。
var projectSorts = map[string]bool{
	"": true, "hot": true, "new": true, "ending": true, "achievement": true,
}

// projectFundingStatuses 是 /v1/projects 允许的筹资状态枚举。
var projectFundingStatuses = map[string]bool{
	"preheating": true, "ongoing": true, "achieved": true, "delivered": true, "failed": true,
}

// reportTargetTypes 与 reportReasonCodes 是举报接口的枚举白名单。
var reportTargetTypes = map[string]bool{
	"project": true, "review": true, "feed": true, "pledge": true,
}

var reportReasonCodes = map[string]bool{
	"fraud": true, "infringe": true, "porn": true, "violence": true, "false_claim": true, "other": true,
}

// reviewRiskWords 命中后评价进入审核态（status=2），不对外展示。
var reviewRiskWords = []string{
	"包过", "代考", "博彩", "赌博", "返现", "加微信", "刷单", "传销", "集资", "分红",
}

// ProjectQuery 是企划列表查询参数。
type ProjectQuery struct {
	Page          int
	PageSize      int
	CategoryID    int64
	FundingStatus string
	Sort          string
	// Hot 为 true 时按精选/权重/支持人数排序（/v1/projects/hot）。
	Hot bool
}

// ProjectSummary 是企划卡片（契约 8.10 ProjectSummary）。
type ProjectSummary struct {
	ProjectID        int64    `json:"projectId"`
	ProjectNo        string   `json:"projectNo"`
	Title            string   `json:"title"`
	Subtitle         string   `json:"subtitle,omitempty"`
	Cover            string   `json:"cover"`
	CategoryID       int64    `json:"categoryId"`
	CategoryName     string   `json:"categoryName"`
	ProjectType      string   `json:"projectType"`
	CreatorID        int64    `json:"creatorId"`
	CreatorName      string   `json:"creatorName"`
	GoalAmount       int64    `json:"goalAmount"`
	RaisedAmount     int64    `json:"raisedAmount"`
	SupporterCount   int      `json:"supporterCount"`
	Deadline         int64    `json:"deadline"`
	RemainingDays    int      `json:"remainingDays"`
	FundingStatus    string   `json:"fundingStatus"`
	AchievementRate  int      `json:"achievementRate"`
	MinSupportAmount int64    `json:"minSupportAmount"`
	Tags             []string `json:"tags"`
	IsFeatured       bool     `json:"isFeatured"`
	// TagIDs 是内部字段：仓储层据此把标签 ID 翻译为名称，不参与 JSON 输出。
	TagIDs []int64 `json:"-"`
}

// CreatorProfile 是创作者主页与信用（契约 8.10 CreatorProfile）。
type CreatorProfile struct {
	CreatorID       int64  `json:"creatorId"`
	DisplayName     string `json:"displayName"`
	Avatar          string `json:"avatar"`
	IsVerified      bool   `json:"isVerified"`
	ProjectNum      int    `json:"projectNum"`
	SuccessNum      int    `json:"successNum"`
	FulfillmentRate int    `json:"fulfillmentRate"`
	Bio             string `json:"bio,omitempty"`
}

// CreatorPage 是创作者主页响应：创作者资料 + 已发布企划。
type CreatorPage struct {
	CreatorProfile
	Projects []*ProjectSummary `json:"projects"`
}

// ProjectFaq 是企划常见问题（契约 8.10 ProjectFaq）。
type ProjectFaq struct {
	FaqID    int64  `json:"faqId"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Sort     int    `json:"sort"`
}

// RewardTier 是回报档位（契约 8.10 RewardTier）。
// RemainingNum 对不限量档位为 nil（JSON 省略），限量档位始终输出（含 0）。
type RewardTier struct {
	RewardTierID          int64    `json:"rewardTierId"`
	Title                 string   `json:"title"`
	Price                 int64    `json:"price"`
	RewardType            string   `json:"rewardType"`
	Items                 []string `json:"items"`
	LimitNum              int      `json:"limitNum"`
	SoldNum               int      `json:"soldNum"`
	RemainingNum          *int     `json:"remainingNum,omitempty"`
	EstimatedDeliveryTime string   `json:"estimatedDeliveryTime,omitempty"`
	IsEarlyBird           bool     `json:"isEarlyBird"`
	IsLimited             bool     `json:"isLimited"`
}

// ProjectUpdate 是企划进度更新（契约 8.10 ProjectUpdate）。
type ProjectUpdate struct {
	UpdateID        int64    `json:"updateId"`
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	Images          []string `json:"images"`
	IsSupporterOnly bool     `json:"isSupporterOnly"`
	PublishTime     int64    `json:"publishTime"`
}

// StretchGoal 是企划解锁目标（详情页 stretchGoals 项）。
type StretchGoal struct {
	GoalID        int64  `json:"goalId"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	TargetAmount  int64  `json:"targetAmount"`
	CurrentAmount int64  `json:"currentAmount"`
	IsAchieved    bool   `json:"isAchieved"`
	Sort          int    `json:"sort"`
}

// ProjectDetail 是企划详情：ProjectSummary + 故事/风险/档位/更新等。
type ProjectDetail struct {
	ProjectSummary
	Story        string           `json:"story"`
	Gallery      []string         `json:"gallery"`
	RiskNote     string           `json:"riskNote"`
	DeliveryPlan string           `json:"deliveryPlan"`
	FundUsage    string           `json:"fundUsage"`
	Faq          []*ProjectFaq    `json:"faq"`
	RefundRule   string           `json:"refundRule"`
	RefundDays   int              `json:"refundDays"`
	RatingAvg    float64          `json:"ratingAvg"`
	ReviewCount  int              `json:"reviewCount"`
	ShowCount    int              `json:"showCount"`
	Creator      *CreatorProfile  `json:"creator"`
	RewardTiers  []*RewardTier    `json:"rewardTiers"`
	Updates      []*ProjectUpdate `json:"updates"`
	StretchGoals []*StretchGoal   `json:"stretchGoals"`
}

// Pledge 是支持记录（契约 8.10 Pledge）。
type Pledge struct {
	PledgeID       int64  `json:"pledgeId"`
	PledgeNo       string `json:"pledgeNo"`
	ProjectID      int64  `json:"projectId"`
	RewardTierID   int64  `json:"rewardTierId"`
	SupportAmount  int64  `json:"supportAmount"`
	Quantity       int    `json:"quantity"`
	IsAnonymous    bool   `json:"isAnonymous"`
	SupportMessage string `json:"supportMessage,omitempty"`
	DeliveryType   string `json:"deliveryType"`
	PledgeStatus   string `json:"pledgeStatus"`
	PaidTime       int64  `json:"paidTime"`
	DeliveredTime  int64  `json:"deliveredTime"`
	TrackingNo     string `json:"trackingNo,omitempty"`
	ContentURL     string `json:"contentUrl,omitempty"`
	RedeemCode     string `json:"redeemCode,omitempty"`
	RedeemedTime   int64  `json:"redeemedTime"`
}

// ProjectReview 是支持者评价与晒单（契约 8.10 ProjectReview）。
type ProjectReview struct {
	ReviewID    int64    `json:"reviewId"`
	ProjectID   int64    `json:"projectId"`
	UserName    string   `json:"userName"`
	UserAvatar  string   `json:"userAvatar,omitempty"`
	Rating      int      `json:"rating"`
	Content     string   `json:"content,omitempty"`
	Images      []string `json:"images"`
	IsShow      bool     `json:"isShow"`
	IsAnonymous bool     `json:"isAnonymous"`
	LikeCount   int      `json:"likeCount"`
	GmtCreate   int64    `json:"gmtCreate"`
}

// ProjectReport 是举报单（契约 8.10 ProjectReport）。
type ProjectReport struct {
	ReportID    int64    `json:"reportId"`
	TargetType  string   `json:"targetType"`
	TargetID    int64    `json:"targetId"`
	ReasonCode  string   `json:"reasonCode"`
	Description string   `json:"description,omitempty"`
	Evidence    []string `json:"evidence"`
	Status      string   `json:"status"`
	HandleNote  string   `json:"handleNote,omitempty"`
	GmtCreate   int64    `json:"gmtCreate"`
}

// ReviewPage 是评价列表响应：条目 + 聚合统计。
type ReviewPage struct {
	Items       []*ProjectReview `json:"items"`
	RatingAvg   float64          `json:"ratingAvg"`
	ReviewCount int              `json:"reviewCount"`
	ShowCount   int              `json:"showCount"`
}

// RankingItem 是榜单条目。
type RankingItem struct {
	Rank        int             `json:"rank"`
	Project     *ProjectSummary `json:"project"`
	MetricValue int64           `json:"metricValue"`
}

// ProjectRanking 是企划榜单（契约 8.10 ProjectRanking）。
type ProjectRanking struct {
	Type        string         `json:"type"`
	GeneratedAt int64          `json:"generatedAt"`
	Items       []*RankingItem `json:"items"`
}

// PledgeInput 是创建支持单入参。
type PledgeInput struct {
	RewardTierID int64
	Quantity     int
	IsAnonymous  bool
	Message      string
	AddressID    int64
}

// ReviewInput 是发布评价入参。
type ReviewInput struct {
	PledgeID    int64
	Rating      int
	Content     string
	Images      []string
	IsShow      bool
	IsAnonymous bool
}

// ReportInput 是举报入参。
type ReportInput struct {
	TargetType  string
	TargetID    int64
	ReasonCode  string
	Description string
	Evidence    []string
}

// ProjectRepo 是企划域仓储端口。
type ProjectRepo interface {
	List(ctx context.Context, q ProjectQuery) ([]*ProjectSummary, error)
	Detail(ctx context.Context, projectID int64) (*ProjectDetail, error)
	RewardTiers(ctx context.Context, projectID int64) ([]*RewardTier, error)
	Updates(ctx context.Context, projectID int64, includeSupporterOnly bool, page, pageSize int) ([]*ProjectUpdate, error)
	HasSupporterOnly(ctx context.Context, projectID int64) (bool, error)
	IsSupporter(ctx context.Context, projectID, userID int64) (bool, error)
	Faqs(ctx context.Context, projectID int64) ([]*ProjectFaq, error)
	Creator(ctx context.Context, creatorID int64) (*CreatorPage, error)
	Reviews(ctx context.Context, projectID int64, sort string, onlyShow bool, page, pageSize int) (*ReviewPage, error)
	CreateReview(ctx context.Context, userID, projectID int64, in *ReviewInput) (*ProjectReview, error)
	CreatePledge(ctx context.Context, userID, projectID int64, in *PledgeInput) (*Pledge, error)
	Pledges(ctx context.Context, userID int64, status string, page, pageSize int) ([]*Pledge, error)
	Pledge(ctx context.Context, userID, pledgeID int64) (*Pledge, error)
	CreateReport(ctx context.Context, userID, projectID int64, in *ReportInput) (*ProjectReport, error)
	Ranking(ctx context.Context, typ string, categoryID int64, limit int) ([]*RankingItem, error)
	Moderation(ctx context.Context, projectID int64) (*ProjectModeration, error)
	SetPublishStatus(ctx context.Context, projectID int64, publishStatus int) error
}

// ProjectUsecase 企划域用例：负责入参规范化与领域错误映射。
type ProjectUsecase struct {
	repo ProjectRepo
}

func NewProjectUsecase(repo ProjectRepo) *ProjectUsecase {
	return &ProjectUsecase{repo: repo}
}

// Hot 返回精选企划列表。
func (uc *ProjectUsecase) Hot(ctx context.Context, page, pageSize int) ([]*ProjectSummary, error) {
	q := ProjectQuery{Page: page, PageSize: pageSize, Hot: true}
	return uc.List(ctx, q)
}

// List 返回企划列表；排序/状态枚举非法时返回 400。
func (uc *ProjectUsecase) List(ctx context.Context, q ProjectQuery) ([]*ProjectSummary, error) {
	if !projectSorts[strings.TrimSpace(q.Sort)] {
		return nil, ErrBadRequest
	}
	q.Sort = strings.TrimSpace(q.Sort)
	if q.FundingStatus = strings.TrimSpace(q.FundingStatus); q.FundingStatus != "" && !projectFundingStatuses[q.FundingStatus] {
		return nil, ErrBadRequest
	}
	normalizeProjectPage(&q.Page, &q.PageSize)
	items, err := uc.repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*ProjectSummary{}
	}
	return items, nil
}

// Detail 返回企划详情。
func (uc *ProjectUsecase) Detail(ctx context.Context, projectID int64) (*ProjectDetail, error) {
	if projectID <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.Detail(ctx, projectID)
}

// RewardTiers 返回回报档位；非法 ID 与不存在的企划统一按 404 处理。
func (uc *ProjectUsecase) RewardTiers(ctx context.Context, projectID int64) ([]*RewardTier, error) {
	if projectID <= 0 {
		return nil, ErrNotFound
	}
	return uc.repo.RewardTiers(ctx, projectID)
}

// Updates 返回进度更新；仅支持者可见的条目对未登录/未支持用户不可见。
func (uc *ProjectUsecase) Updates(ctx context.Context, projectID, userID int64, page, pageSize int) ([]*ProjectUpdate, error) {
	if projectID <= 0 {
		return nil, ErrNotFound
	}
	normalizeProjectPage(&page, &pageSize)

	if userID <= 0 {
		ok, err := uc.repo.HasSupporterOnly(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if ok {
			return nil, ErrUnauthorized
		}
		items, err := uc.repo.Updates(ctx, projectID, false, page, pageSize)
		return nonNilUpdates(items, err)
	}

	supporter, err := uc.repo.IsSupporter(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !supporter {
		ok, err := uc.repo.HasSupporterOnly(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if ok {
			return nil, ErrForbidden
		}
	}
	items, err := uc.repo.Updates(ctx, projectID, supporter, page, pageSize)
	return nonNilUpdates(items, err)
}

func nonNilUpdates(items []*ProjectUpdate, err error) ([]*ProjectUpdate, error) {
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*ProjectUpdate{}
	}
	return items, nil
}

// CreatePledge 创建支持单：校验档位、数量、留言与地址，库存/结束冲突返回 409。
func (uc *ProjectUsecase) CreatePledge(ctx context.Context, userID, projectID int64, in *PledgeInput) (*Pledge, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if projectID <= 0 || in == nil || in.RewardTierID <= 0 {
		return nil, ErrBadRequest
	}
	if in.Quantity == 0 {
		in.Quantity = 1
	}
	if in.Quantity < 0 || in.Quantity > maxPledgeQuantity {
		return nil, ErrBadRequest
	}
	in.Message = strings.TrimSpace(in.Message)
	if len([]rune(in.Message)) > maxSupportMessage {
		return nil, ErrBadRequest
	}
	return uc.repo.CreatePledge(ctx, userID, projectID, in)
}

// Pledges 返回当前用户的支持记录。
func (uc *ProjectUsecase) Pledges(ctx context.Context, userID int64, status string, page, pageSize int) ([]*Pledge, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	normalizeProjectPage(&page, &pageSize)
	items, err := uc.repo.Pledges(ctx, userID, strings.TrimSpace(status), page, pageSize)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*Pledge{}
	}
	return items, nil
}

// Pledge 返回单条支持记录（仅本人）。
func (uc *ProjectUsecase) Pledge(ctx context.Context, userID, pledgeID int64) (*Pledge, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if pledgeID <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.Pledge(ctx, userID, pledgeID)
}

// Creator 返回创作者主页。
func (uc *ProjectUsecase) Creator(ctx context.Context, creatorID int64) (*CreatorPage, error) {
	if creatorID <= 0 {
		return nil, ErrNotFound
	}
	return uc.repo.Creator(ctx, creatorID)
}

// Faqs 返回企划常见问题。
func (uc *ProjectUsecase) Faqs(ctx context.Context, projectID int64) ([]*ProjectFaq, error) {
	if projectID <= 0 {
		return nil, ErrNotFound
	}
	items, err := uc.repo.Faqs(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*ProjectFaq{}
	}
	return items, nil
}

// Reviews 返回评价列表与聚合统计；未知 sort 回落到 latest。
func (uc *ProjectUsecase) Reviews(ctx context.Context, projectID int64, sort string, onlyShow bool, page, pageSize int) (*ReviewPage, error) {
	if projectID <= 0 {
		return nil, ErrNotFound
	}
	sort = strings.TrimSpace(sort)
	if sort != "hot" {
		sort = "latest"
	}
	normalizeProjectPage(&page, &pageSize)
	pageData, err := uc.repo.Reviews(ctx, projectID, sort, onlyShow, page, pageSize)
	if err != nil {
		return nil, err
	}
	if pageData.Items == nil {
		pageData.Items = []*ProjectReview{}
	}
	return pageData, nil
}

// CreateReview 发布评价：仅本人且已交付的支持记录，重复评价返回 409。
func (uc *ProjectUsecase) CreateReview(ctx context.Context, userID, projectID int64, in *ReviewInput) (*ProjectReview, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if projectID <= 0 || in == nil || in.PledgeID <= 0 {
		return nil, ErrBadRequest
	}
	if in.Rating < 1 || in.Rating > 5 {
		return nil, ErrBadRequest
	}
	in.Content = strings.TrimSpace(in.Content)
	if len([]rune(in.Content)) > maxReviewContent {
		return nil, ErrBadRequest
	}
	if in.Images == nil {
		in.Images = []string{}
	}
	return uc.repo.CreateReview(ctx, userID, projectID, in)
}

// CreateReport 提交举报：同用户同对象 24 小时内重复举报返回 409。
func (uc *ProjectUsecase) CreateReport(ctx context.Context, userID, projectID int64, in *ReportInput) (*ProjectReport, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if projectID <= 0 || in == nil || in.TargetID <= 0 {
		return nil, ErrBadRequest
	}
	in.TargetType = strings.TrimSpace(in.TargetType)
	in.ReasonCode = strings.TrimSpace(in.ReasonCode)
	if !reportTargetTypes[in.TargetType] || !reportReasonCodes[in.ReasonCode] {
		return nil, ErrBadRequest
	}
	in.Description = strings.TrimSpace(in.Description)
	if len([]rune(in.Description)) > maxReportDescription {
		return nil, ErrBadRequest
	}
	if in.Evidence == nil {
		in.Evidence = []string{}
	}
	return uc.repo.CreateReport(ctx, userID, projectID, in)
}

// Rankings 返回榜单；type 非法返回 400，limit 上限 50。
func (uc *ProjectUsecase) Rankings(ctx context.Context, typ string, categoryID int64, limit int) (*ProjectRanking, error) {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		typ = "hot"
	}
	if typ != "hot" && typ != "achievement" && typ != "ending" {
		return nil, ErrBadRequest
	}
	if limit <= 0 {
		limit = rankingDefaultLimit
	}
	if limit > rankingMaxLimit {
		limit = rankingMaxLimit
	}
	items, err := uc.repo.Ranking(ctx, typ, categoryID, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*RankingItem{}
	}
	return &ProjectRanking{Type: typ, GeneratedAt: time.Now().UnixMilli(), Items: items}, nil
}

// IsReviewRisky 判断评价内容是否命中风控词。
func IsReviewRisky(content string) bool {
	for _, word := range reviewRiskWords {
		if strings.Contains(content, word) {
			return true
		}
	}
	return false
}

// AchievementPermille 把已筹/目标换算为千分比。
func AchievementPermille(raised, goal float64) int {
	if goal <= 0 {
		return 0
	}
	return int(math.Round(raised / goal * 1000))
}

// RemainingDays 返回距截止日期的整天数（已结束为 0）。
func RemainingDays(deadline time.Time) int {
	if deadline.IsZero() {
		return 0
	}
	days := int(time.Until(deadline).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func normalizeProjectPage(page, pageSize *int) {
	if *page <= 0 {
		*page = 1
	}
	if *pageSize <= 0 || *pageSize > maxProjectPageSize {
		*pageSize = 20
	}
}
