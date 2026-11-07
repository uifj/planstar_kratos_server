package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"

	"planstar/admin/internal/biz"
	"planstar/admin/internal/conf"
)

const (
	defaultOpsTimeout = 5 * time.Second
	maxOpsBody        = 1 << 20
)

// opsClient calls the Planstar business service admin endpoints. Moderation
// rules (including the category blacklist) stay in that service; this client
// only forwards decisions and maps transport failures.
type opsClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewOpsClient creates the ops HTTP client from the planstar config block.
func NewOpsClient(c *conf.Planstar) biz.OpsClient {
	timeout := c.GetOpsTimeout().AsDuration()
	if timeout <= 0 {
		timeout = defaultOpsTimeout
	}
	return &opsClient{
		baseURL: strings.TrimRight(strings.TrimSpace(c.GetOpsBaseUrl()), "/"),
		token:   strings.TrimSpace(c.GetOpsToken()),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *opsClient) ReviewProject(ctx context.Context, id int64, decision string) (*biz.ReviewResult, error) {
	var resp struct {
		ProjectID     int64              `json:"projectId"`
		Title         string             `json:"title"`
		Subtitle      string             `json:"subtitle"`
		CategoryID    int64              `json:"categoryId"`
		CategoryName  string             `json:"categoryName"`
		PublishStatus int32              `json:"publishStatus"`
		FundingStatus string             `json:"fundingStatus"`
		Hits          []biz.BlacklistHit `json:"hits"`
		Blocked       bool               `json:"blocked"`
	}
	path := fmt.Sprintf("/admin/projects/%d/review", id)
	if err := c.post(ctx, path, map[string]string{"decision": decision}, &resp, biz.ErrProjectNotFound); err != nil {
		return nil, err
	}
	return &biz.ReviewResult{
		Project: biz.Project{
			ID:            resp.ProjectID,
			Title:         resp.Title,
			Subtitle:      resp.Subtitle,
			CategoryID:    resp.CategoryID,
			CategoryName:  resp.CategoryName,
			PublishStatus: resp.PublishStatus,
			FundingStatus: resp.FundingStatus,
		},
		Hits:    resp.Hits,
		Blocked: resp.Blocked,
	}, nil
}

func (c *opsClient) HandleReport(ctx context.Context, id int64, status, note string) (*biz.Report, error) {
	var resp struct {
		ReportID    int64      `json:"reportId"`
		TargetType  string     `json:"targetType"`
		TargetID    int64      `json:"targetId"`
		ProjectID   int64      `json:"projectId"`
		UserID      int64      `json:"userId"`
		ReasonCode  string     `json:"reasonCode"`
		Description string     `json:"description"`
		Evidence    []string   `json:"evidence"`
		Status      string     `json:"status"`
		HandleNote  string     `json:"handleNote"`
		HandleTime  *time.Time `json:"handleTime"`
		GmtCreate   time.Time  `json:"gmtCreate"`
	}
	path := fmt.Sprintf("/admin/reports/%d/handle", id)
	if err := c.post(ctx, path, map[string]string{"status": status, "note": note}, &resp, biz.ErrReportNotFound); err != nil {
		return nil, err
	}
	return &biz.Report{
		ID:          resp.ReportID,
		TargetType:  resp.TargetType,
		TargetID:    resp.TargetID,
		ProjectID:   resp.ProjectID,
		UserID:      resp.UserID,
		ReasonCode:  resp.ReasonCode,
		Description: resp.Description,
		Evidence:    resp.Evidence,
		Status:      resp.Status,
		HandleNote:  resp.HandleNote,
		HandleTime:  resp.HandleTime,
		CreatedAt:   resp.GmtCreate,
	}, nil
}

// post sends one JSON request and decodes a successful reply. Transport,
// timeout and unexpected status failures collapse into ErrOpsUnavailable so
// callers report 503 instead of pretending the write happened.
func (c *opsClient) post(ctx context.Context, path string, payload, out any, notFound error) error {
	if c.baseURL == "" {
		return biz.ErrOpsUnavailable
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return biz.ErrOpsUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return biz.ErrOpsUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return biz.ErrOpsUnavailable
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxOpsBody))
	if err != nil {
		return biz.ErrOpsUnavailable
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if err := json.Unmarshal(raw, out); err != nil {
			return biz.ErrOpsUnavailable
		}
		return nil
	case resp.StatusCode == http.StatusBadRequest:
		return opsBadRequest(raw)
	case resp.StatusCode == http.StatusNotFound:
		return notFound
	default:
		return biz.ErrOpsUnavailable
	}
}

// opsBadRequest preserves the ops message and appends rule/keyword pairs from
// the reasons array; it never evaluates the blacklist itself.
func opsBadRequest(raw []byte) error {
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Reasons []struct {
			Rule    string `json:"rule"`
			Keyword string `json:"keyword"`
		} `json:"reasons"`
	}
	_ = json.Unmarshal(raw, &e)
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "ops request rejected"
	}
	if len(e.Reasons) > 0 {
		parts := make([]string, 0, len(e.Reasons))
		for _, reason := range e.Reasons {
			parts = append(parts, fmt.Sprintf("%s(%s)", reason.Rule, reason.Keyword))
		}
		if !strings.Contains(message, parts[0]) {
			message += "：" + strings.Join(parts, "、")
		}
	}
	return kerrors.BadRequest("BAD_REQUEST", message)
}
