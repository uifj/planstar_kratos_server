package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"planstar/internal/biz"
)

// WalletRepo 实现 biz.WalletRepo，账户/流水落 planstar_payment，签到落 planstar_marketing。
type WalletRepo struct {
	d *Data
}

func NewWalletRepo(d *Data) *WalletRepo { return &WalletRepo{d: d} }

func (r *WalletRepo) EnsureAccount(ctx context.Context, userID int64) (*biz.WalletAccount, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	accountID, balance, err := ensureWalletTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &biz.WalletAccount{
		AccountID: accountID, Balance: balance, Currency: "star_coin", Status: 1,
	}, nil
}

// Balance 只读钱包余额；没有 wallet_account 时回退 user_statistic，不写库。
func (r *WalletRepo) Balance(ctx context.Context, userID int64) (*biz.WalletAccount, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	account := &biz.WalletAccount{AccountID: userID, Currency: "star_coin", Status: 1}
	var balance float64
	var status int32
	err = db.QueryRowContext(ctx,
		`SELECT account_id, COALESCE(balance,0), COALESCE(status,0)
		   FROM wallet_account WHERE user_id = ? LIMIT 1`, userID).
		Scan(&account.AccountID, &balance, &status)
	if err == nil {
		account.Balance = biz.Fen(balance)
		account.Status = status
		return account, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 无钱包账户：按用户统计余额回退（兼容存量数据）。
	userDB, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	err = userDB.QueryRowContext(ctx,
		`SELECT COALESCE(balance,0) FROM user_statistic WHERE user_id = ? LIMIT 1`, userID).
		Scan(&balance)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	account.Balance = biz.Fen(balance)
	return account, nil
}

func (r *WalletRepo) Sequences(ctx context.Context, userID int64, limit int) ([]*biz.WalletTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, COALESCE(title,''), COALESCE(amount,0), COALESCE(balance_after,0),
		        COALESCE(type,''), gmt_create
		   FROM wallet_sequence
		  WHERE user_id = ?
		  ORDER BY gmt_create DESC, id DESC
		  LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.WalletTransaction, 0, limit)
	for rows.Next() {
		t := &biz.WalletTransaction{}
		var amount, balanceAfter float64
		var happen time.Time
		if err := rows.Scan(&t.ID, &t.Title, &amount, &balanceAfter, &t.Type, &happen); err != nil {
			return nil, err
		}
		t.Amount = biz.Fen(amount)
		t.BalanceAfter = biz.Fen(balanceAfter)
		t.HappenTime = happen.Format(time.RFC3339)
		items = append(items, t)
	}
	return items, rows.Err()
}

// SignInStatus 返回指定自然日的签到状态（只读，不写库）。
func (r *WalletRepo) SignInStatus(ctx context.Context, userID int64, date time.Time) (*biz.SignInStatus, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	status := &biz.SignInStatus{TodayReward: biz.SignInRewardCoins}
	var signedAt time.Time
	err = db.QueryRowContext(ctx,
		`SELECT gmt_create FROM planstar_marketing.daily_sign_in
		  WHERE user_id = ? AND sign_date = ? LIMIT 1`, userID, date.Format("2006-01-02")).
		Scan(&signedAt)
	switch {
	case err == nil:
		status.Signed = true
		status.AlreadySigned = true
		status.SignedAt = signedAt.Format(time.RFC3339)
	case err == sql.ErrNoRows:
		// 今日未签到。
	default:
		return nil, err
	}

	account, err := r.Balance(ctx, userID)
	if err != nil {
		return nil, err
	}
	status.Balance = account.Balance
	return status, nil
}

