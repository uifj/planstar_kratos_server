package data

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"planstar/internal/biz"
)

// ProjectIdeaRepo 实现 biz.ProjectIdeaRepo：创意与表态均落 planstar_catalog，
// 发起人昵称/头像关联 planstar_user.user_info。
type ProjectIdeaRepo struct {
	d *Data
}

func NewProjectIdeaRepo(d *Data) *ProjectIdeaRepo { return &ProjectIdeaRepo{d: d} }

// ideaSelect 是叠卡流投影（不含 description，避免列表多余负载）。
const ideaSelect = `SELECT i.idea_id, i.user_id, COALESCE(u.user_name,''), COALESCE(u.avatar,''),
       i.title, i.summary, COALESCE(i.cover,''), i.category_id, COALESCE(c.name,''),
       i.stage, i.support_count, i.oppose_count, i.coin_intent_sum, i.hot_score, i.gmt_create
  FROM planstar_catalog.project_idea i
  LEFT JOIN planstar_catalog.project_category c ON c.category_id = i.category_id
  LEFT JOIN planstar_user.user_info u ON u.user_id = i.user_id`

// ideaDetailSelect 在叠卡流投影后追加 description（固定位于最后一列）。
const ideaDetailSelect = `SELECT i.idea_id, i.user_id, COALESCE(u.user_name,''), COALESCE(u.avatar,''),
       i.title, i.summary, COALESCE(i.cover,''), i.category_id, COALESCE(c.name,''),
       i.stage, i.support_count, i.oppose_count, i.coin_intent_sum, i.hot_score, i.gmt_create,
       COALESCE(i.description,'')
  FROM planstar_catalog.project_idea i
  LEFT JOIN planstar_catalog.project_category c ON c.category_id = i.category_id
  LEFT JOIN planstar_user.user_info u ON u.user_id = i.user_id`

// Feed 按 hot_score / idea_id 双列游标分页，游标格式 "hotScore_ideaId"。
func (r *ProjectIdeaRepo) Feed(ctx context.Context, q biz.IdeaQuery) (*biz.IdeaFeedPage, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}

	query := ideaSelect + " WHERE i.publish_status = 1 AND i.status = 1"
	args := make([]any, 0, 4)
	if hot, ideaID, ok := parseIdeaCursor(q.Cursor); ok {
		query += " AND (i.hot_score < ? OR (i.hot_score = ? AND i.idea_id < ?))"
		args = append(args, hot, hot, ideaID)
	}
	// 多取一条用于判断 hasMore。
	query += " ORDER BY i.hot_score DESC, i.idea_id DESC LIMIT ?"
	args = append(args, q.Size+1)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ProjectIdea, 0, q.Size+1)
	for rows.Next() {
		item, err := scanIdea(rows, false)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page := &biz.IdeaFeedPage{Items: items}
	if len(items) > q.Size {
		page.Items = items[:q.Size]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = formatIdeaCursor(last)
	}
	if q.ViewerID > 0 && len(page.Items) > 0 {
		if err := attachIdeaVotes(ctx, db, q.ViewerID, page.Items); err != nil {
			return nil, err
		}
	}
	return page, nil
}

// Detail 返回创意详情：含档位分布；viewerID>0 时附带 myVote。
func (r *ProjectIdeaRepo) Detail(ctx context.Context, ideaID, viewerID int64) (*biz.ProjectIdea, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	item, err := scanIdea(db.QueryRowContext(ctx,
		ideaDetailSelect+" WHERE i.idea_id = ? AND i.publish_status = 1 AND i.status = 1", ideaID), true)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, biz.ErrIdeaNotFound
	}
	if item.TierDistribution, err = loadIdeaTierDistribution(ctx, db, ideaID); err != nil {
		return nil, err
	}
	if item.ReasonCounts, err = loadIdeaReasonCounts(ctx, db, ideaID); err != nil {
		return nil, err
	}
	if viewerID > 0 {
		if item.MyVote, err = loadIdeaVote(ctx, db, ideaID, viewerID); err != nil {
			return nil, err
		}
	}
	return item, nil
}

