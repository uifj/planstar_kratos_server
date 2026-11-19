package conf

import (
	"reflect"
	"testing"

	"github.com/go-kratos/kratos/v3/config"
)

func TestEnvPath(t *testing.T) {
	tests := map[string][]string{
		"DATA_DATABASE_SOURCE":            {"data", "database", "source"},
		"AUTH_JWT_SECRET":                 {"auth", "jwt_secret"},
		"AUTH_LOGIN_MAX_ATTEMPTS":         {"auth", "login_max_attempts"},
		"PLANSTAR_OPS_BASE_URL":           {"planstar", "ops_base_url"},
		"NOTIFY_SMS_ACCESS_KEY":           {"notify", "sms", "access_key"},
		"STORAGE_OSS_PRESIGN_TTL_SECONDS": {"storage", "oss", "presign_ttl_seconds"},
		"SERVER_HTTP_CORS_ALLOW_ORIGINS":  {"server", "http", "cors_allow_origins"},
		"DATA_DATABASE_CONN_MAX_LIFETIME": {"data", "database", "conn_max_lifetime"},
		"SERVER_ADMIN_ADDR":               {"server", "admin", "addr"},
	}
	for key, want := range tests {
		if got := envPath(key); !reflect.DeepEqual(got, want) {
			t.Errorf("envPath(%q) = %v, want %v", key, got, want)
		}
	}
}

func TestDecodeConfigEnv(t *testing.T) {
	target := map[string]any{}
	kv := &config.KeyValue{Key: "DATA_DATABASE_SOURCE", Value: []byte("dsn")}
	if err := DecodeConfig(kv, target); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	db, ok := target["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is %T, want map[string]any", target["data"])
	}
	database, ok := db["database"].(map[string]any)
	if !ok {
		t.Fatalf("data.database is %T, want map[string]any", db["database"])
	}
	if got := database["source"].(string); got != "dsn" {
		t.Fatalf("data.database.source = %q, want %q", got, "dsn")
	}
}

func TestDecodeConfigEnvTypes(t *testing.T) {
	target := map[string]any{}
	kvs := []*config.KeyValue{
		{Key: "AUTH_COOKIE_SECURE", Value: []byte("true")},
		{Key: "DATA_REDIS_DB", Value: []byte("2")},
		{Key: "STORAGE_OSS_PRESIGN_TTL_SECONDS", Value: []byte("600")},
		{Key: "SERVER_HTTP_CORS_ALLOW_ORIGINS", Value: []byte("https://admin.planstar.top")},
	}
	for _, kv := range kvs {
		if err := DecodeConfig(kv, target); err != nil {
			t.Fatalf("DecodeConfig(%s) error = %v", kv.Key, err)
		}
	}
	auth := target["auth"].(map[string]any)
	if got, ok := auth["cookie_secure"].(bool); !ok || !got {
		t.Fatalf("auth.cookie_secure = %#v, want bool true", auth["cookie_secure"])
	}
	redis := target["data"].(map[string]any)["redis"].(map[string]any)
	if got, ok := redis["db"].(int64); !ok || got != 2 {
		t.Fatalf("data.redis.db = %#v, want int64 2", redis["db"])
	}
	oss := target["storage"].(map[string]any)["oss"].(map[string]any)
	if got, ok := oss["presign_ttl_seconds"].(int64); !ok || got != 600 {
		t.Fatalf("storage.oss.presign_ttl_seconds = %#v, want int64 600", oss["presign_ttl_seconds"])
	}
	http := target["server"].(map[string]any)["http"].(map[string]any)
	if got, ok := http["cors_allow_origins"].([]string); !ok || len(got) != 1 || got[0] != "https://admin.planstar.top" {
		t.Fatalf("server.http.cors_allow_origins = %#v", http["cors_allow_origins"])
	}
}
