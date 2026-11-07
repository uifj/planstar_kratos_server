package biz

import (
	"context"
	"strings"
)

// 会话消息分页参数。
const (
	chatDefaultLimit = 20
	chatMaxLimit     = 100
	// maxChatMessageLength 是单条消息的最大字符数。
	maxChatMessageLength = 2000
)

// ChatThread 是会话列表条目（契约 GET /v1/chat/threads）。
type ChatThread struct {
	ThreadID    int64  `json:"threadId"`
	Title       string `json:"title"`
	Preview     string `json:"preview"`
	UpdatedAt   string `json:"updatedAt"`
	UnreadCount int    `json:"unreadCount"`
	AvatarURL   string `json:"avatarUrl"`
	IsOfficial  bool   `json:"isOfficial"`
}

// ChatMessage 是会话消息（契约 GET /v1/chat/threads/{id}/messages）。
type ChatMessage struct {
	ID         int64  `json:"id"`
	Text       string `json:"text"`
	SentAt     string `json:"sentAt"`
	FromMe     bool   `json:"fromMe"`
	SenderName string `json:"senderName"`
}

// ChatRepo 是聊天仓储端口。
type ChatRepo interface {
	Threads(ctx context.Context, userID int64) ([]*ChatThread, error)
	// Messages 按 message_id 倒序取 before 之前（不含）的 limit 条，返回时按时间升序。
	Messages(ctx context.Context, userID, threadID int64, limit int, before int64) ([]*ChatMessage, error)
	SendMessage(ctx context.Context, userID, threadID int64, text string) (*ChatMessage, error)
	MarkRead(ctx context.Context, userID, threadID int64) error
}

// ChatUsecase 聊天用例。
type ChatUsecase struct {
	repo ChatRepo
}

func NewChatUsecase(repo ChatRepo) *ChatUsecase {
	return &ChatUsecase{repo: repo}
}

// Threads 返回当前用户的会话列表。
func (uc *ChatUsecase) Threads(ctx context.Context, userID int64) ([]*ChatThread, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	items, err := uc.repo.Threads(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*ChatThread{}
	}
	return items, nil
}

// Messages 返回会话消息分页；limit 默认 20、上限 100，before 为上一页最小 messageId（0 表示最新）。
func (uc *ChatUsecase) Messages(ctx context.Context, userID, threadID int64, limit int, before int64) ([]*ChatMessage, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if threadID <= 0 {
		return nil, ErrBadRequest
	}
	if limit <= 0 {
		limit = chatDefaultLimit
	}
	if limit > chatMaxLimit {
		limit = chatMaxLimit
	}
	if before < 0 {
		before = 0
	}
	items, err := uc.repo.Messages(ctx, userID, threadID, limit, before)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*ChatMessage{}
	}
	return items, nil
}

// SendMessage 发送消息并返回落库后的消息。
func (uc *ChatUsecase) SendMessage(ctx context.Context, userID, threadID int64, text string) (*ChatMessage, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if threadID <= 0 {
		return nil, ErrBadRequest
	}
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > maxChatMessageLength {
		return nil, ErrBadRequest
	}
	return uc.repo.SendMessage(ctx, userID, threadID, text)
}

// MarkRead 标记会话已读，返回剩余未读数（固定 0）。
func (uc *ChatUsecase) MarkRead(ctx context.Context, userID, threadID int64) (int, error) {
	if userID <= 0 {
		return 0, ErrUnauthorized
	}
	if threadID <= 0 {
		return 0, ErrBadRequest
	}
	if err := uc.repo.MarkRead(ctx, userID, threadID); err != nil {
		return 0, err
	}
	return 0, nil
}