// Vote 事务内写入表态并更新计数；唯一键冲突视为已投票，不重复计数。
func (r *ProjectIdeaRepo) Vote(ctx context.Context, ideaID, userID int64, vote *biz.IdeaVote) (*biz.IdeaVoteResult, bool, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, false, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var ownerID int64
	err = tx.QueryRowContext(ctx,
		`SELECT user_id FROM planstar_catalog.project_idea
		  WHERE idea_id = ? AND publish_status = 1 AND status = 1 FOR UPDATE`, ideaID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return nil, false, biz.ErrIdeaNotFound
	}
	if err != nil {
		return nil, false, err
	}
	if ownerID == userID {
		return nil, false, biz.ErrIdeaSelfVote
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO planstar_catalog.project_idea_vote
		   (idea_id, user_id, vote_type, coin_tier, reason_code, reason_text, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		ideaID, userID, vote.VoteType, vote.CoinTier, vote.ReasonCode, vote.ReasonText); err != nil {
		if !isDuplicateKey(err) {
			return nil, false, err
		}
		// 并发/重复提交：回滚本次写入，回读最新聚合状态并标记已投票。
		_ = tx.Rollback()
		result, readErr := loadIdeaVoteResult(ctx, db, ideaID, userID)
		if readErr != nil {
			return nil, false, readErr
		}
		return result, true, nil
	}

	if vote.VoteType == 1 {
		_, err = tx.ExecContext(ctx,
			`UPDATE planstar_catalog.project_idea
			    SET support_count = support_count + 1, coin_intent_sum = coin_intent_sum + ?,
			        gmt_modified = NOW()
			  WHERE idea_id = ?`, vote.CoinTier, ideaID)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE planstar_catalog.project_idea
			    SET oppose_count = oppose_count + 1, gmt_modified = NOW()
			  WHERE idea_id = ?`, ideaID)
	}
	if err != nil {
		return nil, false, err
	}
	// hot_score 采用定时任务刷新（(support+oppose)/(小时+2)^0.6），避免投票事务内做全表统计。

	result, err := loadIdeaVoteResult(ctx, tx, ideaID, userID)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return result, false, nil
}

// scanIdea 把投影扫成 ProjectIdea；withDescription 要求查询末列为 description。
func scanIdea(sc interface {
	Scan(dest ...any) error
}, withDescription bool) (*biz.ProjectIdea, error) {
	d := &biz.ProjectIdea{}
	var hot float64
	var created time.Time
	dest := []any{
		&d.IdeaID, &d.UserID, &d.UserName, &d.UserAvatar, &d.Title, &d.Summary, &d.Cover,
		&d.CategoryID, &d.CategoryName, &d.Stage, &d.SupportCount, &d.OpposeCount,
		&d.CoinIntentSum, &hot, &created,
	}
	if withDescription {
		dest = append(dest, &d.Description)
	}
	if err := sc.Scan(dest...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	d.HotScore = hot
	d.GmtCreate = created.UnixMilli()
	d.SupportRate = biz.IdeaSupportRate(d.SupportCount, d.OpposeCount)
	return d, nil
}

// ideaQueryer 同时适配 *sql.DB 与 *sql.Tx。
type ideaQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadIdeaTierDistribution(ctx context.Context, q ideaQueryer, ideaID int64) (map[string]int, error) {
	var t5, t10, t20 int
	if err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(coin_tier = 5),0), COALESCE(SUM(coin_tier = 10),0),
		        COALESCE(SUM(coin_tier = 20),0)
		   FROM planstar_catalog.project_idea_vote
		  WHERE idea_id = ? AND vote_type = 1`, ideaID).Scan(&t5, &t10, &t20); err != nil {
		return nil, err
	}
	return map[string]int{"5": t5, "10": t10, "20": t20}, nil
}

// loadIdeaReasonCounts 聚合不支持理由的标准 code 计数（仅 reason_code，不读原文）。
func loadIdeaReasonCounts(ctx context.Context, q ideaQueryer, ideaID int64) (map[string]int, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT reason_code, COUNT(*)
		   FROM planstar_catalog.project_idea_vote
		  WHERE idea_id = ? AND vote_type = 2 AND reason_code IS NOT NULL AND reason_code <> ''
		  GROUP BY reason_code`, ideaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var code string
		var count int
		if err := rows.Scan(&code, &count); err != nil {
			return nil, err
		}
		counts[code] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func loadIdeaVote(ctx context.Context, q ideaQueryer, ideaID, userID int64) (*biz.IdeaVote, error) {
	vote := &biz.IdeaVote{}
	err := q.QueryRowContext(ctx,
		`SELECT vote_type, coin_tier, COALESCE(reason_code,'')
		   FROM planstar_catalog.project_idea_vote
		  WHERE idea_id = ? AND user_id = ?`, ideaID, userID).
		Scan(&vote.VoteType, &vote.CoinTier, &vote.ReasonCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return vote, nil
}

func loadIdeaVoteResult(ctx context.Context, q ideaQueryer, ideaID, userID int64) (*biz.IdeaVoteResult, error) {
	result := &biz.IdeaVoteResult{IdeaID: ideaID}
	if err := q.QueryRowContext(ctx,
		`SELECT support_count, oppose_count, coin_intent_sum
		   FROM planstar_catalog.project_idea WHERE idea_id = ?`, ideaID).
		Scan(&result.SupportCount, &result.OpposeCount, &result.CoinIntentSum); err != nil {
		return nil, err
	}
	result.SupportRate = biz.IdeaSupportRate(result.SupportCount, result.OpposeCount)
	dist, err := loadIdeaTierDistribution(ctx, q, ideaID)
	if err != nil {
		return nil, err
	}
	result.TierDistribution = dist
	if result.MyVote, err = loadIdeaVote(ctx, q, ideaID, userID); err != nil {
		return nil, err
	}
	return result, nil
}

// attachIdeaVotes 一次性补齐当前用户在本页创意上的表态。
func attachIdeaVotes(ctx context.Context, db *sql.DB, userID int64, items []*biz.ProjectIdea) error {
	ids := make([]any, 0, len(items)+1)
	ids = append(ids, userID)
	for _, item := range items {
		ids = append(ids, item.IdeaID)
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(items)), ",")
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`SELECT idea_id, vote_type, coin_tier, COALESCE(reason_code,'')
		               FROM planstar_catalog.project_idea_vote
		              WHERE user_id = ? AND idea_id IN (%s)`, marks), ids...)
	if err != nil {
		return err
	}
	defer rows.Close()

	votes := map[int64]*biz.IdeaVote{}
	for rows.Next() {
		var ideaID int64
		vote := &biz.IdeaVote{}
		if err := rows.Scan(&ideaID, &vote.VoteType, &vote.CoinTier, &vote.ReasonCode); err != nil {
			return err
		}
		votes[ideaID] = vote
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		item.MyVote = votes[item.IdeaID]
	}
	return nil
}

// formatIdeaCursor 把热度与业务主键编码为 "hotScore_ideaId"。
func formatIdeaCursor(item *biz.ProjectIdea) string {
	return strconv.FormatFloat(item.HotScore, 'f', 4, 64) + "_" + strconv.FormatInt(item.IdeaID, 10)
}

// parseIdeaCursor 解析游标；空或非法游标回退到首页。
func parseIdeaCursor(cursor string) (float64, int64, bool) {
	idx := strings.LastIndex(cursor, "_")
	if idx <= 0 || idx >= len(cursor)-1 {
		return 0, 0, false
	}
	hot, err := strconv.ParseFloat(cursor[:idx], 64)
	if err != nil {
		return 0, 0, false
	}
	ideaID, err := strconv.ParseInt(cursor[idx+1:], 10, 64)
	if err != nil || ideaID <= 0 {
		return 0, 0, false
	}
	return hot, ideaID, true
}
