package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"planstar/internal/conf"
)

type fakeBucket struct {
	putKey      string
	putBody     string
	putOptions  int
	deleteKey   string
	signKey     string
	signMethod  oss.HTTPMethod
	signTTL     int64
	signResult  string
	putErr      error
	deleteErr   error
	signErr     error
	signCallCnt int
}

func (f *fakeBucket) PutObject(objectKey string, reader io.Reader, options ...oss.Option) error {
	f.putKey = objectKey
	f.putOptions = len(options)
	body, _ := io.ReadAll(reader)
	f.putBody = string(body)
	return f.putErr
}

func (f *fakeBucket) DeleteObject(objectKey string, _ ...oss.Option) error {
	f.deleteKey = objectKey
	return f.deleteErr
}

func (f *fakeBucket) SignURL(objectKey string, method oss.HTTPMethod, expiredInSec int64, _ ...oss.Option) (string, error) {
	f.signCallCnt++
	f.signKey = objectKey
	f.signMethod = method
	f.signTTL = expiredInSec
	return f.signResult, f.signErr
}

func TestAliyunStorageURLUsesCDN(t *testing.T) {
	cases := []struct {
		host string
		key  string
		want string
	}{
		{host: "cdn.example.com", key: "a/b.png", want: "https://cdn.example.com/a/b.png"},
		{host: "https://cdn.example.com/", key: "/a/b.png", want: "https://cdn.example.com/a/b.png"},
		{host: "http://cdn.example.com//", key: "/a/b.png", want: "https://cdn.example.com/a/b.png"},
		{host: " cdn.example.com/assets/ ", key: "/a/b.png", want: "https://cdn.example.com/assets/a/b.png"},
	}
	for _, tc := range cases {
		bucket := &fakeBucket{}
		store := &aliyunStorage{bucket: bucket, cdnHost: tc.host, presignTTL: 60}
		got, err := store.URL(context.Background(), tc.key)
		if err != nil {
			t.Fatalf("URL(host=%q) error = %v", tc.host, err)
		}
		if got != tc.want {
			t.Errorf("URL(host=%q, key=%q) = %q, want %q", tc.host, tc.key, got, tc.want)
		}
		if bucket.signCallCnt != 0 {
			t.Errorf("CDN URL should not sign, got %d calls", bucket.signCallCnt)
		}
	}
}

func TestAliyunStorageURLSignsWithFallbackTTL(t *testing.T) {
	for _, tc := range []struct {
		name string
		ttl  int64
		want int64
	}{
		{name: "zero falls back to 900", ttl: 0, want: 900},
		{name: "negative falls back to 900", ttl: -1, want: 900},
		{name: "configured ttl", ttl: 120, want: 120},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bucket := &fakeBucket{signResult: "https://bucket.example.com/a/b.png?sig=x"}
			store := &aliyunStorage{bucket: bucket, presignTTL: tc.ttl}
			got, err := store.URL(context.Background(), "/a/b.png")
			if err != nil {
				t.Fatalf("URL() error = %v", err)
			}
			if got != bucket.signResult {
				t.Errorf("URL() = %q, want %q", got, bucket.signResult)
			}
			if bucket.signKey != "a/b.png" {
				t.Errorf("SignURL key = %q, want a/b.png", bucket.signKey)
			}
			if bucket.signMethod != oss.HTTPGet {
				t.Errorf("SignURL method = %v, want GET", bucket.signMethod)
			}
			if bucket.signTTL != tc.want {
				t.Errorf("SignURL ttl = %d, want %d", bucket.signTTL, tc.want)
			}
		})
	}
}

