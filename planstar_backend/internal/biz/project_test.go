package biz

import (
	"context"
	"errors"
	"testing"
)

// fakeProjectRepo 只实现被用例触达的方法，其余沿用 nil 接口。
type fakeProjectRepo struct {
	ProjectRepo

	gotQuery   ProjectQuery
	gotUserID  int64
	gotProject int64
	gotInput   *PledgeInput
	gotReview  *ReviewInput
	gotReport  *ReportInput
	gotRanking struct {
		typ        string
		categoryID int64
		limit      int
	}

	supporterOnly bool
	supporter     bool
	listErr       error

	moderation *ProjectModeration
	modErr     error
	gotStatus  int
}

func (f *fakeProjectRepo) List(_ context.Context, q ProjectQuery) ([]*ProjectSummary, error) {
	f.gotQuery = q
	return nil, f.listErr
}

func (f *fakeProjectRepo) HasSupporterOnly(_ context.Context, _ int64) (bool, error) {
	return f.supporterOnly, nil
}

func (f *fakeProjectRepo) IsSupporter(_ context.Context, _, _ int64) (bool, error) {
	return f.supporter, nil
}

func (f *fakeProjectRepo) Updates(_ context.Context, projectID int64, _ bool, _, _ int) ([]*ProjectUpdate, error) {
	f.gotProject = projectID
	return nil, nil
}

func (f *fakeProjectRepo) CreatePledge(_ context.Context, userID, projectID int64, in *PledgeInput) (*Pledge, error) {
	f.gotUserID, f.gotProject, f.gotInput = userID, projectID, in
	return &Pledge{PledgeID: 1}, nil
}

func (f *fakeProjectRepo) CreateReview(_ context.Context, _, _ int64, in *ReviewInput) (*ProjectReview, error) {
	f.gotReview = in
	return &ProjectReview{ReviewID: 1}, nil
}

func (f *fakeProjectRepo) CreateReport(_ context.Context, _, _ int64, in *ReportInput) (*ProjectReport, error) {
	f.gotReport = in
	return &ProjectReport{ReportID: 1}, nil
}

func (f *fakeProjectRepo) Ranking(_ context.Context, typ string, categoryID int64, limit int) ([]*RankingItem, error) {
	f.gotRanking.typ, f.gotRanking.categoryID, f.gotRanking.limit = typ, categoryID, limit
	return nil, nil
}

func (f *fakeProjectRepo) Faqs(_ context.Context, _ int64) ([]*ProjectFaq, error) {
	return nil, nil
}

func TestProjectListNormalizesPagingAndRejectsBadEnums(t *testing.T) {
	repo := &fakeProjectRepo{}
	uc := NewProjectUsecase(repo)

	if _, err := uc.List(context.Background(), ProjectQuery{Page: 0, PageSize: 101}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotQuery.Page != 1 || repo.gotQuery.PageSize != 20 {
		t.Fatalf("paging not normalized: %+v", repo.gotQuery)
	}

	if _, err := uc.List(context.Background(), ProjectQuery{Sort: "unknown"}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("want ErrBadRequest for bad sort, got %v", err)
	}
	if _, err := uc.List(context.Background(), ProjectQuery{FundingStatus: "closed"}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("want ErrBadRequest for bad funding status, got %v", err)
	}
}

func TestProjectUpdatesHonorsSupporterVisibility(t *testing.T) {
	repo := &fakeProjectRepo{supporterOnly: true}
	uc := NewProjectUsecase(repo)

	if _, err := uc.Updates(context.Background(), 4001, 0, 1, 20); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous should get ErrUnauthorized, got %v", err)
	}

	repo.supporter = false
	if _, err := uc.Updates(context.Background(), 4001, 6, 1, 20); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-supporter should get ErrForbidden, got %v", err)
	}

	repo.supporter = true
	if _, err := uc.Updates(context.Background(), 4001, 6, 1, 20); err != nil {
		t.Fatalf("supporter should pass, got %v", err)
	}
}

