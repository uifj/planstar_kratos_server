package biz

import "context"

// MessageRepo 是消息仓储端口。
type MessageRepo interface {
	Messages(ctx context.Context, userID int64, msgType, pageIndex, pageSize int) ([]*MessageThread, error)
	UnreadCount(ctx context.Context, userID int64) (int, error)
	MarkRead(ctx context.Context, userID int64, messageID int64) error
	AddMessage(ctx context.Context, userID int64, title, content string) error
}

// MessageUsecase 消息用例。
type MessageUsecase struct {
	repo MessageRepo
}

func NewMessageUsecase(repo MessageRepo) *MessageUsecase {
	return &MessageUsecase{repo: repo}
}

func (uc *MessageUsecase) Messages(ctx context.Context, userID int64, msgType, pageIndex, pageSize int) ([]*MessageThread, error) {
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	items, err := uc.repo.Messages(ctx, userID, msgType, pageIndex, pageSize)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*MessageThread{}
	}
	return items, nil
}

func (uc *MessageUsecase) UnreadCount(ctx context.Context, userID int64) (int, error) {
	return uc.repo.UnreadCount(ctx, userID)
}

func (uc *MessageUsecase) MarkRead(ctx context.Context, userID int64, messageID int64) error {
	if messageID <= 0 {
		return ErrBadRequest
	}
	return uc.repo.MarkRead(ctx, userID, messageID)
}

// Notify 写入一条官方通知（异步事件消费者调用）。
func (uc *MessageUsecase) Notify(ctx context.Context, userID int64, title, content string) error {
	if userID <= 0 {
		return ErrBadRequest
	}
	return uc.repo.AddMessage(ctx, userID, title, content)
}
