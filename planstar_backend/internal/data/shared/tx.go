package shared

import (
	"context"
	"database/sql"
)

// WithTx 统一 data 层事务样板（ADR-0010 P2⑧）：Begin → 执行 fn → 失败回滚 / 成功提交。
// fn 返回 error 时回滚并透传，返回 nil 时提交；提交失败原样返回。
//
// fn 不得自行调用 tx.Commit / tx.Rollback；回滚由本函数兜底，避免某个提前 return
// 漏掉回滚、把未结束事务占住连接池。fn 内应只使用传入的 tx，不要混用外层 *sql.DB。
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
