package biz

import (
	"context"
	"errors"
	"math"
	"strings"
)

// 企划创意域输入上限（与 Spec §4.5/§4.6 对齐）。
const (
	ideaDefaultPageSize  = 10
	ideaMaxPageSize      = 20
	ideaMaxSupportReason = 60
	ideaMaxOpposeReason  = 100
)

// 企划创意域哨兵错误，service 层据此映射 HTTP 状态。
var (
	ErrIdeaNotFound       = errors.New("idea not found")
	ErrIdeaSelfVote       = errors.New("idea self vote forbidden")
	ErrIdeaInvalidTier    = errors.New("idea invalid coin tier")
	ErrIdeaReasonRequired = errors.New("idea reason required")
	ErrIdeaInvalidReason  = errors.New("idea invalid reason code")
)

// ideaReasonCodes 是不支持理由的 P0 字典（与前端 l10n 对齐），不落表。
var ideaReasonCodes = map[string]bool{
	"not_interested":       true,
	"too_many_similar":     true,
	"not_enough_info":      true,
	"not_executable":       true,
	"unreasonable_pricing": true,
	"other":                true,
}

// ideaCoinTiers 是支持意愿档位白名单（仅表态，不扣费）。
var ideaCoinTiers = map[int]bool{5: true, 10: true, 20: true}

// IdeaQuery 是叠卡流查询参数：cursor 为空表示首页。
type IdeaQuery struct {
	Cursor   string
	Size     int
	ViewerID int64
}

