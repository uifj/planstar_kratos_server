package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"planstar/internal/biz"
)

// ProjectRepo 实现 biz.ProjectRepo：企划域读模型落 planstar_catalog，
// 支持单落 planstar_order，地址校验落 planstar_user。
type ProjectRepo struct {
	d *Data
}

func NewProjectRepo(d *Data) *ProjectRepo { return &ProjectRepo{d: d} }

// projectSelect 统一企划主档投影（含列表与详情字段），供多个查询复用。
const projectSelect = `SELECT p.project_id, p.project_no, p.title, COALESCE(p.subtitle,''), COALESCE(p.cover,''),
       p.category_id, COALESCE(c.name,''), p.project_type, p.creator_id, COALESCE(cp.display_name,''),
       p.goal_amount, p.raised_amount, p.supporter_count, p.deadline, p.funding_status,
       p.min_support_amount, COALESCE(p.tag_ids,'[]'), p.is_featured,
       COALESCE(p.gallery,'[]'), COALESCE(p.story,''), COALESCE(p.risk_note,''),
       COALESCE(p.delivery_plan,''), COALESCE(p.fund_usage,''), COALESCE(p.refund_rule,''), p.refund_days
  FROM planstar_catalog.project p
  LEFT JOIN planstar_catalog.project_category c ON c.category_id = p.category_id
  LEFT JOIN planstar_catalog.creator_profile cp ON cp.creator_id = p.creator_id`

func (r *ProjectRepo) List(ctx context.Context, q biz.ProjectQuery) ([]*biz.ProjectSummary, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	tags, err := loadTagNames(ctx, db)
	if err != nil {
		return nil, err
	}

	query := projectSelect + " WHERE p.publish_status = 1 AND p.status = 1"
	args := make([]any, 0, 5)
	if q.CategoryID > 0 {
		query += " AND p.category_id = ?"
		args = append(args, q.CategoryID)
	}
	if q.FundingStatus != "" {
		query += " AND p.funding_status = ?"
		args = append(args, q.FundingStatus)
	}
	query += " ORDER BY " + projectOrderBy(q) + " LIMIT ? OFFSET ?"
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ProjectSummary, 0, q.PageSize)
	for rows.Next() {
		d, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		applyTagNames(&d.ProjectSummary, tags)
		items = append(items, &d.ProjectSummary)
	}
	return items, rows.Err()
}

func projectOrderBy(q biz.ProjectQuery) string {
	switch {
	case q.Hot:
		return "p.is_featured DESC, p.rank_weight DESC, p.supporter_count DESC, p.project_id DESC"
	case q.Sort == "new":
		return "p.start_time DESC, p.project_id DESC"
	case q.Sort == "ending":
		return "p.deadline ASC, p.project_id DESC"
	case q.Sort == "achievement":
		return "(p.raised_amount / NULLIF(p.goal_amount, 0)) DESC, p.project_id DESC"
	default:
		return "p.is_featured DESC, p.rank_weight DESC, p.supporter_count DESC, p.project_id DESC"
	}
}

func (r *ProjectRepo) Detail(ctx context.Context, projectID int64) (*biz.ProjectDetail, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	tags, err := loadTagNames(ctx, db)
	if err != nil {
		return nil, err
	}
	d, err := scanProject(db.QueryRowContext(ctx,
		projectSelect+" WHERE p.project_id = ? AND p.publish_status = 1 AND p.status = 1", projectID))
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, biz.ErrNotFound
	}
	applyTagNames(&d.ProjectSummary, tags)

	if d.Faq, err = loadFaqs(ctx, db, projectID); err != nil {
		return nil, err
	}
	if d.RewardTiers, err = loadRewardTiers(ctx, db, projectID); err != nil {
		return nil, err
	}
	d.Updates, err = loadUpdates(ctx, db, projectID, false, 1, 5)
	if err != nil {
		return nil, err
	}
	if d.StretchGoals, err = loadStretchGoals(ctx, db, projectID); err != nil {
		return nil, err
	}
	if d.Creator, err = loadCreator(ctx, db, d.CreatorID); err != nil {
		return nil, err
	}
	stats, err := loadReviewStats(ctx, db, projectID)
	if err != nil {
		return nil, err
	}
	d.RatingAvg = stats.RatingAvg
	d.ReviewCount = stats.ReviewCount
	d.ShowCount = stats.ShowCount
	return d, nil
}