// SignIn 幂等签到：同一自然日仅首次入账并写流水，重复调用返回已有状态。
func (r *WalletRepo) SignIn(ctx context.Context, userID int64, date time.Time, reward int64) (*biz.SignInStatus, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	dateStr := date.Format("2006-01-02")
	res, err := tx.ExecContext(ctx,
		`INSERT INTO planstar_marketing.daily_sign_in (user_id, sign_date, reward_coins)
		 VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE id = id`, userID, dateStr, reward)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	first := affected == 1

	accountID, balance, err := ensureWalletTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	status := &biz.SignInStatus{
		Signed:        true,
		AlreadySigned: !first,
		TodayReward:   reward,
		Balance:       balance,
		SignedAt:      date.Format(time.RFC3339),
	}
	if first {
		balance, err = addBalanceTx(ctx, tx, userID, reward*100)
		if err != nil {
			return nil, err
		}
		if err := writeSequenceTx(ctx, tx, userID, accountID, "每日签到", reward*100, balance, "income"); err != nil {
			return nil, err
		}
		status.Balance = balance
	} else {
		var signedAt time.Time
		if err := tx.QueryRowContext(ctx,
			`SELECT gmt_create FROM planstar_marketing.daily_sign_in
			  WHERE user_id = ? AND sign_date = ? LIMIT 1`, userID, dateStr).Scan(&signedAt); err == nil {
			status.SignedAt = signedAt.Format(time.RFC3339)
		} else if err != sql.ErrNoRows {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return status, nil
}

// ensureWalletTx 在事务内锁定钱包账户；不存在时按 user_statistic.balance 初始化。
// 返回 accountID 与余额（分），供签到与商城兑换复用。
func ensureWalletTx(ctx context.Context, tx *sql.Tx, userID int64) (int64, int64, error) {
	var accountID int64
	var balance float64
	err := tx.QueryRowContext(ctx,
		`SELECT account_id, COALESCE(balance,0)
		   FROM planstar_payment.wallet_account WHERE user_id = ? LIMIT 1 FOR UPDATE`, userID).
		Scan(&accountID, &balance)
	if err == nil {
		return accountID, biz.Fen(balance), nil
	}
	if err != sql.ErrNoRows {
		return 0, 0, err
	}

	// 首次开户：余额继承存量 user_statistic，避免老用户资产归零。
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(balance,0) FROM planstar_user.user_statistic
		  WHERE user_id = ? LIMIT 1`, userID).Scan(&balance)
	if err == sql.ErrNoRows {
		balance, err = 0, nil
	}
	if err != nil {
		return 0, 0, err
	}

	accountID = userID
	_, err = tx.ExecContext(ctx,
		`INSERT INTO planstar_payment.wallet_account
		   (account_id, user_id, balance, status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, 1, NOW(), NOW())`,
		accountID, userID, yuanString(biz.Fen(balance)))
	if isDuplicateKey(err) {
		// 并发开户：重新锁定已存在的账户。
		err = tx.QueryRowContext(ctx,
			`SELECT account_id, COALESCE(balance,0)
			   FROM planstar_payment.wallet_account WHERE user_id = ? LIMIT 1 FOR UPDATE`, userID).
			Scan(&accountID, &balance)
		if err != nil {
			return 0, 0, err
		}
		return accountID, biz.Fen(balance), nil
	}
	if err != nil {
		return 0, 0, err
	}
	return accountID, biz.Fen(balance), nil
}

// addBalanceTx 在已锁定账户的事务内增减余额，返回新余额（分）。delta 为分，可为负。
func addBalanceTx(ctx context.Context, tx *sql.Tx, userID int64, delta int64) (int64, error) {
	if _, err := tx.ExecContext(ctx,
		`UPDATE planstar_payment.wallet_account SET balance = balance + ?, gmt_modified = NOW() WHERE user_id = ?`,
		yuanString(delta), userID); err != nil {
		return 0, err
	}
	var balance float64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(balance,0) FROM planstar_payment.wallet_account WHERE user_id = ? LIMIT 1`, userID).
		Scan(&balance); err != nil {
		return 0, err
	}
	return biz.Fen(balance), nil
}

// writeSequenceTx 写一条钱包流水（金额单位为分，落库为元）。
func writeSequenceTx(ctx context.Context, tx *sql.Tx, userID, accountID int64, title string, amount, balanceAfter int64, typ string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO planstar_payment.wallet_sequence
		   (sequence_id, user_id, account_id, title, amount, balance_after, type, status, gmt_create, gmt_modified)
		 VALUES (UUID_SHORT(), ?, ?, ?, ?, ?, ?, 1, NOW(), NOW())`,
		userID, accountID, title, yuanString(amount), yuanString(balanceAfter), typ)
	return err
}

// yuanString 把「分」格式化为 DECIMAL(12,2) 的元字符串，避免浮点误差。
func yuanString(minor int64) string {
	return fmt.Sprintf("%.2f", float64(minor)/100)
}
