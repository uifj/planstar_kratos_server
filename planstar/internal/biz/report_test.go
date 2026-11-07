package biz

import (
	"context"
	"errors"
	"testing"
)

type fakeReportRepo struct {
	listLimit  int
	handleNote string
}

func (f *fakeReportRepo) List(_ context.Context, status string, limit int) ([]*Report, error) {
	f.listLimit = limit
	return []*Report{{ReportID: 1, Status: status}}, nil
}

func (f *fakeReportRepo) Handle(_ context.Context, reportID int64, status, note string) (*Report, error) {
	f.handleNote = note
	return &Report{ReportID: reportID, Status: status, HandleNote: note}, nil
}

func TestReportUsecaseList(t *testing.T) {
	repo := &fakeReportRepo{}
	uc := NewReportUsecase(repo)

	if _, err := uc.List(context.Background(), "unknown", 10); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("status unknown err = %v, want ErrBadRequest", err)
	}
	if _, err := uc.List(context.Background(), "pending", 0); err != nil {
		t.Fatalf("List err = %v, want nil", err)
	}
	if repo.listLimit != 50 {
		t.Fatalf("limit = %d, want 50 (默认值)", repo.listLimit)
	}
}

func TestReportUsecaseHandle(t *testing.T) {
	repo := &fakeReportRepo{}
	uc := NewReportUsecase(repo)

	if _, err := uc.Handle(context.Background(), 0, "accepted", ""); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("reportID 0 err = %v, want ErrBadRequest", err)
	}
	if _, err := uc.Handle(context.Background(), 1, "pending", ""); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("status pending err = %v, want ErrBadRequest", err)
	}

	item, err := uc.Handle(context.Background(), 9001, "rejected", "")
	if err != nil {
		t.Fatalf("Handle err = %v, want nil", err)
	}
	if item.Status != "rejected" || repo.handleNote == "" {
		t.Fatalf("item = %+v, note = %q; want rejected with default note", item, repo.handleNote)
	}
}
