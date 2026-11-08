package biz

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeIdeaRepo 是内存版创意仓储，用于验证用例层的归一化与参数校验。
type fakeIdeaRepo struct {
	page     *IdeaFeedPage
	pageErr  error
	detail   *ProjectIdea
	detailEr error
	result   *IdeaVoteResult
	already  bool
	voteErr  error

	lastQuery       IdeaQuery
	lastDetailID    int64
	lastViewerID    int64
	lastVoteIdeaID  int64
	lastVoteUserID  int64
	lastVotePayload *IdeaVote
}

func (f *fakeIdeaRepo) Feed(_ context.Context, q IdeaQuery) (*IdeaFeedPage, error) {
	f.lastQuery = q
	if f.pageErr != nil {
		return nil, f.pageErr
	}
	return f.page, nil
}

func (f *fakeIdeaRepo) Detail(_ context.Context, ideaID, viewerID int64) (*ProjectIdea, error) {
	f.lastDetailID, f.lastViewerID = ideaID, viewerID
	if f.detailEr != nil {
		return nil, f.detailEr
	}
	return f.detail, nil
}

func (f *fakeIdeaRepo) Vote(_ context.Context, ideaID, userID int64, vote *IdeaVote) (*IdeaVoteResult, bool, error) {
	f.lastVoteIdeaID, f.lastVoteUserID, f.lastVotePayload = ideaID, userID, vote
	if f.voteErr != nil {
		return nil, false, f.voteErr
	}
	return f.result, f.already, nil
}

func newIdeaUsecase(repo *fakeIdeaRepo) *ProjectIdeaUsecase {
	return NewProjectIdeaUsecase(repo)
}

func TestIdeaFeedNormalizesSize(t *testing.T) {
	repo := &fakeIdeaRepo{page: &IdeaFeedPage{}}
	uc := newIdeaUsecase(repo)

	page, err := uc.Feed(context.Background(), "  12.5000_8  ", 0, 7)
	if err != nil {
		t.Fatalf("Feed err = %v", err)
	}
	if repo.lastQuery.Size != 10 || repo.lastQuery.Cursor != "12.5000_8" || repo.lastQuery.ViewerID != 7 {
		t.Fatalf("query = %+v, want size 10 / trimmed cursor / viewer 7", repo.lastQuery)
	}
	if page.Items == nil {
		t.Fatal("items should be non-nil empty slice")
	}

	if _, err := uc.Feed(context.Background(), "", 100, 0); err != nil {
		t.Fatalf("Feed err = %v", err)
	}
	if repo.lastQuery.Size != 20 {
		t.Fatalf("size = %d, want 20", repo.lastQuery.Size)
	}
}

func TestIdeaDetailRejectsInvalidID(t *testing.T) {
	uc := newIdeaUsecase(&fakeIdeaRepo{})
	if _, err := uc.Detail(context.Background(), 0, 1); !errors.Is(err, ErrIdeaNotFound) {
		t.Fatalf("err = %v, want ErrIdeaNotFound", err)
	}
}

func TestIdeaVoteSupportSuccess(t *testing.T) {
	repo := &fakeIdeaRepo{
		result: &IdeaVoteResult{IdeaID: 1, SupportCount: 11, OpposeCount: 2, CoinIntentSum: 110},
	}
	uc := newIdeaUsecase(repo)

	result, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{
		VoteType:   1,
		CoinTier:   10,
		ReasonText: "  看好创意  ",
	})
	if err != nil {
		t.Fatalf("Vote err = %v", err)
	}
	if repo.lastVotePayload.CoinTier != 10 || repo.lastVotePayload.ReasonText != "看好创意" {
		t.Fatalf("payload = %+v, want tier 10 and trimmed text", repo.lastVotePayload)
	}
	if repo.lastVotePayload.ReasonCode != "" {
		t.Fatalf("support reasonCode = %q, want empty", repo.lastVotePayload.ReasonCode)
	}
	if result.AlreadyVoted {
		t.Fatal("AlreadyVoted should be false")
	}
	if result.SupportRate != 0.846 {
		t.Fatalf("supportRate = %v, want 0.846", result.SupportRate)
	}
	if result.MyVote == nil || result.MyVote.CoinTier != 10 {
		t.Fatalf("myVote = %+v, want tier 10", result.MyVote)
	}
}

func TestIdeaVoteInvalidTier(t *testing.T) {
	repo := &fakeIdeaRepo{}
	uc := newIdeaUsecase(repo)

	for _, tier := range []int{0, 1, 7, 50} {
		_, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: 1, CoinTier: tier})
		if !errors.Is(err, ErrIdeaInvalidTier) {
			t.Fatalf("tier %d err = %v, want ErrIdeaInvalidTier", tier, err)
		}
	}
	if repo.lastVotePayload != nil {
		t.Fatal("repo should not be called for invalid tier")
	}
}

