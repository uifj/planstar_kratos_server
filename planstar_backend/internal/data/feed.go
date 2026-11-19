package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"planstar/internal/biz"
)

// FeedRepo 实现 biz.FeedRepo，落 planstar_community。
type FeedRepo struct {
	d *Data
}

func NewFeedRepo(d *Data) *FeedRepo { return &FeedRepo{d: d} }

const feedBase = `SELECT f.feed_id, COALESCE(f.title,''), COALESCE(f.content,'' ),
	       COALESCE(f.user,'{}'), COALESCE(f.activity,'{}'), COALESCE(f.tags,'[]'), COALESCE(f.images,'[]'),
	       COALESCE(f.likes,0), COALESCE(f.views,0), f.gmt_create,
	       EXISTS(SELECT 1 FROM planstar_community.feed_media m
	               WHERE m.feed_id = f.feed_id AND m.media_type = 'video') AS has_video
	  FROM planstar_community.feed f`

type feedRaw struct {
	FeedID       int64
	Title        string
	Content      string
	UserJSON     string
	ActivityJSON string
	TagsJSON     string
	ImagesJSON   string
	Likes        int
	Views        int
	CreateTime   time.Time
	HasVideo     int
}

func scanFeed(sc interface {
	Scan(dest ...any) error
}) (*feedRaw, error) {
	raw := &feedRaw{}
	err := sc.Scan(&raw.FeedID, &raw.Title, &raw.Content, &raw.UserJSON, &raw.ActivityJSON,
		&raw.TagsJSON, &raw.ImagesJSON, &raw.Likes, &raw.Views, &raw.CreateTime, &raw.HasVideo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (raw *feedRaw) post() *biz.FeedPost {
	var user struct {
		UserName string `json:"userName"`
	}
	_ = json.Unmarshal([]byte(raw.UserJSON), &user)

	var activity struct {
		ActivityID int64  `json:"activityId"`
		Name       string `json:"name"`
	}
	_ = json.Unmarshal([]byte(raw.ActivityJSON), &activity)

	var tags []struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal([]byte(raw.TagsJSON), &tags)
	tag := ""
	if len(tags) > 0 {
		tag = tags[0].Name
	}

	when := biz.FmtDT(raw.CreateTime)
	return &biz.FeedPost{
		ID: raw.FeedID, TweetID: raw.FeedID,
		Title: raw.Title, Content: raw.Content,
		Author: user.UserName, UserName: user.UserName,
		ViewCount: raw.Views, ReadCount: raw.Views,
		PublishedAt: when, CreateTime: when,
		Tag:        tag,
		ShowID:     activity.ActivityID,
		ActivityID: activity.ActivityID,
		LikeCount:  raw.Likes, LikeNum: raw.Likes,
		HasVideo: raw.HasVideo > 0, Video: raw.HasVideo > 0,
		Body:   raw.Content,
		Images: parseImageURLs(raw.ImagesJSON),
	}
}

func parseImageURLs(raw string) []string {
	var objs []struct {
		URL       string `json:"url"`
		Thumbnail string `json:"thumbnail"`
	}
	if err := json.Unmarshal([]byte(raw), &objs); err == nil {
		out := make([]string, 0, len(objs))
		for _, o := range objs {
			if o.URL != "" {
				out = append(out, o.URL)
			} else if o.Thumbnail != "" {
				out = append(out, o.Thumbnail)
			}
		}
		return out
	}
	var strs []string
	if err := json.Unmarshal([]byte(raw), &strs); err == nil {
		return strs
	}
	return []string{}
}

// Feeds 列表分页。nextPageKey 按「下一页页码」解释（客户端原样回传即可）。
func (r *FeedRepo) Feeds(ctx context.Context, q biz.FeedQuery, hot bool) ([]*biz.FeedPost, bool, int64, error) {
	db, err := r.d.DB("community")
	if err != nil {
		return nil, false, 0, err
	}

	page := q.Page
	if q.NextPageKey > 0 {
		page = int(q.NextPageKey)
	}
	offset := (page - 1) * q.PageSize

	order := "f.gmt_create DESC, f.feed_id DESC"
	if hot {
		order = "f.likes DESC, f.views DESC, f.feed_id DESC"
	}

	rows, err := db.QueryContext(ctx,
		feedBase+" WHERE f.status = 1 ORDER BY "+order+" LIMIT ? OFFSET ?",
		q.PageSize+1, offset)
	if err != nil {
		return nil, false, 0, err
	}
	defer rows.Close()

	items := make([]*biz.FeedPost, 0, q.PageSize)
	for rows.Next() {
		raw, err := scanFeed(rows)
		if err != nil {
			return nil, false, 0, err
		}
		items = append(items, raw.post())
	}
	if err := rows.Err(); err != nil {
		return nil, false, 0, err
	}

	hasMore := len(items) > q.PageSize
	if hasMore {
		items = items[:q.PageSize]
	}
	var nextKey int64
	if hasMore {
		nextKey = int64(page + 1)
	}
	return items, hasMore, nextKey, nil
}

func (r *FeedRepo) Feed(ctx context.Context, id int64) (*biz.FeedPost, error) {
	db, err := r.d.DB("community")
	if err != nil {
		return nil, err
	}
	raw, err := scanFeed(db.QueryRowContext(ctx, feedBase+" WHERE f.feed_id = ?", id))
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, biz.ErrNotFound
	}
	return raw.post(), nil
}

// Comments 取动态的评论：先按 feed 的 activityId 关联 review，失败再退回 tweetId = activity_id。
func (r *FeedRepo) Comments(ctx context.Context, tweetID int64) ([]*biz.FeedComment, error) {
	db, err := r.d.DB("community")
	if err != nil {
		return nil, err
	}

	var activityJSON string
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(activity,'{}') FROM feed WHERE feed_id = ?`, tweetID).Scan(&activityJSON)
	var activity struct {
		ActivityID int64 `json:"activityId"`
	}
	_ = json.Unmarshal([]byte(activityJSON), &activity)

	items, err := r.commentsByActivity(ctx, db, activity.ActivityID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 && activity.ActivityID != tweetID {
		return r.commentsByActivity(ctx, db, tweetID)
	}
	return items, nil
}

func (r *FeedRepo) commentsByActivity(ctx context.Context, db *sql.DB, activityID int64) ([]*biz.FeedComment, error) {
	if activityID <= 0 {
		return []*biz.FeedComment{}, nil
	}
	rows, err := db.QueryContext(ctx,
		`SELECT review_id, COALESCE(user,'{}'), COALESCE(content,''), COALESCE(likes,0), gmt_create
		   FROM review WHERE status = 1 AND activity_id = ? ORDER BY gmt_create DESC LIMIT 100`, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.FeedComment, 0, 16)
	for rows.Next() {
		c := &biz.FeedComment{}
		var userJSON string
		var when time.Time
		if err := rows.Scan(&c.ID, &userJSON, &c.Text, &c.LikeCount, &when); err != nil {
			return nil, err
		}
		var user struct {
			UserName string `json:"userName"`
		}
		_ = json.Unmarshal([]byte(userJSON), &user)
		c.Author = user.UserName
		c.UserName = user.UserName
		c.PublishedAt = biz.FmtDT(when)
		c.CreateTime = c.PublishedAt
		items = append(items, c)
	}
	return items, rows.Err()
}
