package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// DefaultMaxBodyBytes 是请求体默认上限（1 MiB），upload 类路径由 HTTP 中间件单独放宽。
const DefaultMaxBodyBytes int64 = 1 << 20

// ErrBodyTooLarge 表示请求体超过 MaxBytesReader 限制，调用方可据此返回 413。
var ErrBodyTooLarge = errors.New("request body too large")

// JSON 输出裸 JSON（契约要求无 {code,message,data} 信封）。
func JSON(w http.ResponseWriter, status int, v any) {
	if v == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("{}"))
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"INTERNAL","message":"服务异常","traceId":""}`))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// Plain 输出 text/plain（保留以兼容历史调用方）。
func Plain(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, msg)
}

// errorBody 是统一错误响应体：{code, message, traceId}。
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId"`
}

// Error 输出统一 JSON 错误体，并回写 X-Trace-Id 响应头。
func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	id := TraceID(r)
	w.Header().Set("X-Trace-Id", id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, err := json.Marshal(errorBody{Code: code, Message: message, TraceID: id})
	if err != nil {
		_, _ = io.WriteString(w, `{"code":"INTERNAL","message":"服务异常","traceId":""}`)
		return
	}
	_, _ = w.Write(b)
}

// traceIDKey 用于在请求 context 中缓存已生成的 traceId，保证响应头与响应体一致。
type traceIDKey struct{}

// WithTraceID 把请求级 traceId 写入 context，供后续 Error 复用。
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

// TraceID 返回请求 traceId：优先上下文缓存，其次 OpenTelemetry span，最后生成短随机串。
func TraceID(r *http.Request) string {
	if r == nil {
		return randomTraceID()
	}
	if id, ok := r.Context().Value(traceIDKey{}).(string); ok && id != "" {
		return id
	}
	if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
		return sc.TraceID().String()
	}
	return randomTraceID()
}

// randomTraceID 生成 16 位十六进制短随机串。
func randomTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ReadJSON 解析请求体（默认上限 1 MiB）。
func ReadJSON(r *http.Request, dst any) error {
	return ReadJSONLimit(r, dst, DefaultMaxBodyBytes)
}

// ReadJSONLimit 按 maxBytes 解析请求体；超限返回 ErrBodyTooLarge，maxBytes<=0 时使用默认值。
func ReadJSONLimit(r *http.Request, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBytes))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return normalizeReadError(err)
	}
	return nil
}

// normalizeReadError 把 MaxBytesReader 的错误归一为 ErrBodyTooLarge，便于上层映射 413。
func normalizeReadError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return ErrBodyTooLarge
	}
	return err
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
