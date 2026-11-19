package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"planstar/internal/biz"
)

// ReportRepo 实现 biz.ReportRepo，落 planstar_catalog.project_report。
type ReportRepo struct {
	d *Data
}

func NewReportRepo(d *Data) *ReportRepo { return &ReportRepo{d: d} }

const reportColumns = `SELECT report_id, target_type, target_id, COALESCE(project_id,0), user_id,
	       reason_code, COALESCE(description,''), COALESCE(evidence,'[]'), status,
	       COALESCE(handle_note,''), handle_time, gmt_create
	  FROM planstar_catalog.project_report`

func (r *ReportRepo) List(ctx context.Context, status string, limit int) ([]*biz.Report, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	query := reportColumns
	args := make([]any, 0, 2)
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY gmt_create DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.Report, 0, limit)
	for rows.Next() {
		item, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ReportRepo) Handle(ctx context.Context, reportID int64, status, note string) (*biz.Report, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx,
		`UPDATE planstar_catalog.project_report
		    SET status = ?, handle_note = ?, handle_time = NOW()
		  WHERE report_id = ?`, status, note, reportID)
	if err != nil {
		return nil, err
	}
	// 用回查结果判断举报单是否存在：RowsAffected 统计「实际变更行数」，
	// 重复提交相同结论会返回 0，不能作为不存在的依据。
	item, err := scanReport(db.QueryRowContext(ctx, reportColumns+" WHERE report_id = ?", reportID))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, biz.ErrNotFound
	}
	return item, nil
}

func scanReport(sc interface {
	Scan(dest ...any) error
}) (*biz.Report, error) {
	item := &biz.Report{}
	var handleTime sql.NullTime
	var created time.Time
	var evidence string
	err := sc.Scan(&item.ReportID, &item.TargetType, &item.TargetID, &item.ProjectID, &item.UserID,
		&item.ReasonCode, &item.Description, &evidence, &item.Status,
		&item.HandleNote, &handleTime, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if evidence != "" {
		_ = json.Unmarshal([]byte(evidence), &item.Evidence)
	}
	if item.Evidence == nil {
		item.Evidence = []string{}
	}
	if handleTime.Valid {
		t := handleTime.Time
		item.HandleTime = &t
	}
	item.GmtCreate = created
	return item, nil
}
