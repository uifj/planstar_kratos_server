package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"planstar/internal/biz"
	"strings"
	"time"
)

func scanProject(sc interface {
	Scan(dest ...any) error
}) (*biz.ProjectDetail, error) {
	d := &biz.ProjectDetail{}
	var (
		goalAmount, raisedAmount, minSupport float64
		deadline                             time.Time
		tagIDs, gallery                      string
		isFeatured                           int
	)
	err := sc.Scan(&d.ProjectID, &d.ProjectNo, &d.Title, &d.Subtitle, &d.Cover,
		&d.CategoryID, &d.CategoryName, &d.ProjectType, &d.CreatorID, &d.CreatorName,
		&goalAmount, &raisedAmount, &d.SupporterCount, &deadline, &d.FundingStatus,
		&minSupport, &tagIDs, &isFeatured,
		&gallery, &d.Story, &d.RiskNote, &d.DeliveryPlan, &d.FundUsage, &d.RefundRule, &d.RefundDays)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	d.GoalAmount = biz.Fen(goalAmount)
	d.RaisedAmount = biz.Fen(raisedAmount)
	d.MinSupportAmount = biz.Fen(minSupport)
	d.Deadline = deadline.UnixMilli()
	d.RemainingDays = biz.RemainingDays(deadline)
	d.AchievementRate = biz.AchievementPermille(raisedAmount, goalAmount)
	d.IsFeatured = isFeatured > 0
	d.Gallery = jsonStringSlice(gallery)
	d.TagIDs = jsonIntSlice(tagIDs)
	return d, nil
}

func applyTagNames(s *biz.ProjectSummary, names map[int64]string) {
	tags := make([]string, 0, len(s.TagIDs))
	for _, id := range s.TagIDs {
		if name, ok := names[id]; ok && name != "" {
			tags = append(tags, name)
		}
	}
	s.Tags = tags
}

func loadTagNames(ctx context.Context, db *sql.DB) (map[int64]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT sub_tag_id, name FROM planstar_catalog.project_sub_tag WHERE status = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := map[int64]string{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

func projectExists(ctx context.Context, db *sql.DB, projectID int64) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx,
		`SELECT 1 FROM planstar_catalog.project WHERE project_id = ? AND publish_status = 1 AND status = 1`,
		projectID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func loadRewardTiers(ctx context.Context, db *sql.DB, projectID int64) ([]*biz.RewardTier, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT reward_tier_id, title, price, reward_type, COALESCE(items,'[]'),
		        limit_num, sold_num, remaining_num, estimated_delivery_time, is_early_bird, is_limited
		   FROM planstar_catalog.reward_tier
		  WHERE project_id = ? AND status = 1
		  ORDER BY sort ASC, reward_tier_id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.RewardTier, 0, 5)
	for rows.Next() {
		t := &biz.RewardTier{}
		var price float64
		var itemsJSON string
		var delivery sql.NullTime
		var earlyBird, limited int
		if err := rows.Scan(&t.RewardTierID, &t.Title, &price, &t.RewardType, &itemsJSON,
			&t.LimitNum, &t.SoldNum, &t.RemainingNum, &delivery, &earlyBird, &limited); err != nil {
			return nil, err
		}
		t.Price = biz.Fen(price)
		t.Items = jsonStringSlice(itemsJSON)
		if delivery.Valid {
			t.EstimatedDeliveryTime = delivery.Time.Format("2006-01-02")
		}
		t.IsEarlyBird = earlyBird > 0
		t.IsLimited = limited > 0
		items = append(items, t)
	}
	return items, rows.Err()
}

func loadUpdates(ctx context.Context, db *sql.DB, projectID int64, includeSupporterOnly bool, page, pageSize int) ([]*biz.ProjectUpdate, error) {
	query := `SELECT update_id, title, COALESCE(content,''), COALESCE(images,'[]'),
	                 is_supporter_only, publish_time
	            FROM planstar_catalog.project_update
	           WHERE project_id = ? AND status = 1`
	if !includeSupporterOnly {
		query += " AND is_supporter_only = 0"
	}
	query += " ORDER BY publish_time DESC, update_id DESC LIMIT ? OFFSET ?"

	rows, err := db.QueryContext(ctx, query, projectID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ProjectUpdate, 0, pageSize)
	for rows.Next() {
		u := &biz.ProjectUpdate{}
		var images string
		var supporterOnly int
		var publishTime time.Time
		if err := rows.Scan(&u.UpdateID, &u.Title, &u.Content, &images, &supporterOnly, &publishTime); err != nil {
			return nil, err
		}
		u.Images = jsonStringSlice(images)
		u.IsSupporterOnly = supporterOnly > 0
		u.PublishTime = publishTime.UnixMilli()
		items = append(items, u)
	}
	return items, rows.Err()
}

func loadStretchGoals(ctx context.Context, db *sql.DB, projectID int64) ([]*biz.StretchGoal, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT goal_id, title, COALESCE(description,''), target_amount, current_amount, is_achieved, sort
		   FROM planstar_catalog.stretch_goal
		  WHERE project_id = ? AND status = 1
		  ORDER BY sort ASC, goal_id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.StretchGoal, 0, 4)
	for rows.Next() {
		g := &biz.StretchGoal{}
		var target, current float64
		var achieved int
		if err := rows.Scan(&g.GoalID, &g.Title, &g.Description, &target, &current, &achieved, &g.Sort); err != nil {
			return nil, err
		}
		g.TargetAmount = biz.Fen(target)
		g.CurrentAmount = biz.Fen(current)
		g.IsAchieved = achieved > 0
		items = append(items, g)
	}
	return items, rows.Err()
}

