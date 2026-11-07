package biz

import (
	"context"
	"strings"
	"time"
)

// Report 是举报处理单（运营后台视角）。
type Report struct {
	ReportID    int64      `json:"reportId"`
	TargetType  string     `json:"targetType"`
	TargetID    int64      `json:"targetId"`
	ProjectID   int64      `json:"projectId,omitempty"`
	UserID      int64      `json:"userId"`
	ReasonCode  string     `json:"reasonCode"`
	Description string     `json:"description,omitempty"`
	Evidence    []string   `json:"evidence"`
	Status      string     `json:"status"`
	HandleNote  string     `json:"handleNote,omitempty"`
	HandleTime  *time.Time `json:"handleTime,omitempty"`
	GmtCreate   time.Time  `json:"gmtCreate"`
}

// ReportRepo 是举报处理单仓储端口（仅运营后台使用）。
type ReportRepo interface {
	List(ctx context.Context, status string, limit int) ([]*Report, error)
	Handle(ctx context.Context, reportID int64, status, note string) (*Report, error)
}

// ReportUsecase 是举报审核用例。
type ReportUsecase struct {
	repo ReportRepo
}

func NewReportUsecase(repo ReportRepo) *ReportUsecase {
	return &ReportUsecase{repo: repo}
}

// List 返回待处理/已处理举报；status 为空表示全部。
func (uc *ReportUsecase) List(ctx context.Context, status string, limit int) ([]*Report, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "", "pending", "accepted", "rejected":
	default:
		return nil, ErrBadRequest
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return uc.repo.List(ctx, status, limit)
}

// Handle 处理举报：仅允许 accepted / rejected，结论为空时补默认文案。
func (uc *ReportUsecase) Handle(ctx context.Context, reportID int64, status, note string) (*Report, error) {
	if reportID <= 0 {
		return nil, ErrBadRequest
	}
	status = strings.TrimSpace(status)
	if status != "accepted" && status != "rejected" {
		return nil, ErrBadRequest
	}
	note = strings.TrimSpace(note)
	if note == "" {
		if status == "accepted" {
			note = "举报成立，已按社区规范处理。"
		} else {
			note = "经复核未发现违规，举报不成立。"
		}
	}
	return uc.repo.Handle(ctx, reportID, status, note)
}
