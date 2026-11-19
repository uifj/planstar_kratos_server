package data

import (
	"testing"

	"planstar/internal/conf"
)

// 低配档位（config.lite.yaml）中 elastic.addr 为空，不应创建 ES 客户端。
func TestNewElasticEmptyAddrSkipsClient(t *testing.T) {
	client, err := newElastic(&conf.Data{})
	if err != nil {
		t.Fatalf("newElastic with empty addr returned error: %v", err)
	}
	if client != nil {
		t.Fatalf("newElastic with empty addr returned non-nil client: %v", client)
	}
}
