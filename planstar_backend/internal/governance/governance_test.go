package governance

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"planstar/internal/platform/pkg/errcode"
)

// repoRoot 返回仓库根（本包位于 <root>/planstar_backend/internal/governance）。
func repoRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("解析仓库根失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p, "planstar_backend", "go.mod")); err != nil {
		t.Fatalf("仓库根定位错误 %q: %v", p, err)
	}
	return p
}

func readFile(t *testing.T, parts ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("读取 %v 失败: %v", parts, err)
	}
	return string(b)
}

// goSources 列出目录下全部 .go 文件；includeTests 控制是否包含 _test.go。
func goSources(t *testing.T, dir string, includeTests bool) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s 失败: %v", dir, err)
	}
	sort.Strings(out)
	return out
}

func importsOf(t *testing.T, path string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		out = append(out, strings.Trim(imp.Path.Value, `"`))
	}
	return out
}

// TestLayerDependencyDirection 把 AGENTS.md 的分层规则变成 CI 可执行的断言。
func TestLayerDependencyDirection(t *testing.T) {
	root := repoRoot(t)
	internal := filepath.Join(root, "planstar_backend", "internal")

	bizForbidden := []string{
		"net/http", "database/sql",
		"planstar/internal/data", "planstar/internal/service", "planstar/internal/server",
		"github.com/redis/go-redis", "github.com/rabbitmq", "github.com/olivere",
	}
	serviceForbidden := []string{"planstar/internal/data"}

	cases := []struct {
		name      string
		dir       string
		forbidden []string
	}{
		{"biz", filepath.Join(internal, "biz"), bizForbidden},
		{"service", filepath.Join(internal, "service"), serviceForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, path := range goSources(t, tc.dir, false) {
				for _, imp := range importsOf(t, path) {
					for _, bad := range tc.forbidden {
						if imp == bad || strings.HasPrefix(imp, bad+"/") {
							t.Errorf("%s 违反分层规则，禁止 import %q（%s）",
								strings.TrimPrefix(path, root+string(os.PathSeparator)), imp, bad)
						}
					}
				}
			}
		})
	}
}

// TestBizHasNoTransitiveInfraDependency 递归检查 biz 的传递 import 图（ADR-0001 P1）。
// 只跟踪 planstar/ Go module 内部包；外部标准库/SDK 直接按黑名单判定。
// 这条断言比 TestLayerDependencyDirection 更严格：它防止「biz 只 import 一个
// 看起来干净的包，但该包又带进 net/http」的隐性污染。
func TestBizHasNoTransitiveInfraDependency(t *testing.T) {
	root := repoRoot(t)
	planstarDir := filepath.Join(root, "planstar_backend")
	forbidden := []string{
		"net/http", "database/sql",
		"github.com/redis/go-redis", "github.com/rabbitmq", "github.com/olivere/elastic",
	}
	isForbidden := func(imp string) bool {
		for _, bad := range forbidden {
			if imp == bad || strings.HasPrefix(imp, bad+"/") {
				return true
			}
		}
		return false
	}

	var queue []string
	for _, f := range goSources(t, filepath.Join(planstarDir, "internal", "biz"), false) {
		queue = append(queue, importsOf(t, f)...)
	}
	visited := map[string]bool{}
	for len(queue) > 0 {
		imp := queue[0]
		queue = queue[1:]
		if isForbidden(imp) {
			t.Errorf("biz 传递 import 图包含禁止项 %q；领域层必须保持纯 Go", imp)
			continue
		}
		if !strings.HasPrefix(imp, "planstar/") || visited[imp] {
			continue
		}
		visited[imp] = true
		dir := filepath.Join(planstarDir, filepath.FromSlash(strings.TrimPrefix(imp, "planstar/")))
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		for _, f := range goSources(t, dir, false) {
			queue = append(queue, importsOf(t, f)...)
		}
	}
}

var upperLiteralRe = regexp.MustCompile(`"([A-Z][A-Z0-9_]{3,})"`)

// TestErrorCodeSingleSource 校验：
//  1. errcode.All() 中每个码都在 API_CONTRACT.md 有记录；
//  2. 生产代码不再出现错误码字符串字面量（errcode 包自身除外）。
func TestErrorCodeSingleSource(t *testing.T) {
	root := repoRoot(t)
	contract := readFile(t, root, "docs", "api", "API_CONTRACT.md")

	for _, code := range errcode.All() {
		if !strings.Contains(contract, code) {
			t.Errorf("API_CONTRACT.md 未记录错误码 %q；新增码必须同步契约文档", code)
		}
	}

	known := map[string]bool{}
	for _, c := range errcode.All() {
		known[c] = true
	}
	errcodeDir := filepath.Join(root, "planstar_backend", "internal", "platform", "pkg", "errcode")
	for _, path := range goSources(t, filepath.Join(root, "planstar_backend", "internal"), false) {
		if filepath.Dir(path) == errcodeDir {
			continue
		}
		for _, m := range upperLiteralRe.FindAllStringSubmatch(readFile(t, path), -1) {
			if known[m[1]] {
				t.Errorf("%s 仍硬编码错误码 %q，请改用 errcode.%s",
					strings.TrimPrefix(path, root+string(os.PathSeparator)), m[1], constName(m[1]))
			}
		}
	}
}

