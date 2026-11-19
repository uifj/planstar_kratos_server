package biz

import (
	"context"
	"errors"
	"sort"
	"testing"
)

// memChatRepo 是内存版聊天仓储，用于验证用例层的分页与未读规则。
type memChatRepo struct {
	owners   map[int64]int64 // threadID -> userID
	threads  map[int64]*ChatThread
	messages map[int64][]*ChatMessage
	lastLim  int
	lastBef  int64
}

func newMemChatRepo() *memChatRepo {
	return &memChatRepo{
		owners:   map[int64]int64{1: 1, 2: 2},
		threads:  map[int64]*ChatThread{1: {ThreadID: 1, Title: "企划星小助手", UnreadCount: 3}},
		messages: map[int64][]*ChatMessage{},
	}
}

func (m *memChatRepo) Threads(_ context.Context, userID int64) ([]*ChatThread, error) {
	items := make([]*ChatThread, 0, len(m.threads))
	for threadID, t := range m.threads {
		if m.owners[threadID] == userID {
			copied := *t
			items = append(items, &copied)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ThreadID < items[j].ThreadID })
	return items, nil
}

func (m *memChatRepo) Messages(_ context.Context, userID, threadID int64, limit int, before int64) ([]*ChatMessage, error) {
	if m.owners[threadID] != userID {
		return nil, ErrNotFound
	}
	m.lastLim, m.lastBef = limit, before
	all := m.messages[threadID]
	filtered := make([]*ChatMessage, 0, len(all))
	for _, msg := range all {
		if before == 0 || msg.ID < before {
			filtered = append(filtered, msg)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered, nil
}

func (m *memChatRepo) SendMessage(_ context.Context, userID, threadID int64, text string) (*ChatMessage, error) {
	if m.owners[threadID] != userID {
		return nil, ErrNotFound
	}
	msg := &ChatMessage{ID: int64(len(m.messages[threadID]) + 1), Text: text, FromMe: true, SentAt: "2026-11-07 12:00:00"}
	m.messages[threadID] = append(m.messages[threadID], msg)
	return msg, nil
}

func (m *memChatRepo) MarkRead(_ context.Context, userID, threadID int64) error {
	if m.owners[threadID] != userID {
		return ErrNotFound
	}
	m.threads[threadID].UnreadCount = 0
	return nil
}

func TestChatMessagesPaging(t *testing.T) {
	repo := newMemChatRepo()
	for i := int64(1); i <= 5; i++ {
		repo.messages[1] = append(repo.messages[1], &ChatMessage{ID: i, Text: "m"})
	}
	uc := NewChatUsecase(repo)

	latest, err := uc.Messages(context.Background(), 1, 1, 2, 0)
	if err != nil {
		t.Fatalf("messages err = %v", err)
	}
	if len(latest) != 2 || latest[0].ID != 4 || latest[1].ID != 5 {
		t.Fatalf("latest = %+v, want ids [4 5]", latest)
	}

	older, err := uc.Messages(context.Background(), 1, 1, 2, 5)
	if err != nil {
		t.Fatalf("messages before err = %v", err)
	}
	if len(older) != 2 || older[0].ID != 3 || older[1].ID != 4 {
		t.Fatalf("older = %+v, want ids [3 4]", older)
	}
	if repo.lastBef != 5 {
		t.Fatalf("before passed = %d, want 5", repo.lastBef)
	}

	if _, err := uc.Messages(context.Background(), 1, 1, 0, 0); err != nil {
		t.Fatalf("default limit err = %v", err)
	}
	if repo.lastLim != chatDefaultLimit {
		t.Fatalf("default limit = %d, want %d", repo.lastLim, chatDefaultLimit)
	}
	if _, err := uc.Messages(context.Background(), 1, 1, 500, 0); err != nil {
		t.Fatalf("max limit err = %v", err)
	}
	if repo.lastLim != chatMaxLimit {
		t.Fatalf("max limit = %d, want %d", repo.lastLim, chatMaxLimit)
	}
	if _, err := uc.Messages(context.Background(), 1, 2, 20, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other user's thread err = %v, want ErrNotFound", err)
	}
}

func TestChatUnreadCountResetsAfterRead(t *testing.T) {
	repo := newMemChatRepo()
	uc := NewChatUsecase(repo)

	threads, err := uc.Threads(context.Background(), 1)
	if err != nil {
		t.Fatalf("threads err = %v", err)
	}
	if len(threads) != 1 || threads[0].UnreadCount != 3 {
		t.Fatalf("threads = %+v, want one thread with unread 3", threads)
	}

	unread, err := uc.MarkRead(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("mark read err = %v", err)
	}
	if unread != 0 {
		t.Fatalf("unread = %d, want 0", unread)
	}
	threads, err = uc.Threads(context.Background(), 1)
	if err != nil {
		t.Fatalf("threads after read err = %v", err)
	}
	if threads[0].UnreadCount != 0 {
		t.Fatalf("unread after read = %d, want 0", threads[0].UnreadCount)
	}
}

func TestChatSendMessageValidatesText(t *testing.T) {
	uc := NewChatUsecase(newMemChatRepo())
	if _, err := uc.SendMessage(context.Background(), 1, 1, "   "); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("empty text err = %v, want ErrBadRequest", err)
	}
	long := make([]rune, maxChatMessageLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := uc.SendMessage(context.Background(), 1, 1, string(long)); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("long text err = %v, want ErrBadRequest", err)
	}
	msg, err := uc.SendMessage(context.Background(), 1, 1, "你好")
	if err != nil || msg.Text != "你好" || !msg.FromMe {
		t.Fatalf("send = %+v, %v; want persisted fromMe message", msg, err)
	}
}
