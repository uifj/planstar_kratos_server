package service

import (
	"sync"
	"time"
)

// 企划创意投票频控阈值（Spec §4.7）：单用户 60 次/小时、单 IP 300 次/小时。
const (
	ideaVoteUserLimit = 60
	ideaVoteIPLimit   = 300
	ideaVoteWindow    = time.Hour
)

// ideaVoteLimiter 是投票频控的进程内滑动窗口限流器（单机语义）。
// 说明：多实例部署时各实例独立计数；窗口为固定一小时滚动。
type ideaVoteLimiter struct {
	mu    sync.Mutex
	users map[int64][]time.Time
	ips   map[string][]time.Time
}

func newIdeaVoteLimiter() *ideaVoteLimiter {
	return &ideaVoteLimiter{
		users: map[int64][]time.Time{},
		ips:   map[string][]time.Time{},
	}
}

// allow 按用户与 IP 双维度检查一小时滚动窗口；任一维度超限则拒绝且不计入本次。
func (l *ideaVoteLimiter) allow(userID int64, ip string) bool {
	now := time.Now()
	cutoff := now.Add(-ideaVoteWindow)

	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(cutoff)

	userTimes := pruneIdeaVoteTimes(l.users[userID], cutoff)
	if len(userTimes) >= ideaVoteUserLimit {
		l.users[userID] = userTimes
		return false
	}
	ipTimes := pruneIdeaVoteTimes(l.ips[ip], cutoff)
	if len(ipTimes) >= ideaVoteIPLimit {
		l.users[userID] = userTimes
		l.ips[ip] = ipTimes
		return false
	}

	l.users[userID] = append(userTimes, now)
	l.ips[ip] = append(ipTimes, now)
	return true
}

// pruneIdeaVoteTimes 丢弃窗口外的旧时间戳，复用底层数组容量。
func pruneIdeaVoteTimes(times []time.Time, cutoff time.Time) []time.Time {
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	return kept
}

// sweep 惰性清理整窗过期的 map 条目，避免进程内 map 无界增长。
func (l *ideaVoteLimiter) sweep(cutoff time.Time) {
	if len(l.users) > 1024 {
		for k, times := range l.users {
			if len(times) == 0 || !times[len(times)-1].After(cutoff) {
				delete(l.users, k)
			}
		}
	}
	if len(l.ips) > 1024 {
		for k, times := range l.ips {
			if len(times) == 0 || !times[len(times)-1].After(cutoff) {
				delete(l.ips, k)
			}
		}
	}
}