func (r *ProjectRepo) RewardTiers(ctx context.Context, projectID int64) ([]*biz.RewardTier, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	exists, err := projectExists(ctx, db, projectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrNotFound
	}
	return loadRewardTiers(ctx, db, projectID)
}

func (r *ProjectRepo) Updates(ctx context.Context, projectID int64, includeSupporterOnly bool, page, pageSize int) ([]*biz.ProjectUpdate, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	return loadUpdates(ctx, db, projectID, includeSupporterOnly, page, pageSize)
}

func (r *ProjectRepo) HasSupporterOnly(ctx context.Context, projectID int64) (bool, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return false, err
	}
	var cnt int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM planstar_catalog.project_update
		  WHERE project_id = ? AND status = 1 AND is_supporter_only = 1`, projectID).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *ProjectRepo) IsSupporter(ctx context.Context, projectID, userID int64) (bool, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return false, err
	}
	var cnt int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM planstar_order.pledge
		  WHERE project_id = ? AND user_id = ?
		    AND pledge_status IN ('paid','confirmed','shipped','delivered')`,
		projectID, userID).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *ProjectRepo) Faqs(ctx context.Context, projectID int64) ([]*biz.ProjectFaq, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	exists, err := projectExists(ctx, db, projectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrNotFound
	}
	return loadFaqs(ctx, db, projectID)
}

