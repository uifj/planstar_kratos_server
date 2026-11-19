package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"planstar/admin/internal/biz"
)

// consoleRepo implements biz.ConsoleRepo. Reads are whitelisted projections of
// the business databases; the only write is user_info.status. No query joins
// across databases.
type consoleRepo struct {
	data *Data
}

// NewConsoleRepo creates the console repository.
func NewConsoleRepo(data *Data) biz.ConsoleRepo {
	return &consoleRepo{data: data}
}

const userSelect = `SELECT user_id, COALESCE(user_name,''), COALESCE(mobile_no,''), COALESCE(avatar,''),
	       status, gmt_create, gmt_modified
	  FROM planstar_user.user_info`

func (r *consoleRepo) ListUsers(ctx context.Context, q biz.UserQuery) ([]*biz.PlanstarUser, error) {
	query := userSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(user_name LIKE ? OR mobile_no LIKE ? OR CAST(user_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY user_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*biz.PlanstarUser, 0, q.Limit)
	for rows.Next() {
		user := &biz.PlanstarUser{}
		if err := rows.Scan(&user.ID, &user.Name, &user.Mobile, &user.Avatar, &user.Status,
			&user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *consoleRepo) UpdateUserStatus(ctx context.Context, id int64, status int32) (*biz.PlanstarUser, error) {
	res, err := r.data.readDB.ExecContext(ctx,
		`UPDATE planstar_user.user_info SET status = ? WHERE user_id = ?`,
		status, id)
	if err != nil {
		return nil, err
	}
	// RowsAffected is 0 when the value is unchanged, so it cannot distinguish
	// a no-op update from a missing user; check existence in that case only.
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		var exists int
		err := r.data.readDB.QueryRowContext(ctx,
			`SELECT 1 FROM planstar_user.user_info WHERE user_id = ?`, id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, biz.ErrUserNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	return r.getUser(ctx, id)
}

func (r *consoleRepo) getUser(ctx context.Context, id int64) (*biz.PlanstarUser, error) {
	user := &biz.PlanstarUser{}
	err := r.data.readDB.QueryRowContext(ctx, userSelect+" WHERE user_id = ?", id).
		Scan(&user.ID, &user.Name, &user.Mobile, &user.Avatar, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, biz.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

const projectSelect = `SELECT p.project_id, p.title, COALESCE(p.subtitle,''), p.category_id, COALESCE(c.name,''),
	       p.publish_status, p.funding_status, p.goal_amount, p.raised_amount, p.supporter_count,
	       COALESCE(cp.display_name,''), p.gmt_create
	  FROM planstar_catalog.project p
	  LEFT JOIN planstar_catalog.project_category c ON c.category_id = p.category_id
	  LEFT JOIN planstar_catalog.creator_profile cp ON cp.creator_id = p.creator_id`

func (r *consoleRepo) ListProjects(ctx context.Context, q biz.ProjectQuery) ([]*biz.Project, error) {
	query := projectSelect
	where, args := make([]string, 0, 2), make([]any, 0, 4)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(p.title LIKE ? OR p.subtitle LIKE ? OR p.project_no LIKE ? OR CAST(p.project_id AS CHAR) LIKE ?)")
		args = append(args, like, like, like, like)
	}
	if q.Status != nil {
		where = append(where, "p.publish_status = ?")
		args = append(args, *q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY p.gmt_create DESC, p.project_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]*biz.Project, 0, q.Limit)
	for rows.Next() {
		project := &biz.Project{}
		var goalAmount, raisedAmount float64
		if err := rows.Scan(&project.ID, &project.Title, &project.Subtitle, &project.CategoryID,
			&project.CategoryName, &project.PublishStatus, &project.FundingStatus,
			&goalAmount, &raisedAmount, &project.SupporterCount, &project.CreatorName,
			&project.CreatedAt); err != nil {
			return nil, err
		}
		project.GoalAmount = yuanToCents(goalAmount)
		project.AchievementPermille = achievementPermille(raisedAmount, goalAmount)
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

const orderSelect = `SELECT order_id, order_number, user_id, pay_amount, status, gmt_create
	  FROM planstar_order.order_info`

func (r *consoleRepo) ListOrders(ctx context.Context, q biz.OrderQuery) ([]*biz.Order, error) {
	query := orderSelect
	where, args := make([]string, 0, 2), make([]any, 0, 3)
	if q.Keyword != "" {
		like := likeArg(q.Keyword)
		where = append(where, "(order_number LIKE ? OR CAST(user_id AS CHAR) LIKE ?)")
		args = append(args, like, like)
	}
	if q.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *q.Status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY gmt_create DESC, order_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*biz.Order, 0, q.Limit)
	for rows.Next() {
		order := &biz.Order{}
		var payAmount float64
		if err := rows.Scan(&order.ID, &order.OrderNo, &order.UserID, &payAmount,
			&order.Status, &order.CreatedAt); err != nil {
			return nil, err
		}
		order.Amount = yuanToCents(payAmount)
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

const reportSelect = `SELECT report_id, target_type, target_id, COALESCE(project_id,0), user_id,
	       reason_code, COALESCE(description,''), COALESCE(evidence,'[]'), status,
	       COALESCE(handle_note,''), handle_time, gmt_create
	  FROM planstar_catalog.project_report`

func (r *consoleRepo) ListReports(ctx context.Context, q biz.ReportQuery) ([]*biz.Report, error) {
	query := reportSelect
	args := make([]any, 0, 3)
	if q.Status != "" {
		query += " WHERE status = ?"
		args = append(args, q.Status)
	}
	query += " ORDER BY gmt_create DESC, report_id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]*biz.Report, 0, q.Limit)
	for rows.Next() {
		report, err := scanConsoleReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func scanConsoleReport(sc interface {
	Scan(dest ...any) error
}) (*biz.Report, error) {
	report := &biz.Report{}
	var evidence string
	var handleTime sql.NullTime
	if err := sc.Scan(&report.ID, &report.TargetType, &report.TargetID, &report.ProjectID,
		&report.UserID, &report.ReasonCode, &report.Description, &evidence, &report.Status,
		&report.HandleNote, &handleTime, &report.CreatedAt); err != nil {
		return nil, err
	}
	report.Evidence = decodeStringSlice(evidence)
	if handleTime.Valid {
		t := handleTime.Time
		report.HandleTime = &t
	}
	return report, nil
}

func likeArg(keyword string) string {
	return "%" + keyword + "%"
}

func decodeStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

// yuanToCents mirrors the Planstar API convention: the database stores yuan,
// the wire format carries cents.
func yuanToCents(yuan float64) int64 {
	return int64(math.Round(yuan * 100))
}

func achievementPermille(raised, goal float64) int32 {
	if goal <= 0 {
		return 0
	}
	return int32(math.Round(raised / goal * 1000))
}
