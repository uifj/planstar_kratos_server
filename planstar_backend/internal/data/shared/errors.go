// Package shared 是 data 层的公共内核：驱动错误 → 领域错误的唯一解释点（ADR-0005）。
// 业务域 repo 不再各自记忆 sql.ErrNoRows / MySQL 1062 / redis.Nil 的处理方式。
package shared

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// MapQueryError 把「没有行」映射为调用方指定的领域错误；其它错误原样透传。
// notFound 传 nil 时返回 nil（用于「查不到也算成功」的读面）。
func MapQueryError(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	return err
}

// IsDuplicateKey 识别 MySQL 1062（唯一键冲突）。
// 注意：1062 在多数场景是「幂等命中」而非错误（重试单号、签到、投票），
// 本函数只负责识别，不擅自转成领域错误。
func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// MapRedisError 把 redis.Nil 视为「未命中」（返回 nil），其它错误原样透传。
func MapRedisError(err error) error {
	if err == nil || errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

// MapContextError 把 context 取消/超时归一为可识别的包装错误；
// 其它错误原样透传。HTTP 中间件会统一把 DeadlineExceeded 映射为 504 TIMEOUT。
func MapContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%w: %v", ctxErr, err)
	}
	return err
}

// IsNoRows 是 sql.ErrNoRows 的语义别名，便于 repo 里表达「读面缺省」。
func IsNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