func loadFaqs(ctx context.Context, db *sql.DB, projectID int64) ([]*biz.ProjectFaq, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT faq_id, question, answer, sort FROM planstar_catalog.project_faq
		  WHERE project_id = ? AND status = 1
		  ORDER BY sort ASC, faq_id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ProjectFaq, 0, 4)
	for rows.Next() {
		f := &biz.ProjectFaq{}
		if err := rows.Scan(&f.FaqID, &f.Question, &f.Answer, &f.Sort); err != nil {
			return nil, err
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items, nil
	}

	// 字典表为空时回退到 project.faq JSON，保证详情页与 /faqs 同源。
	var raw string
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(faq,'[]') FROM planstar_catalog.project WHERE project_id = ?`, projectID).Scan(&raw)
	if err == sql.ErrNoRows {
		return items, nil
	}
	if err != nil {
		return nil, err
	}
	var legacy []struct {
		Q string `json:"q"`
		A string `json:"a"`
	}
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return items, nil
	}
	for i, item := range legacy {
		items = append(items, &biz.ProjectFaq{FaqID: int64(i + 1), Question: item.Q, Answer: item.A, Sort: i + 1})
	}
	return items, nil
}

func loadCreator(ctx context.Context, db *sql.DB, creatorID int64) (*biz.CreatorProfile, error) {
	p := &biz.CreatorProfile{}
	var verified int
	var rate float64
	err := db.QueryRowContext(ctx,
		`SELECT creator_id, display_name, COALESCE(avatar,''), is_verified, project_num, success_num,
		        fulfillment_rate, COALESCE(bio,'')
		   FROM planstar_catalog.creator_profile
		  WHERE creator_id = ? AND status = 1`, creatorID).
		Scan(&p.CreatorID, &p.DisplayName, &p.Avatar, &verified, &p.ProjectNum, &p.SuccessNum, &rate, &p.Bio)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IsVerified = verified > 0
	p.FulfillmentRate = int(math.Round(rate * 10))
	return p, nil
}

func loadReviewStats(ctx context.Context, db *sql.DB, projectID int64) (*biz.ReviewPage, error) {
	page := &biz.ReviewPage{}
	var avg float64
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*), COALESCE(SUM(is_show = 1),0)
		   FROM planstar_catalog.project_review
		  WHERE project_id = ? AND status = 1`, projectID).
		Scan(&avg, &page.ReviewCount, &page.ShowCount); err != nil {
		return nil, err
	}
	// 5 分制 -> 10 分制，保留一位小数。
	page.RatingAvg = math.Round(avg*2*10) / 10
	return page, nil
}

func scanReview(sc interface {
	Scan(dest ...any) error
}) (*biz.ProjectReview, error) {
	item := &biz.ProjectReview{}
	var images string
	var isShow, isAnonymous int
	var created time.Time
	err := sc.Scan(&item.ReviewID, &item.ProjectID, &item.UserName, &item.UserAvatar, &item.Rating,
		&item.Content, &images, &isShow, &isAnonymous, &item.LikeCount, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.Images = jsonStringSlice(images)
	item.IsShow = isShow > 0
	item.IsAnonymous = isAnonymous > 0
	if item.IsAnonymous {
		item.UserName = ""
		item.UserAvatar = ""
	}
	item.GmtCreate = created.UnixMilli()
	return item, nil
}

const pledgeSelect = `SELECT pledge_id, pledge_no, project_id, reward_tier_id, support_amount, quantity,
       is_anonymous, COALESCE(support_message,''), delivery_type, pledge_status,
       paid_time, delivered_time, COALESCE(tracking_no,''), COALESCE(content_url,''),
       COALESCE(redeem_code,''), redeemed_time
  FROM planstar_order.pledge`

func scanPledge(sc interface {
	Scan(dest ...any) error
}) (*biz.Pledge, error) {
	item := &biz.Pledge{}
	var amount float64
	var anonymous int
	var paid, delivered, redeemed sql.NullTime
	err := sc.Scan(&item.PledgeID, &item.PledgeNo, &item.ProjectID, &item.RewardTierID, &amount, &item.Quantity,
		&anonymous, &item.SupportMessage, &item.DeliveryType, &item.PledgeStatus,
		&paid, &delivered, &item.TrackingNo, &item.ContentURL, &item.RedeemCode, &redeemed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.SupportAmount = biz.Fen(amount)
	item.IsAnonymous = anonymous > 0
	item.PaidTime = nullTimeMilli(paid)
	item.DeliveredTime = nullTimeMilli(delivered)
	item.RedeemedTime = nullTimeMilli(redeemed)
	return item, nil
}

func nullTimeMilli(t sql.NullTime) int64 {
	if !t.Valid {
		return 0
	}
	return t.Time.UnixMilli()
}

// pledgeDeliveryType 把回报类型映射为履约方式：实体/混合走物理履约。
func pledgeDeliveryType(rewardType string) string {
	switch rewardType {
	case "physical", "mixed":
		return "physical"
	case "digital", "right", "offline":
		return rewardType
	default:
		return "none"
	}
}

func pledgeNo(pledgeID int64) string {
	return fmt.Sprintf("P%s%06d", time.Now().Format("20060102150405"), pledgeID%1000000)
}

// nextID 取业务主键的 MAX+1（与既有 user_address 的写法保持一致）。
func nextID(ctx context.Context, q rowQuerier, table, column string) (int64, error) {
	var id int64
	if err := q.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COALESCE(MAX(%s),0)+1 FROM %s", column, table)).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func jsonStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func jsonIntSlice(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return []int64{}
	}
	var out []int64
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []int64{}
	}
	return out
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
