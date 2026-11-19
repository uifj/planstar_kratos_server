//go:build integration

package data

import (
	"context"
	"testing"

	"planstar/internal/testutil"
)

// TestRuntimeSchemaSmoke 在真实 MySQL 上校验建库链路的产物（OPEN_ITEMS D3）。
// 只读 information_schema，不写入、不需要 seed，也无需 Cleanup。
// 未设置/不可达 PLANSTAR_TEST_MYSQL_DSN 时由 testutil 自动 skip。
func TestRuntimeSchemaSmoke(t *testing.T) {
	db := testutil.MySQL(t)
	ctx := context.Background()

	rows, err := db.QueryContext(ctx,
		"SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE 'planstar_%' ORDER BY SCHEMA_NAME")
	if err != nil {
		t.Fatalf("查询数据库清单失败: %v", err)
	}
	defer rows.Close()
	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		schemas = append(schemas, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	// 9 个基线库必须齐全（10/11 号增量库由部署脚本追加，允许本地环境只有基线）。
	baseline := map[string]bool{
		"planstar_user": true, "planstar_catalog": true, "planstar_ticket": true,
		"planstar_order": true, "planstar_payment": true, "planstar_marketing": true,
		"planstar_community": true, "planstar_content": true, "planstar_meta": true,
	}
	got := map[string]bool{}
	for _, s := range schemas {
		got[s] = true
	}
	for name := range baseline {
		if !got[name] {
			t.Errorf("缺少基线库 %s（实际 %v）", name, schemas)
		}
	}

	// 关键新域表必须已从 00_all.sql / 增量脚本建出。
	for _, table := range []struct{ schema, name string }{
		{"planstar_catalog", "project"},
		{"planstar_catalog", "reward_tier"},
		{"planstar_order", "pledge"},
		{"planstar_order", "order_idempotency"},
		{"planstar_payment", "pay_order"},
	} {
		var n int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?",
			table.schema, table.name).Scan(&n)
		if err != nil {
			t.Fatalf("查询 %s.%s 失败: %v", table.schema, table.name, err)
		}
		if n != 1 {
			t.Errorf("缺少表 %s.%s", table.schema, table.name)
		}
	}
}
