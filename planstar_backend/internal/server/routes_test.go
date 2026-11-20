package server_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// routeInventoryCount / routeInventoryUnique 与 docs/api/API_CONTRACT.md §1 的口径一致。
// 修改 service.routes() 时必须同步这两个常量与契约文档，避免「只比集合不比条数」的漂移。
const (
	routeInventoryCount  = 59
	routeInventoryUnique = 58
)

// TestRouteInventory 校验路由条数、唯一端点数与尾斜杠别名数量。
// 这是 TestRoutesMatchContract 的补充：集合比对无法发现「新增一条路由且文档也跟着写错」。
func TestRouteInventory(t *testing.T) {
	routes := (&service.Service{}).Routes()
	if len(routes) != routeInventoryCount {
		t.Errorf("注册路由条数 = %d，期望 %d；请同步 service.routes()、API_CONTRACT.md §1 与本常量",
			len(routes), routeInventoryCount)
	}

	unique := map[string]bool{}
	trailingSlash := 0
	for _, pattern := range routes {
		unique[normalizeRoute(pattern)] = true
		_, path, _ := strings.Cut(pattern, " ")
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			trailingSlash++
			if pattern != "GET /v1/user/" {
				t.Errorf("出现非预期的尾斜杠路由 %q；历史别名只允许 GET /v1/user/", pattern)
			}
		}
	}
	if len(unique) != routeInventoryUnique {
		t.Errorf("唯一端点数 = %d，期望 %d", len(unique), routeInventoryUnique)
	}
	if trailingSlash != 1 {
		t.Errorf("尾斜杠别名数量 = %d，期望 1", trailingSlash)
	}
	if !unique["GET /v1/user"] {
		t.Error("缺少 GET /v1/user 主路由；别名不能替代主路由")
	}
}

// TestContractDeclaresSameInventory 防止契约文档里的条数描述与代码常量漂移。
func TestContractDeclaresSameInventory(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "api", "API_CONTRACT.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取契约文档失败: %v", err)
	}
	doc := string(b)
	wantRegistered := strconv.Itoa(routeInventoryCount) + " 条路由"
	wantUnique := "**" + strconv.Itoa(routeInventoryUnique) + " 个唯一端点**"
	if !strings.Contains(doc, wantRegistered) || !strings.Contains(doc, wantUnique) {
		t.Errorf("API_CONTRACT.md §1 的条数描述与代码不一致，期望包含 %q 与 %q", wantRegistered, wantUnique)
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
