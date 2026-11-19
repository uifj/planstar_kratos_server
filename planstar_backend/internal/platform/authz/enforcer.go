// Package authz 提供 Casbin RBAC 判定、策略热更新与 HTTP 授权中间件（ADR-0007）。
package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/redis/go-redis/v9"
)

// Grouping 是一条 g 策略：用户 → 角色。
type Grouping struct {
	User string `json:"user"`
	Role string `json:"role"`
}

// PolicyStore 持久化运行时增量策略；内置基线不入库、不可删除。
type PolicyStore interface {
	Load(ctx context.Context) ([]Rule, []Grouping, error)
	Save(ctx context.Context, rules []Rule, grouping []Grouping) error
}

type policySnapshot struct {
	Rules    []Rule     `json:"rules"`
	Grouping []Grouping `json:"grouping"`
}

// Enforcer 包装 Casbin，保证策略变更与判定在锁内一致，并支持热重建。
type Enforcer struct {
	mu       sync.RWMutex
	inner    *casbin.Enforcer
	store    PolicyStore
	rules    []Rule
	grouping []Grouping
}

func New(store PolicyStore) (*Enforcer, error) {
	m, err := model.NewModelFromString(BuiltinModel)
	if err != nil {
		return nil, fmt.Errorf("解析内置 RBAC 模型失败: %w", err)
	}
	inner, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("创建 Casbin enforcer 失败: %w", err)
	}
	e := &Enforcer{inner: inner, store: store}
	e.rules = append([]Rule(nil), BuiltinPolicies...)
	if store != nil {
		rules, grouping, err := store.Load(context.Background())
		if err != nil {
			return nil, fmt.Errorf("加载运行时 RBAC 策略失败: %w", err)
		}
		e.rules = append(e.rules, rules...)
		e.grouping = append(e.grouping, grouping...)
	}
	if err := e.rebuildLocked(); err != nil {
		return nil, err
	}
	return e, nil
}

// rebuildLocked 全量重建策略；调用方必须持有写锁。
func (e *Enforcer) rebuildLocked() error {
	e.inner.ClearPolicy()
	for _, r := range e.rules {
		if _, err := e.inner.AddPolicy(r.Sub, r.Obj, r.Act); err != nil {
			return fmt.Errorf("加载 p 策略 %+v 失败: %w", r, err)
		}
	}
	for _, g := range e.grouping {
		if _, err := e.inner.AddGroupingPolicy(g.User, g.Role); err != nil {
			return fmt.Errorf("加载 g 策略 %+v 失败: %w", g, err)
		}
	}
	return nil
}

// Authorize 对身份的所有 subject（用户名 + 角色码 + 内置已认证伪角色）逐个判定，
// 任一 allow 即放行；没有任何命中即拒绝（默认拒绝）。
func (e *Enforcer) Authorize(_ context.Context, subjects []string, obj, act string) (bool, error) {
	if obj == "" || act == "" {
		return false, fmt.Errorf("授权判据缺少 obj/act")
	}
	all := append([]string(nil), subjects...)
	all = append(all, SubjectAuthenticated)
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, sub := range all {
		if strings.TrimSpace(sub) == "" {
			continue
		}
		ok, err := e.inner.Enforce(sub, obj, act)
		if err != nil {
			return false, fmt.Errorf("Casbin Enforce 失败: %w", err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// Rules 返回当前生效策略快照（含内置基线），供管理端点展示。
func (e *Enforcer) Rules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]Rule(nil), e.rules...)
}

func (e *Enforcer) Grouping() []Grouping {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]Grouping(nil), e.grouping...)
}

func (e *Enforcer) AddRule(ctx context.Context, r Rule) error {
	if err := validateRule(r); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(e.rules, r)
	if err := e.persistLocked(ctx); err != nil {
		e.rules = e.rules[:len(e.rules)-1]
		return err
	}
	return e.rebuildLocked()
}

