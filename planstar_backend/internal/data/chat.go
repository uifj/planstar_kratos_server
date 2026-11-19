package data

import (
	"context"
	"database/sql"
	"time"

	"planstar/internal/biz"
)

// ChatRepo 实现 biz.ChatRepo，落 planstar_user.chat_thread / chat_message。
type ChatRepo struct {
	d *Data
}

func NewChatRepo(d *Data) *ChatRepo { return &ChatRepo{d: d} }

func (r *ChatRepo) Threads(ctx context.Context, userID int64) ([]*biz.ChatThread, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT thread_id, COALESCE(title,''), COALESCE(preview,''), COALESCE(unread_count,0),
		        COALESCE(avatar_url,''), COALESCE(is_official,0), gmt_modified
		   FROM chat_thread WHERE user_id = ?
		  ORDER BY gmt_modified DESC, thread_id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ChatThread, 0, 8)
	for rows.Next() {
		t := &biz.ChatThread{}
		var official int
		var updated time.Time
		if err := rows.Scan(&t.ThreadID, &t.Title, &t.Preview, &t.UnreadCount,
			&t.AvatarURL, &official, &updated); err != nil {
			return nil, err
		}
		t.IsOfficial = official == 1
		t.UpdatedAt = biz.FmtDT(updated)
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *ChatRepo) Messages(ctx context.Context, userID, threadID int64, limit int, before int64) ([]*biz.ChatMessage, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	if err := ensureThreadOwner(ctx, db, userID, threadID); err != nil {
		return nil, err
	}

	query := `SELECT message_id, COALESCE(text,''), COALESCE(is_from_me,0), COALESCE(sender_name,''), gmt_create
	            FROM chat_message WHERE thread_id = ?`
	args := []any{threadID}
	if before > 0 {
		query += " AND message_id < ?"
		args = append(args, before)
	}
	query += " ORDER BY message_id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ChatMessage, 0, limit)
	for rows.Next() {
		m := &biz.ChatMessage{}
		var fromMe int
		var created time.Time
		if err := rows.Scan(&m.ID, &m.Text, &fromMe, &m.SenderName, &created); err != nil {
			return nil, err
		}
		m.FromMe = fromMe == 1
		m.SentAt = biz.FmtDT(created)
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 查询为倒序，返回前翻转为时间升序，便于客户端直接追加。
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}

func (r *ChatRepo) SendMessage(ctx context.Context, userID, threadID int64, text string) (*biz.ChatMessage, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	err = tx.QueryRowContext(ctx,
		`SELECT 1 FROM chat_thread WHERE thread_id = ? AND user_id = ? LIMIT 1 FOR UPDATE`,
		threadID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	sender := "我"
	_ = tx.QueryRowContext(ctx,
		`SELECT COALESCE(NULLIF(user_name,''),'我') FROM user_info WHERE user_id = ? LIMIT 1`,
		userID).Scan(&sender)

	now := time.Now()
	messageID := now.UnixNano()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO chat_message (message_id, thread_id, text, is_from_me, sender_name, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, 1, ?, NOW(), NOW())`, messageID, threadID, text, sender); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE chat_thread SET preview = ?, gmt_modified = NOW()
		  WHERE thread_id = ? AND user_id = ?`, text, threadID, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &biz.ChatMessage{
		ID: messageID, Text: text, SentAt: biz.FmtDT(now), FromMe: true, SenderName: sender,
	}, nil
}

func (r *ChatRepo) MarkRead(ctx context.Context, userID, threadID int64) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	if err := ensureThreadOwner(ctx, db, userID, threadID); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`UPDATE chat_thread SET unread_count = 0 WHERE thread_id = ? AND user_id = ?`,
		threadID, userID)
	return err
}

// ensureThreadOwner 校验会话归属；不存在或非本人统一返回 ErrNotFound。
func ensureThreadOwner(ctx context.Context, q rowQuerier, userID, threadID int64) error {
	var exists int
	err := q.QueryRowContext(ctx,
		`SELECT 1 FROM chat_thread WHERE thread_id = ? AND user_id = ? LIMIT 1`,
		threadID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return biz.ErrNotFound
	}
	return err
}
