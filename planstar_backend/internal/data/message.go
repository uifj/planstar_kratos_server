package data

import (
	"context"
	"database/sql"
	"time"

	"planstar/internal/biz"
)

// MessageRepo 实现 biz.MessageRepo，落 planstar_user.user_message。
type MessageRepo struct {
	d *Data
}

func NewMessageRepo(d *Data) *MessageRepo { return &MessageRepo{d: d} }

func (r *MessageRepo) Messages(ctx context.Context, userID int64, msgType, pageIndex, pageSize int) ([]*biz.MessageThread, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	where := "user_id = ? AND status = 1"
	args := []any{userID}
	if msgType > 0 {
		where += " AND msg_type = ?"
		args = append(args, msgType)
	}
	args = append(args, pageSize, (pageIndex-1)*pageSize)

	rows, err := db.QueryContext(ctx,
		`SELECT message_id, COALESCE(title,''), COALESCE(content,''), COALESCE(msg_type,0),
		        COALESCE(unread_count,0), gmt_create, gmt_modified
		   FROM user_message WHERE `+where+`
		ORDER BY unread_count DESC, gmt_create DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.MessageThread, 0, pageSize)
	for rows.Next() {
		m := &biz.MessageThread{}
		var msgType, unread int
		var created, updated time.Time
		if err := rows.Scan(&m.MessageID, &m.Title, &m.Content, &msgType, &unread, &created, &updated); err != nil {
			return nil, err
		}
		m.ID = m.MessageID
		m.Subject = m.Title
		m.Preview = m.Content
		m.UnreadCount = unread
		m.Unread = unread
		m.CreateTime = biz.FmtDT(created)
		m.UpdatedAt = biz.FmtDT(updated)
		m.UpdateTime = m.UpdatedAt
		m.IsOfficial = msgType == 1
		m.Official = m.IsOfficial
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *MessageRepo) UnreadCount(ctx context.Context, userID int64) (int, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return 0, err
	}
	var total int
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(unread_count),0) FROM user_message WHERE user_id = ? AND status = 1`, userID).Scan(&total)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return total, err
}

func (r *MessageRepo) MarkRead(ctx context.Context, userID int64, messageID int64) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`UPDATE user_message SET unread_count = 0, gmt_modified = NOW()
		  WHERE message_id = ? AND user_id = ?`, messageID, userID)
	return err
}

// AddMessage 写入一条系统通知（异步事件消费者调用）。
func (r *MessageRepo) AddMessage(ctx context.Context, userID int64, title, content string) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	// message_id 用纳秒时间戳，避免多实例并发时 MAX()+1 的竞争。
	messageID := time.Now().UnixNano()
	_, err = db.ExecContext(ctx,
		`INSERT INTO user_message (message_id, user_id, title, content, msg_type, status, unread_count, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, 1, 1, 1, NOW(), NOW())`, messageID, userID, title, content)
	return err
}