func constName(code string) string {
	parts := strings.Split(strings.ToLower(code), "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// TestADRIndexComplete 保证 docs/ADR/README.md 台账不漏登记任何 ADR。
func TestADRIndexComplete(t *testing.T) {
	root := repoRoot(t)
	adrDir := filepath.Join(root, "docs", "ADR")
	index := readFile(t, adrDir, "README.md")

	entries, err := os.ReadDir(adrDir)
	if err != nil {
		t.Fatalf("读取 ADR 目录失败: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		switch name {
		case "README.md", "ADR-NNNN.md", "planstar_arch_optimize.md":
			continue // 索引 / 模板 / 已标注的研究材料
		}
		if !strings.HasPrefix(name, "ADR-") {
			t.Errorf("docs/ADR/%s 不符合 ADR 命名规范", name)
		}
		if !strings.Contains(index, name) {
			t.Errorf("docs/ADR/README.md 台账未登记 %s", name)
		}
	}
}

// TestContractQuirkGuards 防止旧口径回归（ADR-0001 Q4）。
func TestContractQuirkGuards(t *testing.T) {
	root := repoRoot(t)
	internal := filepath.Join(root, "planstar_backend", "internal")

	plainCalls := 0
	for _, path := range goSources(t, internal, false) {
		plainCalls += strings.Count(readFile(t, path), "httpx.Plain(")
	}
	if plainCalls != 0 {
		t.Errorf("httpx.Plain 被调用 %d 次；登录失败必须使用统一 JSON 错误体", plainCalls)
	}

	guidelines := readFile(t, root, "agents.md")
	if strings.Contains(guidelines, "Login failure: `5xx`") {
		t.Error("agents.md 仍保留「登录失败 5xx + text/plain」旧口径")
	}
	contract := readFile(t, root, "docs", "api", "API_CONTRACT.md")
	if !strings.Contains(contract, "响应头不下发 `psSession`") {
		t.Error("API_CONTRACT.md 缺少「只写 HttpOnly Cookie」约定")
	}
}

// TestLayoutThresholds 落地 ADR-0001 G1/G2 的阈值。
// 默认严格（go test 即门禁）；显式设置 PLANSTAR_GOVERNANCE_STRICT=0 时降级为报告模式，
// 仅在 P2 大批量移动文件的过渡期临时使用。
func TestLayoutThresholds(t *testing.T) {
	root := repoRoot(t)
	strict := os.Getenv("PLANSTAR_GOVERNANCE_STRICT") != "0"
	report := func(format string, args ...any) {
		if strict {
			t.Errorf(format, args...)
		} else {
			t.Logf("[threshold] "+format, args...)
		}
	}

	const maxFilesPerPkg = 40
	const maxFileBytes = 30 << 10
	const maxMainLines = 80

	internal := filepath.Join(root, "planstar_backend", "internal")
	entries, err := os.ReadDir(internal)
	if err != nil {
		t.Fatalf("读取 internal 目录失败: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(internal, e.Name())
		files := goSources(t, dir, false)
		if len(files) > maxFilesPerPkg {
			report("包 internal/%s 有 %d 个非测试 .go 文件，超过阈值 %d", e.Name(), len(files), maxFilesPerPkg)
		}
		for _, path := range files {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat %s 失败: %v", path, err)
			}
			if info.Size() > maxFileBytes {
				report("文件 %s 大小 %s 超过阈值 %s",
					strings.TrimPrefix(path, root+string(os.PathSeparator)),
					strconv.FormatInt(info.Size(), 10)+"B", "30KB")
			}
		}
	}

	mainGo := filepath.Join(root, "planstar_backend", "cmd", "planstar", "main.go")
	lines := strings.Count(readFile(t, mainGo), "\n") + 1
	if lines > maxMainLines {
		report("cmd/planstar/main.go 有 %d 行，超过阈值 %d（P1 需拆 wiring.go）", lines, maxMainLines)
	}
}

// TestGoToolchainPinned 保证 go.mod / Dockerfile / CI 的 Go 版本一致（OPEN_ITEMS D6/D7）。
// 工具链漂移曾导致 http2.TrailerPrefix 与生成器崩溃（见 CODEGEN_VERIFICATION.md）。
func TestGoToolchainPinned(t *testing.T) {
	root := repoRoot(t)
	goMod := readFile(t, root, "planstar_backend", "go.mod")
	dockerfile := readFile(t, root, "planstar_backend", "Dockerfile")
	ci := readFile(t, root, ".github", "workflows", "ci.yml")

	m := regexp.MustCompile(`(?m)^go (\d+\.\d+\.\d+)\s*$`).FindStringSubmatch(goMod)
	if m == nil {
		t.Fatal("go.mod 缺少 go x.y.z 形式的版本声明")
	}
	want := m[1]
	if !strings.Contains(dockerfile, "golang:"+want) {
		t.Errorf("Dockerfile 的基础镜像不是 golang:%s", want)
	}
	pin := `go-version: "` + want + `"`
	if strings.Count(ci, pin) == 0 {
		t.Errorf("ci.yml 缺少 %s", pin)
	}
	tools, err := os.Stat(filepath.Join(root, "planstar_backend", ".golangci.yml"))
	if err != nil || tools.IsDir() {
		t.Error("缺少 planstar_backend/.golangci.yml（A6 必过门禁的配置）")
	}
}
