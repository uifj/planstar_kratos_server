package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"planstar/internal/biz"
	"strings"
	"time"
)

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

	// 3. 单事务完成「扣库存 → 写支持单 → 更新企划聚合指标」。
	// 所有 planstar_* 库位于同一 MySQL 实例，跨 schema DML 在同一 *sql.Tx 内原子提交；
	// 不再使用「先提交扣库存、失败再用已取消 ctx 手工补偿」的跨连接池写法。
	tx, err := catalogDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if isLimited > 0 {
		res, err := tx.ExecContext(ctx,
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
	} else if _, err := tx.ExecContext(ctx,
		`UPDATE planstar_catalog.reward_tier
		    SET sold_num = sold_num + ?, gmt_modified = NOW()
		  WHERE reward_tier_id = ?`, in.Quantity, in.RewardTierID); err != nil {
		return nil, err
	}

	pledgeID, err := nextID(ctx, tx, "planstar_order.pledge", "pledge_id")
	if err != nil {
		return nil, err
	}
	amount := math.Round(price*float64(in.Quantity)*100) / 100
	var addressValue any
	if addressID > 0 {
		addressValue = addressID
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO planstar_order.pledge
		   (pledge_id, pledge_no, project_id, reward_tier_id, user_id, support_amount, quantity,
		    is_anonymous, support_message, contact_name, contact_phone, address_id, delivery_type,
		    pledge_status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', NOW(), NOW())`,
		pledgeID, pledgeNo(pledgeID), projectID, in.RewardTierID, userID, amount, in.Quantity,
		boolInt(in.IsAnonymous), in.Message, contactName, contactPhone, addressValue, deliveryType); err != nil {
		return nil, err
	}
	// 聚合指标与库存、支持单在同一事务推进，避免众筹进度永久停留在种子值。
	if _, err := tx.ExecContext(ctx,
		`UPDATE planstar_catalog.project
		    SET raised_amount = raised_amount + ?, supporter_count = supporter_count + 1,
		        gmt_modified = NOW()
		  WHERE project_id = ?`, amount, projectID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Pledge(ctx, userID, pledgeID)
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
