package data

import (
	"context"
	"database/sql"
	"strings"

	"planstar/internal/biz"
)

// UserRepo 实现 biz.UserRepo，落 planstar_user。
type UserRepo struct {
	d *Data
}

func NewUserRepo(d *Data) *UserRepo { return &UserRepo{d: d} }

func (r *UserRepo) Profile(ctx context.Context, userID int64) (*biz.UserProfile, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	p := &biz.UserProfile{}
	var userName, avatar, mobile, levelDesc string
	var level, wantWatch, follower, following int
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(user_name,''), COALESCE(avatar,''), COALESCE(mobile_no,''), COALESCE(level,0),
		        COALESCE(level_desc,''), COALESCE(want_watch_num,0), COALESCE(follower_num,0), COALESCE(following_num,0)
		   FROM user_info WHERE user_id = ?`, userID).
		Scan(&userName, &avatar, &mobile, &level, &levelDesc, &wantWatch, &follower, &following)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var balance float64
	var couponNum, statWantWatch, statFollower, statFollowing int
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(balance,0), COALESCE(coupon_num,0), COALESCE(want_watch_num,0),
		        COALESCE(follower_num,0), COALESCE(following_num,0)
		   FROM user_statistic WHERE user_id = ?`, userID).
		Scan(&balance, &couponNum, &statWantWatch, &statFollower, &statFollowing)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		wantWatch, follower, following = statWantWatch, statFollower, statFollowing
	}

	// 钱包余额优先读 payment.wallet_account；无账户时沿用 user_statistic.balance。
	payDB, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	var walletBalance float64
	err = payDB.QueryRowContext(ctx,
		`SELECT COALESCE(balance,0) FROM wallet_account WHERE user_id = ? LIMIT 1`, userID).
		Scan(&walletBalance)
	if err == nil {
		balance = walletBalance
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	p.UserID = userID
	p.ID = userID
	p.Nickname = userName
	p.UserName = userName
	p.Avatar = avatar
	p.HeadImg = avatar
	p.Mobile = mobile
	p.Phone = mobile
	p.Level = level
	p.UserLevel = level
	p.LevelLabel = levelDesc
	p.WannaSeeCount = wantWatch
	p.FollowActivityCount = wantWatch
	p.FollowCount = following
	p.FollowUserCount = following
	p.CouponCount = couponNum
	p.WalletBalance = biz.Fen(balance)
	p.Balance = p.WalletBalance
	return p, nil
}

func (r *UserRepo) Addresses(ctx context.Context, userID int64) ([]*biz.ShippingAddress, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT address_id, COALESCE(name,''), COALESCE(phone,''), COALESCE(province,''), COALESCE(city,''),
		        COALESCE(district,''), COALESCE(detail_address,''), COALESCE(is_default,0)
		   FROM user_address WHERE user_id = ? AND status = 1
		ORDER BY is_default DESC, address_id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.ShippingAddress, 0, 8)
	for rows.Next() {
		a := &biz.ShippingAddress{}
		var isDefault int
		if err := rows.Scan(&a.AddressID, &a.Name, &a.Mobile, &a.Province, &a.City, &a.District, &a.Detail, &isDefault); err != nil {
			return nil, err
		}
		a.ID = a.AddressID
		a.Consignee = a.Name
		a.Phone = a.Mobile
		a.Region = strings.TrimSpace(strings.Join([]string{a.Province, a.City, a.District}, " "))
		a.Address = a.Detail
		a.IsDefault = isDefault > 0
		a.Default = a.IsDefault
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *UserRepo) AddAddress(ctx context.Context, userID int64, in *biz.AddressInput) (*biz.ShippingAddress, error) {
	db, err := r.d.DB("user")
	if err != nil {
		return nil, err
	}
	province, city, district := splitRegion(in.Region)
	if in.Province != "" {
		province = in.Province
	}
	if in.City != "" {
		city = in.City
	}
	if in.District != "" {
		district = in.District
	}
	if city == "" && province != "" {
		city = province
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if in.IsDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE user_address SET is_default = 0 WHERE user_id = ?`, userID); err != nil {
			return nil, err
		}
	}
	var addressID int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(address_id),0)+1 FROM user_address`).Scan(&addressID); err != nil {
		return nil, err
	}
	isDefault := 0
	if in.IsDefault {
		isDefault = 1
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO user_address
		   (address_id, user_id, name, phone, province, city, district, detail_address, is_default, status, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, NOW(), NOW())`,
		addressID, userID, in.Name, in.Mobile, province, city, district, in.Detail, isDefault); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &biz.ShippingAddress{
		ID: addressID, AddressID: addressID, Name: in.Name, Consignee: in.Name,
		Mobile: in.Mobile, Phone: in.Mobile,
		Province: province, City: city, District: district,
		Region: strings.TrimSpace(strings.Join([]string{province, city, district}, " ")),
		Detail: in.Detail, Address: in.Detail,
		IsDefault: in.IsDefault, Default: in.IsDefault,
	}, nil
}

func (r *UserRepo) DeleteAddress(ctx context.Context, userID int64, addressID int64) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`UPDATE user_address SET status = 0, gmt_modified = NOW() WHERE user_id = ? AND address_id = ?`,
		userID, addressID)
	return err
}

func (r *UserRepo) SetDefaultAddress(ctx context.Context, userID int64, addressID int64) error {
	db, err := r.d.DB("user")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE user_address SET is_default = 0 WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE user_address SET is_default = 1, gmt_modified = NOW()
		  WHERE user_id = ? AND address_id = ? AND status = 1`, userID, addressID); err != nil {
		return err
	}
	return tx.Commit()
}

// splitRegion 把「省 市 区」拆成三段；空格分隔，多于三段时并入 district。
func splitRegion(region string) (string, string, string) {
	parts := strings.Fields(region)
	switch len(parts) {
	case 0:
		return "", "", ""
	case 1:
		return parts[0], parts[0], ""
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], strings.Join(parts[2:], "")
	}
}
