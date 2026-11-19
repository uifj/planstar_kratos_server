package biz

import (
	"context"
	"strings"

	"planstar/internal/biz/shared"
)

// 兼容层（ADR-0001 P1）：黑名单类型与 Error/Unwrap 实现已下沉到 biz/shared。
// 类型别名保证 server/admin.go 的 errors.As(&biz.BlacklistBlockedError{}) 仍然成立。
type BlacklistHit = shared.BlacklistHit
type BlacklistBlockedError = shared.BlacklistBlockedError

// ProjectModeration 是运营审核视图：企划摘要 + 黑名单判定结果。
type ProjectModeration struct {
	ProjectID     int64          `json:"projectId"`
	Title         string         `json:"title"`
	Subtitle      string         `json:"subtitle,omitempty"`
	CategoryID    int64          `json:"categoryId"`
	CategoryName  string         `json:"categoryName"`
	PublishStatus int            `json:"publishStatus"`
	FundingStatus string         `json:"fundingStatus"`
	Hits          []BlacklistHit `json:"hits"`
	Blocked       bool           `json:"blocked"`
}

// blacklistRules 对应 POSITIONING 9.3「不适合入驻」的七类硬性拦截。
var blacklistRules = []struct {
	Rule     string
	Keywords []string
}{
	{"金融集资", []string{"股权", "分红", "利息", "理财", "虚拟货币", "加密货币", "比特币", "usdt", "集资", "投资回报", "保本", "收益承诺"}},
	{"慈善募捐", []string{"慈善", "募捐", "捐助", "医疗救助", "个人求助", "筹款治病"}},
	{"黄牛票务", []string{"代购", "黄牛", "转售门票", "门票转售", "抢票", "代抢", "现货"}},
	{"违法违规", []string{"盗版", "侵权", "涉黄", "赌博", "传销", "刷单", "刷量", "虚假宣传"}},
	{"特殊资质", []string{"药品", "医疗器械", "保健品", "活体动物", "食品经营", "处方药"}},
	{"预付卡", []string{"预付卡", "储值", "预付式消费", "资金池"}},
}

// CheckProjectBlacklist 对企划文本与结构化字段做黑名单判定。
// content 为标题/副标题/简介/故事/资金用途的拼接文本；结构化缺失归入「空气企划」。
func CheckProjectBlacklist(content string, goalAmount int64, deliveryPlan, riskNote string) []BlacklistHit {
	text := strings.ToLower(content)
	hits := make([]BlacklistHit, 0, 4)
	for _, rule := range blacklistRules {
		for _, keyword := range rule.Keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				hits = append(hits, BlacklistHit{Rule: rule.Rule, Keyword: keyword})
			}
		}
	}
	if goalAmount <= 0 {
		hits = append(hits, BlacklistHit{Rule: "空气企划", Keyword: "缺少目标金额"})
	}
	if strings.TrimSpace(deliveryPlan) == "" {
		hits = append(hits, BlacklistHit{Rule: "空气企划", Keyword: "缺少交付计划"})
	}
	if strings.TrimSpace(riskNote) == "" {
		hits = append(hits, BlacklistHit{Rule: "空气企划", Keyword: "缺少风险披露"})
	}
	return hits
}

// Moderation 返回企划审核视图（不做状态过滤，草稿/驳回单同样可查）。
func (uc *ProjectUsecase) Moderation(ctx context.Context, projectID int64) (*ProjectModeration, error) {
	if projectID <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.Moderation(ctx, projectID)
}

// ReviewProject 审核企划：approve 前必须通过黑名单硬校验，reject 始终允许。
func (uc *ProjectUsecase) ReviewProject(ctx context.Context, projectID int64, decision string) (*ProjectModeration, error) {
	if projectID <= 0 {
		return nil, ErrBadRequest
	}
	decision = strings.TrimSpace(decision)
	if decision != "approve" && decision != "reject" {
		return nil, ErrBadRequest
	}
	mod, err := uc.repo.Moderation(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if decision == "approve" {
		if len(mod.Hits) > 0 {
			return nil, &BlacklistBlockedError{Hits: mod.Hits}
		}
		if err := uc.repo.SetPublishStatus(ctx, projectID, 1); err != nil {
			return nil, err
		}
		mod.PublishStatus = 1
		return mod, nil
	}
	if err := uc.repo.SetPublishStatus(ctx, projectID, 3); err != nil {
		return nil, err
	}
	mod.PublishStatus = 3
	return mod, nil
}
