package biz

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// memWalletRepo 是内存版钱包仓储，用于验证签到幂等与余额叠加规则。
// signedAt 以「自然日 → 签到时间」模拟 daily_sign_in 的唯一键语义。
type memWalletRepo struct {
	mu       sync.Mutex
	balance  int64 // 分
	signedAt map[string]string
}

func (m *memWalletRepo) EnsureAccount(_ context.Context, userID int64) (*WalletAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &WalletAccount{AccountID: userID, Balance: m.balance, Currency: "star_coin", Status: 1}, nil
}

func (m *memWalletRepo) Balance(_ context.Context, userID int64) (*WalletAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &WalletAccount{AccountID: userID, Balance: m.balance, Currency: "star_coin", Status: 1}, nil
}

func (m *memWalletRepo) Sequences(_ context.Context, _ int64, _ int) ([]*WalletTransaction, error) {
	return nil, nil
}

func (m *memWalletRepo) SignInStatus(_ context.Context, _ int64, date time.Time) (*SignInStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	at, ok := m.signedAt[signDateKey(date)]
	return &SignInStatus{Signed: ok, AlreadySigned: ok, TodayReward: SignInRewardCoins, Balance: m.balance, SignedAt: at}, nil
}

func (m *memWalletRepo) SignIn(_ context.Context, _ int64, date time.Time, reward int64) (*SignInStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.signedAt == nil {
		m.signedAt = map[string]string{}
	}
	key := signDateKey(date)
	at, ok := m.signedAt[key]
	if !ok {
		at = date.Format(time.RFC3339)
		m.signedAt[key] = at
		m.balance += reward * 100
	}
	return &SignInStatus{
		Signed: true, AlreadySigned: ok, TodayReward: reward,
		Balance: m.balance, SignedAt: at,
	}, nil
}

func signDateKey(date time.Time) string { return date.Format("2006-01-02") }

func TestWalletSignInIdempotent(t *testing.T) {
	repo := &memWalletRepo{}
	uc := NewWalletUsecase(repo)

	first, err := uc.SignIn(context.Background(), 1)
	if err != nil {
		t.Fatalf("first sign in err = %v", err)
	}
	if first.AlreadySigned {
		t.Fatalf("first sign in alreadySigned = true, want false")
	}
	if first.TodayReward != SignInRewardCoins {
		t.Fatalf("todayReward = %d, want %d", first.TodayReward, SignInRewardCoins)
	}
	if first.Balance != SignInRewardCoins*100 {
		t.Fatalf("balance = %d, want %d", first.Balance, SignInRewardCoins*100)
	}
	if first.SignedAt == "" {
		t.Fatalf("first sign in signedAt is empty")
	}

	second, err := uc.SignIn(context.Background(), 1)
	if err != nil {
		t.Fatalf("second sign in err = %v", err)
	}
	if !second.AlreadySigned {
		t.Fatalf("second sign in alreadySigned = false, want true")
	}
	if second.Balance != first.Balance {
		t.Fatalf("balance after repeat = %d, want %d (no duplicate credit)", second.Balance, first.Balance)
	}
}

// TestWalletSignInConcurrent 验证同一自然日内 20 次并发签到只入账一次。
func TestWalletSignInConcurrent(t *testing.T) {
	repo := &memWalletRepo{}
	uc := NewWalletUsecase(repo)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := uc.SignIn(context.Background(), 1); err != nil {
				t.Errorf("concurrent sign in err = %v", err)
			}
		}()
	}
	wg.Wait()

	if repo.balance != SignInRewardCoins*100 {
		t.Fatalf("balance = %d, want %d (only one credit)", repo.balance, SignInRewardCoins*100)
	}
}

func TestWalletSignInRequiresUser(t *testing.T) {
	uc := NewWalletUsecase(&memWalletRepo{})
	if _, err := uc.SignIn(context.Background(), 0); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous sign in err = %v, want ErrUnauthorized", err)
	}
	if _, err := uc.Status(context.Background(), 0); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous status err = %v, want ErrUnauthorized", err)
	}
}