func TestIdeaVoteInvalidType(t *testing.T) {
	uc := newIdeaUsecase(&fakeIdeaRepo{})
	for _, typ := range []int{0, 3, -1} {
		if _, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: typ, CoinTier: 5}); !errors.Is(err, ErrBadRequest) {
			t.Fatalf("voteType %d err = %v, want ErrBadRequest", typ, err)
		}
	}
}

func TestIdeaVoteRequiresReasonCode(t *testing.T) {
	repo := &fakeIdeaRepo{}
	uc := newIdeaUsecase(repo)

	_, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: 2})
	if !errors.Is(err, ErrIdeaReasonRequired) {
		t.Fatalf("err = %v, want ErrIdeaReasonRequired", err)
	}
	if repo.lastVotePayload != nil {
		t.Fatal("repo should not be called without reasonCode")
	}
}

func TestIdeaVoteRejectsUnknownReasonCode(t *testing.T) {
	uc := newIdeaUsecase(&fakeIdeaRepo{})
	_, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: 2, ReasonCode: "bogus"})
	if !errors.Is(err, ErrIdeaInvalidReason) {
		t.Fatalf("err = %v, want ErrIdeaInvalidReason", err)
	}
}

func TestIdeaVoteOtherRequiresText(t *testing.T) {
	uc := newIdeaUsecase(&fakeIdeaRepo{})
	_, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: 2, ReasonCode: "other", ReasonText: "   "})
	if !errors.Is(err, ErrIdeaReasonRequired) {
		t.Fatalf("err = %v, want ErrIdeaReasonRequired", err)
	}
}

func TestIdeaVoteRejectsOverlongText(t *testing.T) {
	uc := newIdeaUsecase(&fakeIdeaRepo{})

	// 支持理由上限 60 字（按 rune 计数）。
	if _, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{
		VoteType: 1, CoinTier: 5, ReasonText: strings.Repeat("支", 61),
	}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("support text err = %v, want ErrBadRequest", err)
	}
	if _, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{
		VoteType: 1, CoinTier: 5, ReasonText: strings.Repeat("支", 60),
	}); err != nil {
		t.Fatalf("60-rune support text err = %v, want nil", err)
	}

	// 不支持理由上限 100 字。
	if _, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{
		VoteType: 2, ReasonCode: "other", ReasonText: strings.Repeat("反", 101),
	}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("oppose text err = %v, want ErrBadRequest", err)
	}
}

func TestIdeaVoteSelfVote(t *testing.T) {
	repo := &fakeIdeaRepo{voteErr: ErrIdeaSelfVote}
	uc := newIdeaUsecase(repo)

	_, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: 1, CoinTier: 5})
	if !errors.Is(err, ErrIdeaSelfVote) {
		t.Fatalf("err = %v, want ErrIdeaSelfVote", err)
	}
}

func TestIdeaVoteIdempotent(t *testing.T) {
	repo := &fakeIdeaRepo{
		already: true,
		result:  &IdeaVoteResult{IdeaID: 1, SupportCount: 20, OpposeCount: 4, CoinIntentSum: 160},
	}
	uc := newIdeaUsecase(repo)

	result, err := uc.Vote(context.Background(), 1, 9, &IdeaVote{VoteType: 2, ReasonCode: "too_many_similar"})
	if err != nil {
		t.Fatalf("Vote err = %v", err)
	}
	if !result.AlreadyVoted {
		t.Fatal("AlreadyVoted should be true for duplicate vote")
	}
	if result.SupportCount != 20 || result.OpposeCount != 4 {
		t.Fatalf("result = %+v, want existing counts unchanged", result)
	}
	if result.TierDistribution == nil || result.TierDistribution["5"] != 0 {
		t.Fatalf("tierDistribution = %+v, want zero-filled", result.TierDistribution)
	}
}

func TestIdeaVoteRequiresLogin(t *testing.T) {
	uc := newIdeaUsecase(&fakeIdeaRepo{})
	if _, err := uc.Vote(context.Background(), 1, 0, &IdeaVote{VoteType: 1, CoinTier: 5}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestIdeaSupportRate(t *testing.T) {
	cases := []struct {
		support, oppose int
		want            float64
	}{
		{0, 0, 0},
		{10, 0, 1},
		{128, 12, 0.914},
		{1, 2, 0.333},
	}
	for _, c := range cases {
		if got := IdeaSupportRate(c.support, c.oppose); got != c.want {
			t.Fatalf("IdeaSupportRate(%d,%d) = %v, want %v", c.support, c.oppose, got, c.want)
		}
	}
}