func (e *Enforcer) RemoveRule(ctx context.Context, r Rule) error {
	if err := validateRule(r); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	idx := -1
	for i, cur := range e.rules {
		if cur == r {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("策略不存在")
	}
	if idx < len(BuiltinPolicies) {
		return fmt.Errorf("内置基线策略不可删除（防止锁死运维入口）")
	}
	e.rules = append(e.rules[:idx], e.rules[idx+1:]...)
	if err := e.persistLocked(ctx); err != nil {
		return err
	}
	return e.rebuildLocked()
}

func (e *Enforcer) AddGrouping(ctx context.Context, g Grouping) error {
	g.User, g.Role = strings.TrimSpace(g.User), strings.TrimSpace(g.Role)
	if g.User == "" || g.Role == "" {
		return fmt.Errorf("g 策略的 user/role 不能为空")
	}
	// 超级管理员不能通过 g 策略授予，否则 rbac_admin 可给自己授 superadmin 提权。
	if !IsGrantableRole(g.Role) {
		return fmt.Errorf("角色 %q 不在可授予白名单内（superadmin 仅由 break-glass 运维令牌持有）", g.Role)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, cur := range e.grouping {
		if cur == g {
			return nil
		}
	}
	e.grouping = append(e.grouping, g)
	if err := e.persistLocked(ctx); err != nil {
		e.grouping = e.grouping[:len(e.grouping)-1]
		return err
	}
	return e.rebuildLocked()
}

// Reload 从存储重读增量并全量重建（策略热更新，参考 go-wind 的 ResetPolicies）。
func (e *Enforcer) Reload(ctx context.Context) error {
	if e.store == nil {
		return nil
	}
	rules, grouping, err := e.store.Load(ctx)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = append(append([]Rule(nil), BuiltinPolicies...), rules...)
	e.grouping = grouping
	return e.rebuildLocked()
}

// runtimeSnapshot 只持久化增量（跳过前 len(BuiltinPolicies) 条基线）。
func (e *Enforcer) persistLocked(ctx context.Context) error {
	if e.store == nil {
		return nil
	}
	rules := append([]Rule(nil), e.rules[len(BuiltinPolicies):]...)
	grouping := append([]Grouping(nil), e.grouping...)
	return e.store.Save(ctx, rules, grouping)
}

func validateRule(r Rule) error {
	if strings.TrimSpace(r.Sub) == "" || strings.TrimSpace(r.Obj) == "" || strings.TrimSpace(r.Act) == "" {
		return fmt.Errorf("策略 sub/obj/act 不能为空")
	}
	if !strings.HasPrefix(r.Obj, "/") {
		return fmt.Errorf("策略 obj 必须以 / 开头")
	}
	// 运行时策略禁止全放行通配符：唯一的 {superadmin,*,*} 是编译期基线，不可通过 API 复制（PRODUCTION_AUDIT S1）。
	if r.Obj == "*" || r.Act == "*" {
		return fmt.Errorf("运行时策略禁止使用 obj=* 或 act=* 全放行通配")
	}
	if r.Sub == RuleSubjectSuperAdmin {
		return fmt.Errorf("superadmin 基线策略不可通过 API 增删")
	}
	return nil
}

// ---------------------------------------------------------------------------
// 策略存储：内存（测试/单实例降级）与 Redis（生产，兼容 read_only 容器）。

type MemoryPolicyStore struct {
	mu       sync.RWMutex
	rules    []Rule
	grouping []Grouping
}

func NewMemoryPolicyStore() *MemoryPolicyStore { return &MemoryPolicyStore{} }

func (m *MemoryPolicyStore) Load(_ context.Context) ([]Rule, []Grouping, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Rule(nil), m.rules...), append([]Grouping(nil), m.grouping...), nil
}

func (m *MemoryPolicyStore) Save(_ context.Context, rules []Rule, grouping []Grouping) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append([]Rule(nil), rules...)
	m.grouping = append([]Grouping(nil), grouping...)
	return nil
}

const redisPolicyKey = "planstar:authz:policies"

type RedisPolicyStore struct {
	client *redis.Client
}

func NewRedisPolicyStore(client *redis.Client) *RedisPolicyStore {
	return &RedisPolicyStore{client: client}
}

func (s *RedisPolicyStore) Load(ctx context.Context) ([]Rule, []Grouping, error) {
	if s.client == nil {
		return nil, nil, nil
	}
	raw, err := s.client.Get(ctx, redisPolicyKey).Bytes()
	if err == redis.Nil {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var snap policySnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, nil, fmt.Errorf("解析 Redis RBAC 策略失败: %w", err)
	}
	return snap.Rules, snap.Grouping, nil
}

func (s *RedisPolicyStore) Save(ctx context.Context, rules []Rule, grouping []Grouping) error {
	if s.client == nil {
		return nil
	}
	raw, err := json.Marshal(policySnapshot{Rules: rules, Grouping: grouping})
	if err != nil {
		return err
	}
	return s.client.Set(ctx, redisPolicyKey, raw, 0).Err()
}
