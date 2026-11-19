package data

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"planstar/internal/biz"
	"planstar/internal/pkg/secret"
)

// AuthRepo 实现 biz.AuthRepo，落 planstar_user。
//
// testCredentialPassword 非空时表示当前为本地版本，允许给新注册用户下发测试口令；
// 生产版本该字段恒为空，新用户 password_hash 为空串，密码登录分支必然失败。
// codePepper 用于把短信验证码以 HMAC 摘要入库，避免数据库泄露直接暴露可用验证码。
type AuthRepo struct {
	d                      *Data
	testCredentialPassword string
	codePepper             string
}

// NewAuthRepo 构造生产（安全默认）仓储：不启用测试口令。
func NewAuthRepo(d *Data) *AuthRepo { return &AuthRepo{d: d} }

// NewAuthRepoForEnvironment 按环境构造仓储；仅本地版本传入非空测试口令与验证码 pepper。
func NewAuthRepoForEnvironment(d *Data, testCredentialPassword, codePepper string) *AuthRepo {
	return &AuthRepo{d: d, testCredentialPassword: testCredentialPassword, codePepper: codePepper}
}

// codeDigest 生成确定性的验证码摘要（HMAC-SHA256 截断 64 bit）。
// 截断为 16 个 hex 字符是为了适配 login_code.code VARCHAR(16)（见 15_auth_support.sql）；
// HMAC 截断到 64 bit 仍具备足够的不可伪造性，且远优于原来的明文验证码。
func (r *AuthRepo) codeDigest(mobile, code string) string {
	m := hmac.New(sha256.New, []byte(r.codePepper))
	_, _ = m.Write([]byte(mobile + ":" + code))
	return hex.EncodeToString(m.Sum(nil)[:8])
}

// AssertNoTestCredentials 是生产启动断言：扫描启用状态用户的口令哈希，
// 命中已知测试口令即终止启动，避免种子账号或弱口令被带进生产。
func (r *AuthRepo) AssertNoTestCredentials(ctx context.Context, candidates []string) error {
	if len(candidates) == 0 {
		return nil
	}
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT user_id, password_hash FROM user_info WHERE status = 1 AND COALESCE(password_hash,'') <> ''`)
	if err != nil {
		return fmt.Errorf("check test credentials: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var userID int64
		var hash string
		if err := rows.Scan(&userID, &hash); err != nil {
			return fmt.Errorf("check test credentials: %w", err)
		}
		for _, candidate := range candidates {
			if candidate != "" && secret.VerifyPassword(candidate, hash) {
				return fmt.Errorf("检测到生产环境存在测试/弱口令账号 user_id=%d，请先执行 21_disable_test_accounts.sql 并重置口令", userID)
			}
		}
	}
	return rows.Err()
}

const userCols = `id, user_id, COALESCE(user_name,''), COALESCE(avatar,''), COALESCE(badge_icon,''), COALESCE(bio,''),
	                 birthday, COALESCE(gender,0), COALESCE(mobile_no,''), COALESCE(password_hash,''),
	                 COALESCE(level,0), COALESCE(level_desc,''), COALESCE(follower_num,0), COALESCE(following_num,0),
	                 COALESCE(want_watch_num,0), COALESCE(is_douli_user,0), COALESCE(status,0)`

func scanUser(row *sql.Row) (*biz.UserAccount, error) {
	u := &biz.UserAccount{}
	var birthday sql.NullTime
	err := row.Scan(&u.UserID, &u.UserID, &u.UserName, &u.Avatar, &u.BadgeIcon, &u.Bio, &birthday, &u.Gender,
		&u.MobileNo, &u.PasswordHash, &u.Level, &u.LevelDesc, &u.FollowerNum, &u.FollowingNum,
		&u.WantWatchNum, &u.IsDouliUser, &u.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if birthday.Valid {
		u.Birthday = &birthday.Time
	}
	return u, nil
}

func (r *AuthRepo) FindUserByMobile(ctx context.Context, mobile string) (*biz.UserAccount, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	return scanUser(db.QueryRowContext(ctx, `SELECT `+userCols+` FROM user_info WHERE mobile_no = ?`, mobile))
}

func (r *AuthRepo) FindUserByID(ctx context.Context, userID int64) (*biz.UserAccount, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	return scanUser(db.QueryRowContext(ctx, `SELECT `+userCols+` FROM user_info WHERE user_id = ?`, userID))
}

// CreateUser 用手机号创建用户（短信验证码登录的新用户）。
func (r *AuthRepo) CreateUser(ctx context.Context, mobile string) (*biz.UserAccount, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	var userID int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(user_id),0)+1 FROM user_info`).Scan(&userID); err != nil {
		return nil, err
	}
	// 生产版本：新用户不设置口令，密码登录分支校验空哈希必然失败；
	// 仅本地版本（显式开启测试口令）才下发可登录的测试口令。
	hash := ""
	if r.testCredentialPassword != "" {
		hash, err = secret.HashPassword(r.testCredentialPassword)
		if err != nil {
			return nil, err
		}
	}
	name := "用户" + mobile
	if len(mobile) > 4 {
		name = "用户" + mobile[len(mobile)-4:]
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO user_info (user_id, user_name, mobile_no, password_hash, status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, 1, NOW(), NOW())`, userID, name, mobile, hash)
	if err != nil {
		return nil, err
	}
	return &biz.UserAccount{
		UserID: userID, UserName: name, MobileNo: mobile, PasswordHash: hash, Status: 1,
	}, nil
}

// CreateSession 写入登录会话（access_token 即 psSession）。
func (r *AuthRepo) CreateSession(ctx context.Context, userID int64, loginType int, token string, expireAt time.Time, newUser bool) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	var sessionID int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(session_id),0)+1 FROM login_session`).Scan(&sessionID); err != nil {
		return err
	}
	newFlag := 0
	if newUser {
		newFlag = 1
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO login_session (session_id, user_id, login_type, access_token, is_new_user, expire_time, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		sessionID, userID, loginType, token, newFlag, expireAt)
	return err
}

