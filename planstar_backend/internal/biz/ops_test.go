package biz

import (
	"context"
	"errors"
	"testing"
)

// fakeOpsRepo records delegations so tests can assert the usecase validates at
// the boundary and only forwards well-formed calls to the repo.
type fakeOpsRepo struct {
	setID         int64
	setStatus     int32
	setCalls      int
	reviewID      int64
	reviewApprove bool
	reviewCalls   int
	err           error
}

func (f *fakeOpsRepo) SetFeedStatus(_ context.Context, id int64, status int32) error {
	return f.recordSet(id, status)
}
func (f *fakeOpsRepo) SetGoodsStatus(_ context.Context, id int64, status int32) error {
	return f.recordSet(id, status)
}
func (f *fakeOpsRepo) SetGoodsSKUStatus(_ context.Context, id int64, status int32) error {
	return f.recordSet(id, status)
}
func (f *fakeOpsRepo) recordSet(id int64, status int32) error {
	f.setID, f.setStatus = id, status
	f.setCalls++
	return f.err
}
func (f *fakeOpsRepo) ReviewRefund(_ context.Context, id int64, approve bool) (*RefundReview, error) {
	f.reviewID, f.reviewApprove = id, approve
	f.reviewCalls++
	if f.err != nil {
		return nil, f.err
	}
	return &RefundReview{Status: "success"}, nil
}

func TestOpsSetStatusValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		method  func(*OpsUsecase, context.Context, int64, int32) error
		id      int64
		status  int32
		wantErr bool
	}{
		{"feed hide", (*OpsUsecase).SetFeedStatus, 1, 0, false},
		{"feed show", (*OpsUsecase).SetFeedStatus, 1, 1, false},
		{"feed zero id", (*OpsUsecase).SetFeedStatus, 0, 1, true},
		{"feed negative id", (*OpsUsecase).SetFeedStatus, -1, 1, true},
		{"feed status above range", (*OpsUsecase).SetFeedStatus, 1, 2, true},
		{"feed status below range", (*OpsUsecase).SetFeedStatus, 1, -1, true},
		{"goods off", (*OpsUsecase).SetGoodsStatus, 1, 0, false},
		{"goods on", (*OpsUsecase).SetGoodsStatus, 1, 1, false},
		{"goods zero id", (*OpsUsecase).SetGoodsStatus, 0, 1, true},
		{"goods status above range", (*OpsUsecase).SetGoodsStatus, 1, 2, true},
		{"sku off", (*OpsUsecase).SetGoodsSKUStatus, 2, 0, false},
		{"sku on", (*OpsUsecase).SetGoodsSKUStatus, 2, 1, false},
		{"sku zero id", (*OpsUsecase).SetGoodsSKUStatus, 0, 0, true},
		{"sku status below range", (*OpsUsecase).SetGoodsSKUStatus, 2, -1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeOpsRepo{}
			uc := NewOpsUsecase(repo)
			err := tc.method(uc, ctx, tc.id, tc.status)
			if tc.wantErr {
				if !errors.Is(err, ErrBadRequest) {
					t.Fatalf("err = %v, want ErrBadRequest", err)
				}
				if repo.setCalls != 0 {
					t.Fatalf("repo called %d times for invalid input, want 0", repo.setCalls)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if repo.setCalls != 1 || repo.setID != tc.id || repo.setStatus != tc.status {
				t.Fatalf("delegation = id %d status %d calls %d, want %d/%d/1",
					repo.setID, repo.setStatus, repo.setCalls, tc.id, tc.status)
			}
		})
	}
}

func TestOpsSetStatusPropagatesRepoError(t *testing.T) {
	repo := &fakeOpsRepo{err: ErrNotFound}
	uc := NewOpsUsecase(repo)
	if err := uc.SetFeedStatus(context.Background(), 1, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want repo ErrNotFound", err)
	}
}

func TestOpsReviewRefundValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		id        int64
		approve   bool
		wantErr   bool
		wantCalls int
	}{
		{"zero id rejected", 0, true, true, 0},
		{"negative id rejected", -1, true, true, 0},
		{"approve delegated", 7, true, false, 1},
		{"reject delegated", 7, false, false, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeOpsRepo{}
			uc := NewOpsUsecase(repo)
			out, err := uc.ReviewRefund(ctx, tc.id, tc.approve)
			if tc.wantErr {
				if !errors.Is(err, ErrBadRequest) {
					t.Fatalf("err = %v, want ErrBadRequest", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error %v", err)
			} else if out.Status != "success" {
				t.Fatalf("out = %+v, want success", out)
			}
			if repo.reviewCalls != tc.wantCalls {
				t.Fatalf("review calls = %d, want %d", repo.reviewCalls, tc.wantCalls)
			}
			if tc.wantCalls == 1 && (repo.reviewID != tc.id || repo.reviewApprove != tc.approve) {
				t.Fatalf("delegation = id %d approve %v, want %d/%v",
					repo.reviewID, repo.reviewApprove, tc.id, tc.approve)
			}
		})
	}
}