func TestAliyunStoragePutDelete(t *testing.T) {
	bucket := &fakeBucket{signResult: "https://bucket.example.com/avatar.png"}
	store := &aliyunStorage{bucket: bucket}

	got, err := store.Put(context.Background(), "avatar.png", strings.NewReader("data"), "image/custom")
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if got != bucket.signResult {
		t.Errorf("Put() url = %q, want %q", got, bucket.signResult)
	}
	if bucket.putKey != "avatar.png" || bucket.putBody != "data" {
		t.Errorf("PutObject key/body = %q/%q", bucket.putKey, bucket.putBody)
	}
	if bucket.putOptions != 1 {
		t.Errorf("PutObject options = %d, want 1 content-type option", bucket.putOptions)
	}

	if err := store.Delete(context.Background(), "avatar.png"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if bucket.deleteKey != "avatar.png" {
		t.Errorf("DeleteObject key = %q", bucket.deleteKey)
	}
}

func TestAliyunStoragePropagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	store := &aliyunStorage{bucket: &fakeBucket{putErr: boom}}
	if _, err := store.Put(context.Background(), "a.txt", strings.NewReader("x"), ""); err == nil || !errors.Is(err, boom) {
		t.Fatalf("Put() error = %v, want boom", err)
	}

	store = &aliyunStorage{bucket: &fakeBucket{deleteErr: boom}}
	if err := store.Delete(context.Background(), "a.txt"); err == nil || !errors.Is(err, boom) {
		t.Fatalf("Delete() error = %v, want boom", err)
	}

	store = &aliyunStorage{bucket: &fakeBucket{signErr: boom}}
	if _, err := store.URL(context.Background(), "a.txt"); err == nil || !errors.Is(err, boom) {
		t.Fatalf("URL() error = %v, want boom", err)
	}
}

func TestResolveContentType(t *testing.T) {
	if got := resolveContentType("a/b.png", ""); !strings.HasPrefix(got, "image/png") {
		t.Errorf("resolveContentType(png) = %q, want image/png", got)
	}
	if got := resolveContentType("a/b.bin", "application/x-custom"); got != "application/x-custom" {
		t.Errorf("resolveContentType(explicit) = %q", got)
	}
	if got := resolveContentType("a/b.unknown-ext", ""); got != "application/octet-stream" {
		t.Errorf("resolveContentType(unknown) = %q, want application/octet-stream", got)
	}
}

func TestNewObjectStorageAliyunValidatesConfig(t *testing.T) {
	valid := conf.OSS{
		Enabled:      true,
		Provider:     "aliyun",
		Endpoint:     "https://oss-cn-hangzhou.aliyuncs.com",
		Bucket:       "planstar-test",
		AccessKey:    "ak",
		AccessSecret: "sk",
	}
	cases := []struct {
		name  string
		blank func(*conf.OSS)
		field string
	}{
		{name: "endpoint", blank: func(c *conf.OSS) { c.Endpoint = " " }, field: "storage.oss.endpoint"},
		{name: "bucket", blank: func(c *conf.OSS) { c.Bucket = "" }, field: "storage.oss.bucket"},
		{name: "access key", blank: func(c *conf.OSS) { c.AccessKey = "" }, field: "storage.oss.access_key"},
		{name: "access secret", blank: func(c *conf.OSS) { c.AccessSecret = "" }, field: "storage.oss.access_secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.blank(&cfg)
			store, err := NewObjectStorage(cfg, nil)
			if err == nil {
				t.Fatalf("NewObjectStorage() = %T, want config error", store)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error = %v, want field %q", err, tc.field)
			}
		})
	}
}

func TestNewObjectStorageAliyunProviderAliases(t *testing.T) {
	base := conf.OSS{
		Enabled:      true,
		Endpoint:     "https://oss-cn-hangzhou.aliyuncs.com",
		Bucket:       "planstar-test",
		AccessKey:    "ak",
		AccessSecret: "sk",
		CDNHost:      "cdn.example.com",
	}
	for _, provider := range []string{"aliyun", "oss", " OSS "} {
		cfg := base
		cfg.Provider = provider
		store, err := NewObjectStorage(cfg, nil)
		if err != nil {
			t.Fatalf("NewObjectStorage(provider=%q) error = %v", provider, err)
		}
		if _, ok := store.(*aliyunStorage); !ok {
			t.Fatalf("NewObjectStorage(provider=%q) = %T, want *aliyunStorage", provider, store)
		}
	}
}

func TestNewObjectStorageDisabledNeverCallsOSS(t *testing.T) {
	// 禁用时即使 provider 指向 aliyun 也应直接返回 Noop，不触发任何 SDK 初始化。
	store, err := NewObjectStorage(conf.OSS{Enabled: false, Provider: "aliyun"}, nil)
	if err != nil {
		t.Fatalf("NewObjectStorage() error = %v", err)
	}
	if _, ok := store.(Noop); !ok {
		t.Fatalf("NewObjectStorage() = %T, want Noop", store)
	}
	url, err := store.Put(context.Background(), "a.png", strings.NewReader("x"), "")
	if err != nil || url != "" {
		t.Fatalf("Noop.Put() = %q, %v", url, err)
	}
}
