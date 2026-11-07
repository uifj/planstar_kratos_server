package server_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"planstar/internal/service"
)

// routeLineRe 匹配契约文档中独立成行的 `METHOD /path`。
var routeLineRe = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(/\S*)$`)

// tableRowRe 兼容契约文档的编号表格行：| 1 | POST | `v1/user/login` | ... |
var tableRowRe = regexp.MustCompile("^\\|\\s*\\d+\\s*\\|\\s*(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\\s*\\|\\s*`([^`]+)`")

// TestRoutesMatchContract 校验后端注册路由与 docs/api/API_CONTRACT.md 是否一致。
func TestRoutesMatchContract(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "api", "API_CONTRACT.md")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("读取契约文档失败: %v", err)
	}
	defer f.Close()

	docRoutes := parseContractRoutes(t, f)
	if len(docRoutes) == 0 {
		t.Fatalf("未从 %s 解析到任何端点，请检查文档是否包含 `METHOD /path` 或编号表格", path)
	}

	registered := map[string]bool{}
	for _, pattern := range (&service.Service{}).Routes() {
		// GET /v1/user/ 是 GET /v1/user 的历史别名，归一化后自然合并，无需单独比对。
		registered[normalizeRoute(pattern)] = true
	}

	var missing, extra []string
	for route := range docRoutes {
		if !registered[route] {
			missing = append(missing, route)
		}
	}
	for route := range registered {
		if !docRoutes[route] {
			extra = append(extra, route)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("路由与契约不一致\n契约有但后端未注册 (%d): %v\n后端有但契约未声明 (%d): %v",
			len(missing), missing, len(extra), extra)
	}
}

// parseContractRoutes 从契约文档提取 METHOD + PATH 集合，跳过「未实现/仅注释」章节。
func parseContractRoutes(t *testing.T, r io.Reader) map[string]bool {
	t.Helper()
	routes := map[string]bool{}
	skipSection := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			skipSection = strings.Contains(trimmed, "未实现") || strings.Contains(trimmed, "仅注释")
			continue
		}
		if skipSection {
			continue
		}

		if m := routeLineRe.FindStringSubmatch(trimmed); m != nil {
			routes[normalizeRoute(m[1]+" "+m[2])] = true
			continue
		}
		if m := tableRowRe.FindStringSubmatch(trimmed); m != nil {
			routes[normalizeRoute(m[1]+" "+m[2])] = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("读取契约文档失败: %v", err)
	}
	return routes
}

// normalizeRoute 统一为 "METHOD /path"：补前导斜杠、去尾斜杠与查询串。
func normalizeRoute(route string) string {
	method, path, ok := strings.Cut(strings.TrimSpace(route), " ")
	if !ok {
		return strings.TrimSpace(route)
	}
	path = strings.TrimSpace(path)
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		path = "/"
	}
	return method + " " + path
}
