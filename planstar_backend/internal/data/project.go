package data

import (
	"context"
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

// Categories 返回启用的一级品类，并挂上各自的二级标签。
//
// 两张表都是几十行的枚举数据，两次全表扫描比分页/联表更简单；
// 标签按 category_id 归组，未挂到任何启用分类下的标签直接丢弃。
func (r *ProjectRepo) Categories(ctx context.Context) ([]*biz.ProjectCategory, error) {
	db, err := r.d.DB("catalog")
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx,
		`SELECT category_id, name, COALESCE(en_name,''), COALESCE(icon,''), sort, is_select, status
		   FROM planstar_catalog.project_category
		  WHERE status = 1
		  ORDER BY sort ASC, category_id ASC`)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.ProjectCategory, 0, 9)
	index := make(map[int64]*biz.ProjectCategory, 9)
	for rows.Next() {
		c := &biz.ProjectCategory{SubTags: []*biz.ProjectSubCategory{}}
		var isSelect int
		if err := rows.Scan(&c.ID, &c.Name, &c.EnName, &c.Icon, &c.Sort, &isSelect, &c.Status); err != nil {
			rows.Close()
			return nil, err
		}
		c.IsSelect = isSelect > 0
		items = append(items, c)
		index[c.ID] = c
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagRows, err := db.QueryContext(ctx,
		`SELECT sub_tag_id, category_id, name
		   FROM planstar_catalog.project_sub_tag
		  WHERE status = 1
		  ORDER BY sort ASC, sub_tag_id ASC`)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		tag := &biz.ProjectSubCategory{}
		if err := tagRows.Scan(&tag.SubCategoryID, &tag.CategoryID, &tag.Name); err != nil {
			return nil, err
		}
		if category, ok := index[tag.CategoryID]; ok {
			category.SubTags = append(category.SubTags, tag)
		}
	}
	return items, tagRows.Err()
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
