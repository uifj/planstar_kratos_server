package service

import "testing"

func TestIdeaVoteLimiterUserLimit(t *testing.T) {
	l := newIdeaVoteLimiter()
	for i := 0; i < ideaVoteUserLimit; i++ {
		if !l.allow(1, "10.0.0.1") {
			t.Fatalf("vote %d should be allowed", i+1)
		}
	}
	if l.allow(1, "10.0.0.1") {
		t.Fatalf("vote %d should be rejected by user limit", ideaVoteUserLimit+1)
	}
}

func TestIdeaVoteLimiterIPLimit(t *testing.T) {
	l := newIdeaVoteLimiter()
	for i := 0; i < ideaVoteIPLimit; i++ {
		if !l.allow(int64(i+1), "10.0.0.2") {
			t.Fatalf("vote %d should be allowed", i+1)
		}
	}
	if l.allow(int64(ideaVoteIPLimit+1), "10.0.0.2") {
		t.Fatalf("vote %d should be rejected by ip limit", ideaVoteIPLimit+1)
	}
}

func TestIdeaVoteLimiterUsersIndependent(t *testing.T) {
	l := newIdeaVoteLimiter()
	for i := 0; i < ideaVoteUserLimit; i++ {
		if !l.allow(1, "10.0.0.3") {
			t.Fatalf("user 1 vote %d should be allowed", i+1)
		}
	}
	if l.allow(1, "10.0.0.3") {
		t.Fatal("user 1 should be limited")
	}
	if !l.allow(2, "10.0.0.3") {
		t.Fatal("user 2 should not be affected by user 1 limit")
	}
}
