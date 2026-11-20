package biz

import (
	"context"
	"time"
)

// Release 是应用版本发布记录（planstar_meta.app_release）。
type Release struct {
	Platform       string    `json:"platform"`
	Channel        string    `json:"channel"`
	VersionCode    int64     `json:"versionCode"`
	VersionName    string    `json:"versionName"`
	DownloadURL    string    `json:"downloadUrl"`
	SHA256         string    `json:"sha256"`
	SizeBytes      int64     `json:"sizeBytes"`
	ForceUpgrade   int64     `json:"-"` // 显式强制升级（1/0）
	MinVersionCode int64     `json:"-"` // 低于此版本号强制升级
	Changelog      string    `json:"changelog"`
	ReleasedAt     time.Time `json:"-"` // 发布时间（appcast pubDate 用）
}

// Force 是否强制升级：显式 force_upgrade，或当前版本低于最低强制版本。
func (r *Release) Force(currentVersionCode int64) bool {
	return r.ForceUpgrade == 1 || (r.MinVersionCode > 0 && currentVersionCode < r.MinVersionCode)
}

// UpdateRepo 是应用版本发布仓储端口。
type UpdateRepo interface {
	LatestRelease(ctx context.Context, platform, channel string) (*Release, error)
}

// UpdateUsecase 应用更新用例。
type UpdateUsecase struct {
	repo UpdateRepo
}

func NewUpdateUsecase(repo UpdateRepo) *UpdateUsecase {
	return &UpdateUsecase{repo: repo}
}

// LatestRelease 返回指定平台/渠道的最新生效版本；无记录时返回 (nil, nil)。
func (uc *UpdateUsecase) LatestRelease(ctx context.Context, platform, channel string) (*Release, error) {
	if platform == "" {
		return nil, ErrBadRequest
	}
	if channel == "" {
		channel = "stable"
	}
	return uc.repo.LatestRelease(ctx, platform, channel)
}
