package shared

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

var errNotFound = errors.New("not found")

func TestMapQueryError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		fb   error
		want error
	}{
		{"nil", nil, errNotFound, nil},
		{"no rows", sql.ErrNoRows, errNotFound, errNotFound},
		{"no rows 且无 fallback", sql.ErrNoRows, nil, nil},
		{"包装过的 no rows", errors.Join(sql.ErrNoRows), errNotFound, errNotFound},
		{"其它错误透传", errors.New("boom"), errNotFound, errors.New("boom")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapQueryError(tc.in, tc.fb)
			if errors.Is(tc.want, errNotFound) {
				if !errors.Is(got, errNotFound) {
					t.Fatalf("got %v, want notFound", got)
				}
				return
			}
			if tc.in == nil && got != nil {
				t.Fatalf("got %v, want nil", got)
			}
			if tc.in != nil && !errors.Is(tc.in, sql.ErrNoRows) && got == nil {
				t.Fatal("非 no-rows 错误不应被吞掉")
			}
		})
	}
}

func TestIsDuplicateKey(t *testing.T) {
	if !IsDuplicateKey(&mysql.MySQLError{Number: 1062, Message: "duplicate"}) {
		t.Error("1062 应识别为重复键")
	}
	if IsDuplicateKey(&mysql.MySQLError{Number: 1045}) || IsDuplicateKey(errors.New("boom")) || IsDuplicateKey(nil) {
		t.Error("非 1062 不应识别为重复键")
	}
}

func TestMapRedisError(t *testing.T) {
	if MapRedisError(nil) != nil || MapRedisError(redis.Nil) != nil {
		t.Error("nil / redis.Nil 应视为未命中")
	}
	boom := errors.New("redis down")
	if !errors.Is(MapRedisError(boom), boom) {
		t.Error("其它 Redis 错误应透传")
	}
}

func TestMapContextError(t *testing.T) {
	boom := errors.New("db slow")
	if got := MapContextError(context.Background(), boom); !errors.Is(got, boom) {
		t.Errorf("正常 ctx 应透传: %v", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := MapContextError(ctx, boom); !errors.Is(got, context.Canceled) {
		t.Errorf("已取消 ctx 应包装 context.Canceled: %v", got)
	}
	if MapContextError(context.Background(), nil) != nil {
		t.Error("nil 错误应返回 nil")
	}
}

func TestIsNoRows(t *testing.T) {
	if !IsNoRows(sql.ErrNoRows) || IsNoRows(errors.New("x")) {
		t.Error("IsNoRows 语义错误")
	}
}
