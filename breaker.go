package main

import (
	"log"
	"sync"
	"time"
)

// ============================================================
// 主备路由熔断器（健康记忆）
//
// 目标：主站故障时，不要让每个请求都白等一次失败/超时（默认 60s）。
// 某个 provider 连续失败达到阈值后进入冷却期，期间直接跳过该站点、
// 顺延到下一个可用站点；冷却期结束后放行一次探测请求。
// ============================================================

const (
	breakerFailThreshold = 3              // 连续失败次数达到该值触发熔断
	breakerCooldown      = 30 * time.Second // 熔断冷却期
)

type breakerState struct {
	consecutiveFails int
	openUntil        time.Time // 熔断截止时间；零值表示未熔断
}

// failoverBreaker 全局熔断器：providerID -> 状态
var failoverBreaker = &breaker{
	states: make(map[string]*breakerState),
}

type breaker struct {
	mu     sync.Mutex
	states map[string]*breakerState
}

// allow 判断是否允许请求该 provider（false = 熔断中，跳过）
func (b *breaker) allow(providerID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.states[providerID]
	if s == nil || s.openUntil.IsZero() {
		return true
	}
	if time.Now().After(s.openUntil) {
		// 冷却期结束：半开，重置计数并放行一次探测
		s.openUntil = time.Time{}
		s.consecutiveFails = 0
		log.Printf("[breaker] %s 冷却期结束，恢复探测", providerID)
		return true
	}
	return false
}

// recordFailure 记录一次失败；连续失败达到阈值则熔断
func (b *breaker) recordFailure(providerID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.states[providerID]
	if s == nil {
		s = &breakerState{}
		b.states[providerID] = s
	}
	s.consecutiveFails++
	if s.consecutiveFails >= breakerFailThreshold {
		s.openUntil = time.Now().Add(breakerCooldown)
		log.Printf("[breaker] %s 连续失败 %d 次，熔断 %s", providerID, s.consecutiveFails, breakerCooldown)
	}
}

// recordSuccess 记录一次成功，重置熔断状态
func (b *breaker) recordSuccess(providerID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.states[providerID]
	if s == nil {
		return
	}
	if s.consecutiveFails > 0 || !s.openUntil.IsZero() {
		log.Printf("[breaker] %s 恢复成功，重置熔断状态", providerID)
	}
	s.consecutiveFails = 0
	s.openUntil = time.Time{}
}
