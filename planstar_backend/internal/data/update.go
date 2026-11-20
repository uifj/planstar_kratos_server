package data

import (
	"context"
	"database/sql"

	"planstar/internal/biz"
)

// UpdateRepo 实现 biz.UpdateRepo，落 planstar_meta。
type UpdateRepo struct {
	d *Data
}

func NewUpdateRepo(d *Data) *UpdateRepo { return &UpdateRepo{d: d} }

// LatestRelease 查询指定平台/渠道最新的生效版本；无记录返回 (nil, nil)。
func (r *UpdateRepo) LatestRelease(ctx context.Context, platform, channel string) (*biz.Release, error) {
	db, err := r.d.DB("meta")
	if err != nil {
		return nil, err
	}
	rel := &biz.Release{Platform: platform, Channel: channel}
	err = db.QueryRowContext(ctx,
		`SELECT version_code, version_name, download_url, sha256, size_bytes,
		        force_upgrade, min_version_code, COALESCE(changelog,''), released_at
		   FROM app_release
		  WHERE platform = ? AND channel = ? AND is_active = 1
		  ORDER BY version_code DESC LIMIT 1`, platform, channel).
		Scan(&rel.VersionCode, &rel.VersionName, &rel.DownloadURL, &rel.SHA256, &rel.SizeBytes,
			&rel.ForceUpgrade, &rel.MinVersionCode, &rel.Changelog, &rel.ReleasedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rel, nil
}