func TestCreatePledgeValidation(t *testing.T) {
	repo := &fakeProjectRepo{}
	uc := NewProjectUsecase(repo)

	if _, err := uc.CreatePledge(context.Background(), 0, 4001, &PledgeInput{RewardTierID: 1}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	if _, err := uc.CreatePledge(context.Background(), 6, 4001, &PledgeInput{}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("missing tier should be bad request, got %v", err)
	}
	if _, err := uc.CreatePledge(context.Background(), 6, 4001, &PledgeInput{RewardTierID: 1, Quantity: 100}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("quantity over limit should be bad request, got %v", err)
	}

	if _, err := uc.CreatePledge(context.Background(), 6, 4001, &PledgeInput{RewardTierID: 9}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotInput.Quantity != 1 {
		t.Fatalf("quantity should default to 1, got %d", repo.gotInput.Quantity)
	}
}

func TestCreateReviewValidation(t *testing.T) {
	repo := &fakeProjectRepo{}
	uc := NewProjectUsecase(repo)

	if _, err := uc.CreateReview(context.Background(), 0, 4001, &ReviewInput{PledgeID: 1, Rating: 5}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	if _, err := uc.CreateReview(context.Background(), 6, 4001, &ReviewInput{PledgeID: 1, Rating: 6}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("rating out of range should be bad request, got %v", err)
	}
	if _, err := uc.CreateReview(context.Background(), 6, 4001, &ReviewInput{PledgeID: 1, Rating: 5, Images: nil}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotReview.Images == nil {
		t.Fatal("images should be normalized to empty slice")
	}
}

func TestCreateReportValidation(t *testing.T) {
	repo := &fakeProjectRepo{}
	uc := NewProjectUsecase(repo)

	base := &ReportInput{TargetType: "project", TargetID: 4001, ReasonCode: "fraud"}
	if _, err := uc.CreateReport(context.Background(), 0, 4001, base); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	base.ReasonCode = "unknown"
	if _, err := uc.CreateReport(context.Background(), 6, 4001, base); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad reason should be bad request, got %v", err)
	}
	base.ReasonCode = "fraud"
	if _, err := uc.CreateReport(context.Background(), 6, 4001, base); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRankingsDefaultsAndLimits(t *testing.T) {
	repo := &fakeProjectRepo{}
	uc := NewProjectUsecase(repo)

	if _, err := uc.Rankings(context.Background(), "", 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotRanking.typ != "hot" || repo.gotRanking.limit != 10 {
		t.Fatalf("ranking defaults wrong: %+v", repo.gotRanking)
	}
	if _, err := uc.Rankings(context.Background(), "achievement", 9, 999); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotRanking.limit != 50 || repo.gotRanking.categoryID != 9 {
		t.Fatalf("ranking limits wrong: %+v", repo.gotRanking)
	}
	if _, err := uc.Rankings(context.Background(), "bogus", 0, 10); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("want ErrBadRequest, got %v", err)
	}
}

func (f *fakeProjectRepo) Moderation(_ context.Context, _ int64) (*ProjectModeration, error) {
	if f.modErr != nil {
		return nil, f.modErr
	}
	return f.moderation, nil
}

func (f *fakeProjectRepo) SetPublishStatus(_ context.Context, _ int64, status int) error {
	f.gotStatus = status
	return nil
}

func TestCheckProjectBlacklistRules(t *testing.T) {
	hits := CheckProjectBlacklist("股权分红型企划", 100000, "按季度交付", "存在延期风险")
	if len(hits) == 0 || hits[0].Rule != "金融集资" {
		t.Fatalf("financial keyword should be blocked, got %+v", hits)
	}

	hits = CheckProjectBlacklist("山区儿童慈善募捐", 100000, "按月反馈", "存在延期风险")
	if len(hits) == 0 || hits[0].Rule != "慈善募捐" {
		t.Fatalf("charity keyword should be blocked, got %+v", hits)
	}

	hits = CheckProjectBlacklist("原创设定集", 0, "", "")
	var air bool
	for _, hit := range hits {
		if hit.Rule == "空气企划" {
			air = true
		}
	}
	if !air {
		t.Fatalf("missing goal/delivery/risk should be blocked, got %+v", hits)
	}

	if hits := CheckProjectBlacklist("原创设定集", 500000, "60 天内发货", "可能延期"); len(hits) != 0 {
		t.Fatalf("clean project should pass, got %+v", hits)
	}
}

func TestReviewProjectBlocksBlacklistedApprove(t *testing.T) {
	blocked := &ProjectModeration{
		ProjectID: 4009,
		Hits:      []BlacklistHit{{Rule: "金融集资", Keyword: "分红"}},
		Blocked:   true,
	}
	repo := &fakeProjectRepo{moderation: blocked}
	uc := NewProjectUsecase(repo)

	_, err := uc.ReviewProject(context.Background(), 4009, "approve")
	var blockedErr *BlacklistBlockedError
	if !errors.As(err, &blockedErr) {
		t.Fatalf("approve should be blocked with reasons, got %v", err)
	}
	if repo.gotStatus != 0 {
		t.Fatalf("publish status must not change on blocked approve, got %d", repo.gotStatus)
	}

	if _, err := uc.ReviewProject(context.Background(), 4009, "reject"); err != nil {
		t.Fatalf("reject should always be allowed: %v", err)
	}
	if repo.gotStatus != 3 {
		t.Fatalf("reject should set publish_status=3, got %d", repo.gotStatus)
	}

	clean := &ProjectModeration{ProjectID: 4009}
	repo = &fakeProjectRepo{moderation: clean}
	uc = NewProjectUsecase(repo)
	if _, err := uc.ReviewProject(context.Background(), 4009, "approve"); err != nil {
		t.Fatalf("clean approve should pass: %v", err)
	}
	if repo.gotStatus != 1 {
		t.Fatalf("approve should set publish_status=1, got %d", repo.gotStatus)
	}
}

func TestProjectHelpers(t *testing.T) {
	if got := AchievementPermille(150, 100); got != 1500 {
		t.Fatalf("achievement permille = %d, want 1500", got)
	}
	if got := AchievementPermille(150, 0); got != 0 {
		t.Fatalf("zero goal should be 0, got %d", got)
	}
	if !IsReviewRisky("支持后返现") {
		t.Fatal("risk word should be detected")
	}
	if IsReviewRisky("支持一下，期待发货") {
		t.Fatal("normal review should not be risky")
	}
}
