package biz

import (
	"context"
	"time"
)

// SignInRewardCoins 是每日签到的星币奖励（1 星币 = 100 分）。
const SignInRewardCoins int64 = 100

// WalletAccount 是钱包账户（余额单位为分）。
type WalletAccount struct {
	AccountID int64  `json:"accountId"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
	Status    int32  `json:"status"`
}

// WalletTransaction 是钱包流水（金额单位为分，amount 有符号）。
type WalletTransaction struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Amount       int64  `json:"amount"`
	BalanceAfter int64  `json:"balanceAfter"`
	Type         string `json:"type"`
	HappenTime   string `json:"happenTime"`
}

// SignInStatus 是签到结果与当日状态（奖励与余额单位为分）。
type SignInStatus struct {
	Signed        bool   `json:"signed"`
	AlreadySigned bool   `json:"alreadySigned"`
	TodayReward   int64  `json:"todayReward"`
	Balance       int64  `json:"balance"`
	SignedAt      string `json:"signedAt,omitempty"`
}

// WalletRepo 是钱包与签到仓储端口。
type WalletRepo interface {
	// EnsureAccount 确保钱包账户存在（不存在时按 user_statistic 初始化）。
	EnsureAccount(ctx context.Context, userID int64) (*WalletAccount, error)
	// Balance 只读查询余额；无账户时回退 user_statistic，不写库。
	Balance(ctx context.Context, userID int64) (*WalletAccount, error)
	// Sequences 返回最近的钱包流水。
	Sequences(ctx context.Context, userID int64, limit int) ([]*WalletTransaction, error)
	// SignInStatus 返回指定自然日的签到状态（只读）。
	SignInStatus(ctx context.Context, userID int64, date time.Time) (*SignInStatus, error)
	// SignIn 幂等签到：同一自然日仅首次入账。
	SignIn(ctx context.Context, userID int64, date time.Time, reward int64) (*SignInStatus, error)
}

// WalletUsecase 钱包与签到用例。
type WalletUsecase struct {
	repo WalletRepo
}

func NewWalletUsecase(repo WalletRepo) *WalletUsecase {
	return &WalletUsecase{repo: repo}
}

func (uc *WalletUsecase) Account(ctx context.Context, userID int64) (*WalletAccount, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	return uc.repo.EnsureAccount(ctx, userID)
}

func (uc *WalletUsecase) Balance(ctx context.Context, userID int64) (*WalletAccount, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	return uc.repo.Balance(ctx, userID)
}

func (uc *WalletUsecase) Sequences(ctx context.Context, userID int64, limit int) ([]*WalletTransaction, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	items, err := uc.repo.Sequences(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*WalletTransaction{}
	}
	return items, nil
}

// Status 返回当日签到状态；GET /v1/user/signIn 使用，不产生任何写入。
func (uc *WalletUsecase) Status(ctx context.Context, userID int64) (*SignInStatus, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	status, err := uc.repo.SignInStatus(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}
	if status == nil {
		status = &SignInStatus{TodayReward: SignInRewardCoins}
	}
	return status, nil
}

// SignIn 执行每日签到；同一自然日重复调用返回 alreadySigned，不重复入账。
func (uc *WalletUsecase) SignIn(ctx context.Context, userID int64) (*SignInStatus, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	return uc.repo.SignIn(ctx, userID, time.Now(), SignInRewardCoins)
}