// ProjectIdea 是创意卡片与详情的读模型。
// Description 仅详情返回；TierDistribution/MyVote 仅详情或有会话时填充。
type ProjectIdea struct {
	IdeaID           int64          `json:"ideaId"`
	UserID           int64          `json:"userId"`
	UserName         string         `json:"userName"`
	UserAvatar       string         `json:"userAvatar,omitempty"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	Cover            string         `json:"cover,omitempty"`
	CategoryID       int64          `json:"categoryId"`
	CategoryName     string         `json:"categoryName"`
	Stage            int            `json:"stage"`
	Description      string         `json:"description,omitempty"`
	SupportCount     int            `json:"supportCount"`
	OpposeCount      int            `json:"opposeCount"`
	CoinIntentSum    int            `json:"coinIntentSum"`
	SupportRate      float64        `json:"supportRate"`
	HotScore         float64        `json:"hotScore"`
	TierDistribution map[string]int `json:"tierDistribution,omitempty"`
	// ReasonCounts 是详情页的不支持理由标准 code 计数聚合（不含原文）。
	ReasonCounts map[string]int `json:"reasonCounts,omitempty"`
	MyVote       *IdeaVote      `json:"myVote,omitempty"`
	GmtCreate    int64          `json:"gmtCreate"`
}

// IdeaVote 是当前用户的表态（不输出理由原文）。
type IdeaVote struct {
	VoteType   int    `json:"voteType"`
	CoinTier   int    `json:"coinTier"`
	ReasonCode string `json:"reasonCode,omitempty"`
	// ReasonText 是自定义理由原文，只入库聚合，不对外输出。
	ReasonText string `json:"-"`
}

// IdeaVoteResult 是投票提交后的最新聚合结果。
type IdeaVoteResult struct {
	IdeaID           int64          `json:"ideaId"`
	SupportCount     int            `json:"supportCount"`
	OpposeCount      int            `json:"opposeCount"`
	SupportRate      float64        `json:"supportRate"`
	CoinIntentSum    int            `json:"coinIntentSum"`
	TierDistribution map[string]int `json:"tierDistribution"`
	MyVote           *IdeaVote      `json:"myVote"`
	AlreadyVoted     bool           `json:"alreadyVoted"`
}

// IdeaFeedPage 是叠卡流分页结果。
type IdeaFeedPage struct {
	Items      []*ProjectIdea `json:"items"`
	NextCursor string         `json:"nextCursor"`
	HasMore    bool           `json:"hasMore"`
}

// ProjectIdeaRepo 是企划创意域仓储端口。
type ProjectIdeaRepo interface {
	Feed(ctx context.Context, q IdeaQuery) (*IdeaFeedPage, error)
	Detail(ctx context.Context, ideaID, viewerID int64) (*ProjectIdea, error)
	Vote(ctx context.Context, ideaID, userID int64, vote *IdeaVote) (*IdeaVoteResult, bool, error)
}

// ProjectIdeaUsecase 企划创意域用例：负责入参归一化与领域错误映射。
type ProjectIdeaUsecase struct {
	repo ProjectIdeaRepo
}

func NewProjectIdeaUsecase(repo ProjectIdeaRepo) *ProjectIdeaUsecase {
	return &ProjectIdeaUsecase{repo: repo}
}

// Feed 返回叠卡流：size 默认 10、上限 20；非法游标由仓储层回退到首页。
func (uc *ProjectIdeaUsecase) Feed(ctx context.Context, cursor string, size int, viewerID int64) (*IdeaFeedPage, error) {
	if size <= 0 {
		size = ideaDefaultPageSize
	}
	if size > ideaMaxPageSize {
		size = ideaMaxPageSize
	}
	page, err := uc.repo.Feed(ctx, IdeaQuery{
		Cursor:   strings.TrimSpace(cursor),
		Size:     size,
		ViewerID: viewerID,
	})
	if err != nil {
		return nil, err
	}
	if page == nil {
		page = &IdeaFeedPage{}
	}
	if page.Items == nil {
		page.Items = []*ProjectIdea{}
	}
	return page, nil
}

// Detail 返回创意详情；viewerID>0 时附带 myVote。
func (uc *ProjectIdeaUsecase) Detail(ctx context.Context, ideaID, viewerID int64) (*ProjectIdea, error) {
	if ideaID <= 0 {
		return nil, ErrIdeaNotFound
	}
	return uc.repo.Detail(ctx, ideaID, viewerID)
}

// Vote 提交表态：校验类型、档位、理由与文本长度，作者自投由仓储层拦截。
func (uc *ProjectIdeaUsecase) Vote(ctx context.Context, ideaID, userID int64, vote *IdeaVote) (*IdeaVoteResult, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if ideaID <= 0 {
		return nil, ErrIdeaNotFound
	}
	if vote == nil || (vote.VoteType != 1 && vote.VoteType != 2) {
		return nil, ErrBadRequest
	}

	vote.ReasonCode = strings.TrimSpace(vote.ReasonCode)
	voteText := strings.TrimSpace(vote.ReasonText)
	vote.ReasonText = voteText
	if vote.VoteType == 1 {
		if !ideaCoinTiers[vote.CoinTier] {
			return nil, ErrIdeaInvalidTier
		}
		if len([]rune(voteText)) > ideaMaxSupportReason {
			return nil, ErrBadRequest
		}
		// 支持不记录标准理由；自定义文本作为作者改进输入。
		vote.ReasonCode = ""
	} else {
		vote.CoinTier = 0
		if vote.ReasonCode == "" {
			return nil, ErrIdeaReasonRequired
		}
		if !ideaReasonCodes[vote.ReasonCode] {
			return nil, ErrIdeaInvalidReason
		}
		if len([]rune(voteText)) > ideaMaxOpposeReason {
			return nil, ErrBadRequest
		}
		if vote.ReasonCode == "other" && voteText == "" {
			return nil, ErrIdeaReasonRequired
		}
	}

	result, alreadyVoted, err := uc.repo.Vote(ctx, ideaID, userID, vote)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &IdeaVoteResult{}
	}
	result.AlreadyVoted = alreadyVoted
	if result.MyVote == nil {
		result.MyVote = &IdeaVote{VoteType: vote.VoteType, CoinTier: vote.CoinTier, ReasonCode: vote.ReasonCode}
	}
	if result.TierDistribution == nil {
		result.TierDistribution = map[string]int{"5": 0, "10": 0, "20": 0}
	}
	result.SupportRate = IdeaSupportRate(result.SupportCount, result.OpposeCount)
	return result, nil
}

// IdeaSupportRate 计算支持率（支持 /（支持+不支持）），保留三位小数。
func IdeaSupportRate(support, oppose int) float64 {
	total := support + oppose
	if total <= 0 {
		return 0
	}
	return math.Round(float64(support)/float64(total)*1000) / 1000
}
