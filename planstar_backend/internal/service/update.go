package service

import (
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"

	"planstar/internal/biz"
	"planstar/internal/platform/pkg/httpx"
)

// updateCheck 返回最新可用版本（匿名可访问，移动端 APK 走此端点）。
// 无更新或请求版本已是最新时返回 {"latest":true}。
func (s *Service) updateCheck(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "android"
	}
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "stable"
	}
	versionCode := httpx.Int64Query(r, "versionCode", 0)

	rel, err := s.update.LatestRelease(r.Context(), platform, channel)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if rel == nil || rel.VersionCode <= versionCode {
		httpx.JSON(w, http.StatusOK, map[string]any{"latest": true})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"latest":      false,
		"versionCode": rel.VersionCode,
		"versionName": rel.VersionName,
		"downloadUrl": rel.DownloadURL,
		"sha256":      rel.SHA256,
		"sizeBytes":   rel.SizeBytes,
		"force":       rel.Force(versionCode),
		"changelog":   rel.Changelog,
	})
}

// updateAppcast 输出 WinSparkle 兼容的 appcast XML（匿名可访问，桌面 EXE 走此端点）。
func (s *Service) updateAppcast(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "windows"
	}
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "stable"
	}
	rel, err := s.update.LatestRelease(r.Context(), platform, channel)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(appcastDocument(rel)))
}

// appcastDocument 渲染 WinSparkle/Sparkle 兼容的 RSS2.0 appcast。
func appcastDocument(rel *biz.Release) string {
	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString(`<rss version="2.0" ` +
		`xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/">` + "\n")
	b.WriteString("<channel>\n")
	b.WriteString("  <title>Planstar</title>\n")
	b.WriteString("  <description>Planstar 应用更新</description>\n")
	b.WriteString("  <language>zh</language>\n")
	if rel != nil {
		b.WriteString("  <item>\n")
		b.WriteString("    <title>Version " + xmlEscape(rel.VersionName) + "</title>\n")
		b.WriteString("    <description>" + xmlEscape(rel.Changelog) + "</description>\n")
		if !rel.ReleasedAt.IsZero() {
			b.WriteString("    <pubDate>" + rel.ReleasedAt.UTC().Format(http.TimeFormat) + "</pubDate>\n")
		}
		b.WriteString("    <enclosure url=\"" + xmlEscape(rel.DownloadURL) + "\" " +
			"sparkle:version=\"" + xmlEscape(rel.VersionName) + "\" " +
			"sparkle:os=\"windows\" " +
			"length=\"" + strconv.FormatInt(rel.SizeBytes, 10) + "\" " +
			"type=\"application/octet-stream\" />\n")
		b.WriteString("  </item>\n")
	}
	b.WriteString("</channel>\n</rss>\n")
	return b.String()
}

// xmlEscape 转义 XML 文本/属性值。
func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
