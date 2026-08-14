package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// ============================================================
// 可观测性：Prometheus 文本格式 /metrics 端点
//
// 提供网关自身的运行指标（不含任何密钥/敏感信息），可被
// Prometheus 抓取或人工 curl 查看：
//   - 请求总数 / 进行中请求数（gauge）
//   - 上游错误总数
//   - 首 token 延迟（summary：sum/count）
//   - 输入/输出 token 总量
//   - 缓存命中/未命中 token 总量
// ============================================================

var (
	metricReqsTotal      atomic.Uint64 // 网关接收的代理请求总数
	metricReqsInFlight   atomic.Int64  // 当前进行中请求数（gauge）
	metricUpstreamErrors atomic.Uint64 // 上游请求失败/错误总数
	metricTTFTSumMs      atomic.Uint64 // 首 token 延迟汇总（毫秒）
	metricTTFTCount      atomic.Uint64 // 有 TTFT 记录的请求数
	metricInputTokens    atomic.Uint64 // 输入 token 总量
	metricOutputTokens   atomic.Uint64 // 输出 token 总量
	metricCacheHit       atomic.Uint64 // 缓存命中输入 token 总量
	metricCacheMiss      atomic.Uint64 // 缓存未命中输入 token 总量
)

// metricsHandler 输出 Prometheus 文本格式指标
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP aimodels_requests_total 网关接收的代理请求总数\n")
	fmt.Fprintf(w, "# TYPE aimodels_requests_total counter\n")
	fmt.Fprintf(w, "aimodels_requests_total %d\n", metricReqsTotal.Load())
	fmt.Fprintf(w, "# HELP aimodels_requests_in_flight 当前进行中的请求数\n")
	fmt.Fprintf(w, "# TYPE aimodels_requests_in_flight gauge\n")
	fmt.Fprintf(w, "aimodels_requests_in_flight %d\n", metricReqsInFlight.Load())
	fmt.Fprintf(w, "# HELP aimodels_upstream_errors_total 上游请求失败/错误总数\n")
	fmt.Fprintf(w, "# TYPE aimodels_upstream_errors_total counter\n")
	fmt.Fprintf(w, "aimodels_upstream_errors_total %d\n", metricUpstreamErrors.Load())
	fmt.Fprintf(w, "# HELP aimodels_ttft_seconds 首 token 延迟（秒）\n")
	fmt.Fprintf(w, "# TYPE aimodels_ttft_seconds summary\n")
	fmt.Fprintf(w, "aimodels_ttft_seconds_sum %f\n", float64(metricTTFTSumMs.Load())/1000.0)
	fmt.Fprintf(w, "aimodels_ttft_seconds_count %d\n", metricTTFTCount.Load())
	fmt.Fprintf(w, "# HELP aimodels_input_tokens_total 输入 token 总量\n")
	fmt.Fprintf(w, "# TYPE aimodels_input_tokens_total counter\n")
	fmt.Fprintf(w, "aimodels_input_tokens_total %d\n", metricInputTokens.Load())
	fmt.Fprintf(w, "# HELP aimodels_output_tokens_total 输出 token 总量\n")
	fmt.Fprintf(w, "# TYPE aimodels_output_tokens_total counter\n")
	fmt.Fprintf(w, "aimodels_output_tokens_total %d\n", metricOutputTokens.Load())
	fmt.Fprintf(w, "# HELP aimodels_cache_hit_total 缓存命中输入 token 总量\n")
	fmt.Fprintf(w, "# TYPE aimodels_cache_hit_total counter\n")
	fmt.Fprintf(w, "aimodels_cache_hit_total %d\n", metricCacheHit.Load())
	fmt.Fprintf(w, "# HELP aimodels_cache_miss_total 缓存未命中输入 token 总量\n")
	fmt.Fprintf(w, "# TYPE aimodels_cache_miss_total counter\n")
	fmt.Fprintf(w, "aimodels_cache_miss_total %d\n", metricCacheMiss.Load())
}
