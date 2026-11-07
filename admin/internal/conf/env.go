package conf

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/encoding"
)

// Environment values arrive as raw strings, but Scan decodes the merged map
// with protojson, which rejects "true" for a bool field or "0" for an int32.
// These path sets tell the decoder which leaf fields need typed values.
var envBools = map[string]bool{
	"data.database.debug":        true,
	"data.database.auto_migrate": true,
	"auth.cookie_secure":         true,
	"notify.sms.enabled":         true,
	"notify.email.enabled":       true,
	"storage.oss.enabled":        true,
}

var envInts = map[string]bool{
	"data.database.max_open_conns":    true,
	"data.database.max_idle_conns":    true,
	"data.redis.db":                   true,
	"auth.login_max_attempts":         true,
	"notify.email.smtp_port":          true,
	"storage.oss.presign_ttl_seconds": true,
}

// envLeaves are config field names that contain underscores. Env sources
// expose ADMIN_DATA_DATABASE_SOURCE as DATA_DATABASE_SOURCE, so the decoder
// keeps these leaf names intact instead of treating every underscore as a
// nesting separator.
var envLeaves = []string{
	"cors_allow_origins",
	"presign_ttl_seconds",
	"conn_max_lifetime",
	"login_max_attempts",
	"max_open_conns",
	"max_idle_conns",
	"access_secret",
	"cookie_domain",
	"cookie_secure",
	"login_window",
	"read_timeout",
	"cookie_name",
	"auto_migrate",
	"ops_base_url",
	"ops_timeout",
	"write_timeout",
	"access_key",
	"jwt_secret",
	"login_lock",
	"sign_name",
	"smtp_host",
	"smtp_port",
	"template_code",
	"token_ttl",
	"cdn_host",
	"ops_token",
}

// DecodeConfig extends the default Kratos decoder so ADMIN_* environment
// variables (Format is empty) land on the same nested paths as config.yaml,
// e.g. ADMIN_DATA_DATABASE_SOURCE overrides data.database.source.
func DecodeConfig(kv *config.KeyValue, target map[string]any) error {
	if kv.Format != "" {
		codec := encoding.GetCodec(kv.Format)
		if codec == nil {
			return fmt.Errorf("unsupported config format %q", kv.Format)
		}
		return codec.Unmarshal(kv.Value, &target)
	}

	keys := envPath(kv.Key)
	for i, k := range keys {
		if i == len(keys)-1 {
			target[k] = envValue(strings.Join(keys, "."), kv.Value)
		} else {
			sub, ok := target[k].(map[string]any)
			if !ok {
				sub = make(map[string]any)
				target[k] = sub
			}
			target = sub
		}
	}
	return nil
}

func envPath(key string) []string {
	key = strings.ToLower(key)
	for _, leaf := range envLeaves {
		if key == leaf {
			return []string{leaf}
		}
		if parent, ok := strings.CutSuffix(key, "_"+leaf); ok {
			return append(strings.Split(parent, "_"), leaf)
		}
	}
	return strings.Split(key, "_")
}

// envValue converts a raw environment value into the JSON type protojson
// expects for the given config path.
func envValue(path string, raw []byte) any {
	s := string(raw)
	switch {
	case envBools[path]:
		if v, err := strconv.ParseBool(strings.TrimSpace(s)); err == nil {
			return v
		}
	case envInts[path]:
		if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			return v
		}
	case path == "server.http.cors_allow_origins":
		return envList(s)
	}
	return s
}

// envList accepts either a JSON array or a comma separated origin list.
func envList(s string) any {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var out []any
		if err := json.Unmarshal([]byte(s), &out); err == nil {
			return out
		}
	}
	out := make([]string, 0, 2)
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
