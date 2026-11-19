package biz

import (
	"context"
	"errors"
	"testing"
)

// 本文件补齐 OPEN_ITEMS D4 中曾缺少 biz 单测的 6 个 usecase 的边界路径。
// stub 通过嵌入 nil 接口满足完整方法集，只实现被测路径实际调用的方法；
// 未实现方法一旦被调用会 panic，从而暴露「意外触达存储」的回归。

type stubCatalogRepo struct {
	CatalogRepo
	got ActivityQuery
}

func (s *stubCatalogRepo) Activities(_ context.Context, q ActivityQuery, _ bool) ([]*ShowSummary, error) {
	s.got = q
	return []*ShowSummary{}, nil
}

type stubFeedRepo struct {
	FeedRepo
	got FeedQuery
}

func (s *stubFeedRepo) Feeds(_ context.Context, q FeedQuery, _ bool) ([]*FeedPost, bool, int64, error) {
	s.got = q
	return []*FeedPost{}, false, 0, nil
}

type stubMessageRepo struct {
	MessageRepo
	gotPage, gotSize int
}

func (s *stubMessageRepo) Messages(_ context.Context, _ int64, _, page, size int) ([]*MessageThread, error) {
	s.gotPage, s.gotSize = page, size
	return []*MessageThread{}, nil
}

type stubUserRepo struct{ UserRepo }

type stubOrderRepo struct{ OrderRepo }

func (s *stubOrderRepo) Orders(_ context.Context, _ int64, _, _, _, _ int) ([]*TicketOrder, error) {
	return []*TicketOrder{}, nil
}

func TestCatalogUsecaseBoundary(t *testing.T) {
	repo := &stubCatalogRepo{}
	uc := NewCatalogUsecase(repo)
	if _, err := uc.Activities(context.Background(), ActivityQuery{Page: 0, PageSize: 101}, false); err != nil {
		t.Fatalf("Activities: %v", err)
	}
	if repo.got.Page != 1 || repo.got.PageSize != 20 {
		t.Errorf("分页归一 = %+v, want Page=1 PageSize=20", repo.got)
	}
	if _, err := uc.Activity(context.Background(), 0); !errors.Is(err, ErrBadRequest) {
		t.Errorf("Activity(0) = %v, want ErrBadRequest", err)
	}
}

func TestFeedUsecaseBoundary(t *testing.T) {
	repo := &stubFeedRepo{}
	uc := NewFeedUsecase(repo)
	page, err := uc.Feeds(context.Background(), FeedQuery{PageSize: 101}, false)
	if err != nil {
		t.Fatalf("Feeds: %v", err)
	}
	if page == nil || page.Data == nil || page.HasMore {
		t.Errorf("Feeds 应返回非 nil 的空页: %+v", page)
	}
	if repo.got.PageSize != 20 {
		t.Errorf("pageSize 归一 = %d, want 20", repo.got.PageSize)
	}
	if _, err := uc.Feed(context.Background(), 0); !errors.Is(err, ErrBadRequest) {
		t.Errorf("Feed(0) = %v, want ErrBadRequest", err)
	}
}

func TestMessageUsecaseBoundary(t *testing.T) {
	repo := &stubMessageRepo{}
	uc := NewMessageUsecase(repo)
	msgs, err := uc.Messages(context.Background(), 7, 0, 0, 101)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if msgs == nil {
		t.Error("Messages 应返回非 nil 空数组")
	}
	if repo.gotPage != 1 || repo.gotSize != 20 {
		t.Errorf("分页归一 = %d/%d, want 1/20", repo.gotPage, repo.gotSize)
	}
	if err := uc.MarkRead(context.Background(), 7, 0); !errors.Is(err, ErrBadRequest) {
		t.Errorf("MarkRead(0) = %v, want ErrBadRequest", err)
	}
}

func TestUserUsecaseBoundary(t *testing.T) {
	uc := NewUserUsecase(&stubUserRepo{})
	if _, err := uc.AddAddress(context.Background(), 7, &AddressInput{}); !errors.Is(err, ErrBadRequest) {
		t.Errorf("空地址应被拒绝: %v", err)
	}
	if err := uc.DeleteAddress(context.Background(), 7, 0); !errors.Is(err, ErrBadRequest) {
		t.Errorf("DeleteAddress(0) = %v, want ErrBadRequest", err)
	}
	if err := uc.SetDefaultAddress(context.Background(), 7, -1); !errors.Is(err, ErrBadRequest) {
		t.Errorf("SetDefaultAddress(-1) = %v, want ErrBadRequest", err)
	}
}

func TestAuthUsecaseLoginBoundary(t *testing.T) {
	// 构造函数会为 nil 依赖补安全默认值；空手机号在触达存储前即被拒绝。
	uc := NewAuthUsecase(nil, nil, nil, nil, nil, 0)
	if _, _, err := uc.Login(context.Background(), "", "", ""); !errors.Is(err, ErrBadRequest) {
		t.Errorf("Login(空手机号) = %v, want ErrBadRequest", err)
	}
	if _, err := uc.SendLoginCode(context.Background(), ""); !errors.Is(err, ErrBadRequest) {
		t.Errorf("SendLoginCode(空手机号) = %v, want ErrBadRequest", err)
	}
	if _, err := uc.ResolveSession(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("ResolveSession(空 token) = %v, want ErrUnauthorized", err)
	}
}

func TestOrderUsecaseMakeOrderBoundary(t *testing.T) {
	uc := NewOrderUsecase(&stubOrderRepo{}, nil, nil)
	if _, err := uc.MakeOrder(context.Background(), 7, &OrderInput{}); !errors.Is(err, ErrBadRequest) {
		t.Errorf("无 ActivityID/QuoteID 应被拒绝: %v", err)
	}
	if _, err := uc.MakeOrder(context.Background(), 7, &OrderInput{ActivityID: 1, Quantity: MaxQuantityPerOrder + 1}); !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("超限数量应被拒绝: %v", err)
	}
	if _, err := uc.Orders(context.Background(), 7, 0, 0, 0, 101); err != nil {
		t.Errorf("Orders 分页归一不应报错: %v", err)
	}
}
