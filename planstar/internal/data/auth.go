package data

import (
	"context"
	"database/sql"
	"time"

	"planstar/internal/biz"
	"planstar/internal/pkg/secret"
)

// AuthRepo 实现 biz.AuthRepo，落 planstar_user。
type AuthRepo struct {
	d *Data
}

func NewAuthRepo(d *Data) *AuthRepo { return &AuthRepo{d: d} }

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
	hash, err := secret.HashPassword("123456")
	if err != nil {
		return nil, err
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

// SaveLoginCode 保存短信验证码。
func (r *AuthRepo) SaveLoginCode(ctx context.Context, mobile, code string, expireAt time.Time) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO login_code (mobile_no, code, expire_time, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, NOW(), NOW())`, mobile, code, expireAt)
	return err
}

// VerifyLoginCode 校验并一次性消费短信验证码。
func (r *AuthRepo) VerifyLoginCode(ctx context.Context, mobile, code string) bool {
	db, err := r.d.DB("user")
	if err != nil {
		return false
	}
	var id int64
	var expire sql.NullTime
	err = db.QueryRowContext(ctx,
		`SELECT id, expire_time FROM login_code WHERE mobile_no = ? AND code = ? ORDER BY id DESC LIMIT 1`,
		mobile, code).Scan(&id, &expire)
	if err != nil {
		return false
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM login_code WHERE id = ?`, id); err != nil {
		return false
	}
	if expire.Valid && time.Now().After(expire.Time) {
		return false
	}
	return true
}