// UserIDByToken 用 psSession 查用户，过期返回 0。
func (r *AuthRepo) UserIDByToken(ctx context.Context, token string) (int64, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return 0, err
	}
	var userID int64
	var expire sql.NullTime
	err = db.QueryRowContext(ctx,
		`SELECT user_id, expire_time FROM login_session WHERE access_token = ? LIMIT 1`, token).
		Scan(&userID, &expire)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if expire.Valid && time.Now().After(expire.Time) {
		return 0, nil
	}
	return userID, nil
}

// SaveLoginCode 只保存验证码的 HMAC 摘要，数据库泄露不会直接给出可用验证码。
func (r *AuthRepo) SaveLoginCode(ctx context.Context, mobile, code string, expireAt time.Time) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	// 同一手机号旧码立即作废，避免多码并存扩大爆破面。
	if _, err := db.ExecContext(ctx, `DELETE FROM login_code WHERE mobile_no = ?`, mobile); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO login_code (mobile_no, code, expire_time, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, NOW(), NOW())`, mobile, r.codeDigest(mobile, code), expireAt)
	return err
}

// VerifyLoginCode 用单条原子 DELETE 消费验证码：只有恰好删除 1 行才校验通过，
// 消除「SELECT 后 DELETE」的并发重放窗口，也不再静默吞掉数据库错误。
func (r *AuthRepo) VerifyLoginCode(ctx context.Context, mobile, code string) bool {
	db, err := r.d.DB("user")
	if err != nil {
		return false
	}
	res, err := db.ExecContext(ctx,
		`DELETE FROM login_code WHERE mobile_no = ? AND code = ? AND expire_time > NOW()`,
		mobile, r.codeDigest(mobile, code))
	if err != nil {
		return false
	}
	affected, err := res.RowsAffected()
	return err == nil && affected == 1
}