func (r *ProjectRepo) Creator(ctx context.Context, creatorID int64) (*biz.CreatorPage, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	profile, err := loadCreator(ctx, db, creatorID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, biz.ErrNotFound
	}

	tags, err := loadTagNames(ctx, db)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		projectSelect+` WHERE p.creator_id = ? AND p.publish_status = 1 AND p.status = 1
		 ORDER BY p.rank_weight DESC, p.project_id DESC`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]*biz.ProjectSummary, 0, 8)
	for rows.Next() {
		d, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		applyTagNames(&d.ProjectSummary, tags)
		projects = append(projects, &d.ProjectSummary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &biz.CreatorPage{CreatorProfile: *profile, Projects: projects}, nil
}

func (r *ProjectRepo) Reviews(ctx context.Context, projectID int64, sort string, onlyShow bool, page, pageSize int) (*biz.ReviewPage, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	exists, err := projectExists(ctx, db, projectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrNotFound
	}

	stats, err := loadReviewStats(ctx, db, projectID)
	if err != nil {
		return nil, err
	}

	query := `SELECT r.review_id, r.project_id, COALESCE(u.user_name,''), COALESCE(u.avatar,''),
	                 r.rating, COALESCE(r.content,''), COALESCE(r.images,'[]'),
	                 r.is_show, r.is_anonymous, r.like_count, r.gmt_create
	            FROM planstar_catalog.project_review r
	            LEFT JOIN planstar_user.user_info u ON u.user_id = r.user_id
	           WHERE r.project_id = ? AND r.status = 1`
	if onlyShow {
		query += " AND r.is_show = 1"
	}
	if sort == "hot" {
		query += " ORDER BY r.like_count DESC, r.gmt_create DESC, r.review_id DESC"
	} else {
		query += " ORDER BY r.gmt_create DESC, r.review_id DESC"
	}
	query += " LIMIT ? OFFSET ?"

	rows, err := db.QueryContext(ctx, query, projectID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ProjectReview, 0, pageSize)
	for rows.Next() {
		item, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &biz.ReviewPage{
		Items:       items,
		RatingAvg:   stats.RatingAvg,
		ReviewCount: stats.ReviewCount,
		ShowCount:   stats.ShowCount,
	}, nil
}

func (r *ProjectRepo) CreateReview(ctx context.Context, userID, projectID int64, in *biz.ReviewInput) (*biz.ProjectReview, error) {
	orderDB, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	var ownerID, pledgeProjectID int64
	var pledgeStatus string
	err = orderDB.QueryRowContext(ctx,
		`SELECT user_id, project_id, pledge_status FROM planstar_order.pledge WHERE pledge_id = ?`,
		in.PledgeID).Scan(&ownerID, &pledgeProjectID, &pledgeStatus)
	if err == sql.ErrNoRows {
		return nil, biz.ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	if ownerID != userID || pledgeProjectID != projectID || pledgeStatus != "delivered" {
		return nil, biz.ErrForbidden
	}

	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	var cnt int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM planstar_catalog.project_review WHERE pledge_id = ?`, in.PledgeID).Scan(&cnt); err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, biz.ErrConflict
	}

	status := 1
	if biz.IsReviewRisky(in.Content) {
		status = 2
	}
	images, err := json.Marshal(in.Images)
	if err != nil {
		return nil, err
	}

	reviewID, err := nextID(ctx, db, "planstar_catalog.project_review", "review_id")
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO planstar_catalog.project_review
		   (review_id, project_id, pledge_id, user_id, rating, content, images, is_show, is_anonymous,
		    like_count, status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, NOW(), NOW())`,
		reviewID, projectID, in.PledgeID, userID, in.Rating, in.Content, string(images),
		boolInt(in.IsShow), boolInt(in.IsAnonymous), status); err != nil {
		return nil, err
	}

	item := &biz.ProjectReview{
		ReviewID:    reviewID,
		ProjectID:   projectID,
		Rating:      in.Rating,
		Content:     in.Content,
		Images:      in.Images,
		IsShow:      in.IsShow,
		IsAnonymous: in.IsAnonymous,
		GmtCreate:   time.Now().UnixMilli(),
	}
	if !in.IsAnonymous {
		userDB, err := r.d.DB("user")
		if err != nil {
			return nil, err
		}
		_ = userDB.QueryRowContext(ctx,
			`SELECT COALESCE(user_name,''), COALESCE(avatar,'') FROM user_info WHERE user_id = ?`, userID).
			Scan(&item.UserName, &item.UserAvatar)
	}
	return item, nil
}

func (r *ProjectRepo) CreatePledge(ctx context.Context, userID, projectID int64, in *biz.PledgeInput) (*biz.Pledge, error) {
	catalogDB, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	// 1. 企划必须处于可支持状态。
	var fundingStatus string
	err = catalogDB.QueryRowContext(ctx,
		`SELECT funding_status FROM planstar_catalog.project
		  WHERE project_id = ? AND publish_status = 1 AND status = 1`, projectID).Scan(&fundingStatus)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if fundingStatus != "preheating" && fundingStatus != "ongoing" {
		return nil, biz.ErrConflict
	}

	// 2. 档位必须属于该企划且启用。
	var price float64
	var rewardType string
	var isLimited, remaining int
	err = catalogDB.QueryRowContext(ctx,
		`SELECT price, reward_type, is_limited, remaining_num
		   FROM planstar_catalog.reward_tier
		  WHERE reward_tier_id = ? AND project_id = ? AND status = 1`,
		in.RewardTierID, projectID).Scan(&price, &rewardType, &isLimited, &remaining)
	if err == sql.ErrNoRows {
		return nil, biz.ErrBadRequest
	}
	if err != nil {
		return nil, err
	}
	if isLimited > 0 && remaining < in.Quantity {
		return nil, biz.ErrConflict
	}

	deliveryType := pledgeDeliveryType(rewardType)
	contactName, contactPhone := "", ""
	addressID := in.AddressID
	if deliveryType == "physical" {
		if in.AddressID <= 0 {
			return nil, biz.ErrBadRequest
		}
		userDB, err := r.d.DB("user")
		if err != nil {
			return nil, err
		}
		err = userDB.QueryRowContext(ctx,
			`SELECT COALESCE(name,''), COALESCE(phone,'') FROM user_address
			  WHERE user_id = ? AND address_id = ? AND status = 1`,
			userID, in.AddressID).Scan(&contactName, &contactPhone)
		if err == sql.ErrNoRows {
			return nil, biz.ErrBadRequest
		}
		if err != nil {
			return nil, err
		}
	} else {
		addressID = 0
	}

	// 3. 原子扣减库存（限量档位），失败即冲突。
	if isLimited > 0 {
		res, err := catalogDB.ExecContext(ctx,
			`UPDATE planstar_catalog.reward_tier
			    SET sold_num = sold_num + ?, remaining_num = remaining_num - ?, gmt_modified = NOW()
			  WHERE reward_tier_id = ? AND is_limited = 1 AND remaining_num >= ?`,
			in.Quantity, in.Quantity, in.RewardTierID, in.Quantity)
		if err != nil {
			return nil, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected == 0 {
			return nil, biz.ErrConflict
		}
	} else if _, err := catalogDB.ExecContext(ctx,
		`UPDATE planstar_catalog.reward_tier
		    SET sold_num = sold_num + ?, gmt_modified = NOW()
		  WHERE reward_tier_id = ?`, in.Quantity, in.RewardTierID); err != nil {
		return nil, err
	}

	orderDB, err := r.d.DB("order")
	if err != nil {
		r.restoreStock(ctx, catalogDB, in.RewardTierID, in.Quantity, isLimited > 0)
		return nil, err
	}
	pledgeID, err := nextID(ctx, orderDB, "planstar_order.pledge", "pledge_id")
	if err != nil {
		r.restoreStock(ctx, catalogDB, in.RewardTierID, in.Quantity, isLimited > 0)
		return nil, err
	}
	amount := math.Round(price*float64(in.Quantity)*100) / 100
	var addressValue any
	if addressID > 0 {
		addressValue = addressID
	}
	if _, err := orderDB.ExecContext(ctx,
		`INSERT INTO planstar_order.pledge
		   (pledge_id, pledge_no, project_id, reward_tier_id, user_id, support_amount, quantity,
		    is_anonymous, support_message, contact_name, contact_phone, address_id, delivery_type,
		    pledge_status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', NOW(), NOW())`,
		pledgeID, pledgeNo(pledgeID), projectID, in.RewardTierID, userID, amount, in.Quantity,
		boolInt(in.IsAnonymous), in.Message, contactName, contactPhone, addressValue, deliveryType); err != nil {
		r.restoreStock(ctx, catalogDB, in.RewardTierID, in.Quantity, isLimited > 0)
		return nil, err
	}

	return r.Pledge(ctx, userID, pledgeID)
}

func (r *ProjectRepo) restoreStock(ctx context.Context, db *sql.DB, tierID int64, quantity int, limited bool) {
	if limited {
		_, _ = db.ExecContext(ctx,
			`UPDATE planstar_catalog.reward_tier
			    SET sold_num = GREATEST(sold_num - ?, 0),
			        remaining_num = remaining_num + ?, gmt_modified = NOW()
			  WHERE reward_tier_id = ?`, quantity, quantity, tierID)
		return
	}
	_, _ = db.ExecContext(ctx,
		`UPDATE planstar_catalog.reward_tier
		    SET sold_num = GREATEST(sold_num - ?, 0), gmt_modified = NOW()
		  WHERE reward_tier_id = ?`, quantity, tierID)
}

func (r *ProjectRepo) Pledges(ctx context.Context, userID int64, status string, page, pageSize int) ([]*biz.Pledge, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	query := pledgeSelect + " WHERE user_id = ?"
	args := []any{userID}
	if status != "" {
		query += " AND pledge_status = ?"
		args = append(args, status)
	}
	query += " ORDER BY gmt_create DESC, pledge_id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.Pledge, 0, pageSize)
	for rows.Next() {
		item, err := scanPledge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ProjectRepo) Pledge(ctx context.Context, userID, pledgeID int64) (*biz.Pledge, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	item, err := scanPledge(db.QueryRowContext(ctx,
		pledgeSelect+" WHERE pledge_id = ? AND user_id = ?", pledgeID, userID))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, biz.ErrNotFound
	}
	return item, nil
}

func (r *ProjectRepo) CreateReport(ctx context.Context, userID, projectID int64, in *biz.ReportInput) (*biz.ProjectReport, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	var cnt int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM planstar_catalog.project_report
		  WHERE user_id = ? AND target_type = ? AND target_id = ?
		    AND gmt_create >= DATE_SUB(NOW(), INTERVAL 24 HOUR)`,
		userID, in.TargetType, in.TargetID).Scan(&cnt); err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, biz.ErrConflict
	}

	evidence, err := json.Marshal(in.Evidence)
	if err != nil {
		return nil, err
	}
	reportID, err := nextID(ctx, db, "planstar_catalog.project_report", "report_id")
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO planstar_catalog.project_report
		   (report_id, target_type, target_id, project_id, user_id, reason_code, description,
		    evidence, status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)`,
		reportID, in.TargetType, in.TargetID, projectID, userID, in.ReasonCode, in.Description,
		string(evidence), now, now); err != nil {
		return nil, err
	}
	return &biz.ProjectReport{
		ReportID:    reportID,
		TargetType:  in.TargetType,
		TargetID:    in.TargetID,
		ReasonCode:  in.ReasonCode,
		Description: in.Description,
		Evidence:    in.Evidence,
		Status:      "pending",
		GmtCreate:   now.UnixMilli(),
	}, nil
}

func (r *ProjectRepo) Ranking(ctx context.Context, typ string, categoryID int64, limit int) ([]*biz.RankingItem, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	tags, err := loadTagNames(ctx, db)
	if err != nil {
		return nil, err
	}

	query := projectSelect + ` WHERE p.publish_status = 1 AND p.status = 1
		AND p.funding_status IN ('ongoing','achieved')`
	args := make([]any, 0, 2)
	if categoryID > 0 {
		query += " AND p.category_id = ?"
		args = append(args, categoryID)
	}
	switch typ {
	case "achievement":
		query += " ORDER BY (p.raised_amount / NULLIF(p.goal_amount, 0)) DESC, p.project_id DESC"
	case "ending":
		query += " ORDER BY p.deadline ASC, p.project_id DESC"
	default:
		query += " ORDER BY p.supporter_count DESC, p.rank_weight DESC, p.project_id DESC"
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.RankingItem, 0, limit)
	for rows.Next() {
		d, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		applyTagNames(&d.ProjectSummary, tags)
		metric := int64(d.SupporterCount)
		if typ == "achievement" {
			metric = int64(d.AchievementRate)
		} else if typ == "ending" {
			metric = int64(d.RemainingDays)
		}
		items = append(items, &biz.RankingItem{
			Rank:        len(items) + 1,
			Project:     &d.ProjectSummary,
			MetricValue: metric,
		})
	}
	return items, rows.Err()
}

func (r *ProjectRepo) Moderation(ctx context.Context, projectID int64) (*biz.ProjectModeration, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}
	mod := &biz.ProjectModeration{}
	var goalAmount float64
	var story, fundUsage, summary, deliveryPlan, riskNote string
	err = db.QueryRowContext(ctx,
		`SELECT p.project_id, p.title, COALESCE(p.subtitle,''), p.category_id, COALESCE(c.name,''),
		        p.publish_status, p.funding_status, p.goal_amount,
		        COALESCE(p.story,''), COALESCE(p.fund_usage,''), COALESCE(p.summary,''),
		        COALESCE(p.delivery_plan,''), COALESCE(p.risk_note,'')
		   FROM planstar_catalog.project p
		   LEFT JOIN planstar_catalog.project_category c ON c.category_id = p.category_id
		  WHERE p.project_id = ?`, projectID).
		Scan(&mod.ProjectID, &mod.Title, &mod.Subtitle, &mod.CategoryID, &mod.CategoryName,
			&mod.PublishStatus, &mod.FundingStatus, &goalAmount,
			&story, &fundUsage, &summary, &deliveryPlan, &riskNote)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	content := strings.Join([]string{mod.Title, mod.Subtitle, mod.CategoryName, story, fundUsage, summary}, " ")
	mod.Hits = biz.CheckProjectBlacklist(content, biz.Fen(goalAmount), deliveryPlan, riskNote)
	mod.Blocked = len(mod.Hits) > 0
	return mod, nil
}

func (r *ProjectRepo) SetPublishStatus(ctx context.Context, projectID int64, publishStatus int) error {
	db, err := r.d.DB("catalog")
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE planstar_catalog.project SET publish_status = ?, gmt_modified = NOW()
		  WHERE project_id = ?`, publishStatus, projectID); err != nil {
		return err
	}
	return nil
}

// scanProject 把 projectSelect 的一行扫成 ProjectDetail（列表只取内嵌 Summary）。
// 标签 ID 暂存于 ProjectSummary.TagIDs，由调用方按字典补名称。
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
func nextID(ctx context.Context, db *sql.DB, table, column string) (int64, error) {
	var id int64
	if err := db.QueryRowContext(ctx,
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
