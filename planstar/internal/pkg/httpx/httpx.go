package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// JSON 输出裸 JSON（契约要求无 {code,message,data} 信封）。
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		_, _ = w.Write([]byte("{}"))
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(b)
}

// Plain 输出 text/plain（登录失败等历史风格）。
func Plain(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, msg)
}

// ReadJSON 解析请求体。
func ReadJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

// ReadMap 解析为通用 map，方便客户端透传体。
func ReadMap(r *http.Request) (map[string]any, error) {
	m := map[string]any{}
	if err := ReadJSON(r, &m); err != nil {
		if errors.Is(err, io.EOF) {
			return m, nil
		}
		return nil, err
	}
	return m, nil
}

// IntQuery 读取整型 query 参数。
func IntQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// Int64Query 读取 int64 query 参数。
func Int64Query(r *http.Request, key string, def int64) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// Str 从 map 取字符串。
func Str(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// StrAny 依次尝试多个键，返回首个非空字符串。
func StrAny(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := Str(m, k); s != "" {
			return s
		}
	}
	return ""
}

// Int64Any 依次尝试多个键，返回首个可解析为整数的值（支持分/元小数按分取整）。
func Int64Any(m map[string]any, keys ...string) int64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			n, err := strconv.ParseInt(t, 10, 64)
			if err == nil {
				return n
			}
		case json.Number:
			n, err := t.Int64()
			if err == nil {
				return n
			}
		case float64:
			return int64(t)
		}
	}
	return 0
}

// BoolAny 依次尝试多个键。
func BoolAny(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		switch t := m[k].(type) {
		case bool:
			return t
		case string:
			return t == "true" || t == "1"
		case json.Number:
			n, _ := t.Int64()
			return n == 1
		case float64:
			return t == 1
		}
	}
	return false
}
