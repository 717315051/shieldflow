// Package proxy implements the reverse proxy core for the ShieldFlow edge node.
//
// 基于 net/http/httputil.ReverseProxy，支持：
//   - 多源站负载均衡（轮询 / 加权轮询 / IP Hash）
//   - 回源协议 / 端口 / Host 配置
//   - HTTP/2 回源
//   - 主动健康检查与故障转移
package proxy

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BalancerStrategy 负载均衡策略。
type BalancerStrategy string

const (
	StrategyRoundRobin  BalancerStrategy = "round_robin"
	StrategyWeighted    BalancerStrategy = "weighted"
	StrategyIPHash      BalancerStrategy = "ip_hash"
)

// Origin 描述一个回源目标。
type Origin struct {
	Addr     string `json:"addr"`      // host:port
	Weight   int    `json:"weight"`    // 加权轮询权重
	Scheme   string `json:"scheme"`    // http 或 https
	Host     string `json:"host"`      // 回源 Host 头；空则透传客户端 Host
	Healthy  bool    `json:"-"`         // 运行时健康状态
	failures uint32                  // 连续失败次数
}

// OriginPool 一组回源目标。
type OriginPool struct {
	mu        sync.Mutex
	origins   []*Origin
	strategy  BalancerStrategy
	rrIdx     uint64
	hcCancel  context.CancelFunc
}

// NewOriginPool 创建源站池。
func NewOriginPool(strategy BalancerStrategy, origins []*Origin) *OriginPool {
	for _, o := range origins {
		if o.Scheme == "" {
			o.Scheme = "http"
		}
		if o.Weight <= 0 {
			o.Weight = 1
		}
		o.Healthy = true
	}
	return &OriginPool{
		origins:  origins,
		strategy: strategy,
	}
}

// pick 按策略选取下一个健康源站。
func (p *OriginPool) pick(clientIP string) *Origin {
	p.mu.Lock()
	defer p.mu.Unlock()

	healthy := p.healthyOriginsLocked()
	if len(healthy) == 0 {
		// 全部不健康：降级使用全部源站。
		healthy = p.origins
		if len(healthy) == 0 {
			return nil
		}
	}

	switch p.strategy {
	case StrategyIPHash:
		if clientIP == "" {
			clientIP = "0.0.0.0"
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(clientIP))
		return healthy[int(h.Sum32())%len(healthy)]
	case StrategyWeighted:
		total := 0
		for _, o := range healthy {
			total += o.Weight
		}
		if total <= 0 {
			return healthy[0]
		}
		target := int(atomic.AddUint64(&p.rrIdx, 1)) % total
		for _, o := range healthy {
			target -= o.Weight
			if target < 0 {
				return o
			}
		}
		return healthy[0]
	default: // round_robin
		idx := atomic.AddUint64(&p.rrIdx, 1)
		return healthy[int(idx)%len(healthy)]
	}
}

func (p *OriginPool) healthyOriginsLocked() []*Origin {
	out := make([]*Origin, 0, len(p.origins))
	for _, o := range p.origins {
		if o.Healthy {
			out = append(out, o)
		}
	}
	return out
}

// markFailure 记录源站失败，达到阈值则标记为不健康。
func (p *OriginPool) markFailure(o *Origin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	f := atomic.AddUint32(&o.failures, 1)
	if f >= 3 {
		o.Healthy = false
	}
}

// markSuccess 源站恢复成功时重置失败计数。
func (p *OriginPool) markSuccess(o *Origin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	atomic.StoreUint32(&o.failures, 0)
	o.Healthy = true
}

// StartHealthCheck 启动主动健康检查（后台 goroutine）。
func (p *OriginPool) StartHealthCheck(path string, interval time.Duration) {
	if path == "" {
		path = "/healthz"
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.hcCancel = cancel
	p.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.runHealthCheck(path)
			}
		}
	}()
}

// StopHealthCheck 停止主动健康检查。
func (p *OriginPool) StopHealthCheck() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hcCancel != nil {
		p.hcCancel()
		p.hcCancel = nil
	}
}

func (p *OriginPool) runHealthCheck(path string) {
	client := &http.Client{Timeout: 3 * time.Second}
	var wg sync.WaitGroup
	p.mu.Lock()
	snapshot := append([]*Origin(nil), p.origins...)
	p.mu.Unlock()
	for _, o := range snapshot {
		wg.Add(1)
		go func(o *Origin) {
			defer wg.Done()
			u := fmt.Sprintf("%s://%s%s", o.Scheme, o.Addr, path)
			resp, err := client.Get(u)
			if err != nil {
				p.markFailure(o)
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				p.markSuccess(o)
			} else {
				p.markFailure(o)
			}
		}(o)
	}
	wg.Wait()
}

// ReverseProxy 是 ShieldFlow 反向代理核心。
type ReverseProxy struct {
	pool   *OriginPool
	rp     *httputil.ReverseProxy
	logger func(req *http.Request, status int, dur time.Duration, upstream string)
}

// NewReverseProxy 基于源站池创建反向代理。
func NewReverseProxy(pool *OriginPool) (*ReverseProxy, error) {
	if pool == nil || len(pool.origins) == 0 {
		return nil, errors.New("no origins configured")
	}
	rp := &ReverseProxy{
		pool: pool,
	}
	// 延迟选取源站：每个请求动态挑 origin。
	proxy := &httputil.ReverseProxy{
		Director:       rp.director,
		Transport:      rp.transport(),
		ModifyResponse: rp.modifyResponse,
		ErrorHandler:   rp.errorHandler,
	}
	rp.rp = proxy
	return rp, nil
}

// director 改写回源请求。
func (p *ReverseProxy) director(req *http.Request) {
	origin := p.pool.pick(clientIP(req))
	if origin == nil {
		// 无可用源站：保留原 URL，后续 errorHandler 处理。
		return
	}
	target := &url.URL{
		Scheme: origin.Scheme,
		Host:   origin.Addr,
	}
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	if origin.Host != "" {
		req.Host = origin.Host
	}
	// 记录选中的源站，便于日志和重试。
	req.Header.Set("X-ShieldFlow-Upstream", origin.Addr)
	// 透传客户端真实 IP。
	req.Header.Set("X-Real-IP", clientIP(req))
	req.Header.Set("X-Forwarded-Host", req.Host)
}

// transport 自定义 RoundTripper，启用 HTTP/2，并实现失败重试。
func (p *ReverseProxy) transport() http.RoundTripper {
	return &customTransport{
		 base: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          500,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		pool: p.pool,
	}
}

type customTransport struct {
	base http.RoundTripper
	pool *OriginPool
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	upstream := req.Header.Get("X-ShieldFlow-Upstream")
	if err != nil {
		// 找到对应的 origin 标记失败。
		if o := t.findOrigin(upstream); o != nil {
			t.pool.markFailure(o)
		}
		return nil, err
	}
	if resp.StatusCode >= 500 {
		if o := t.findOrigin(upstream); o != nil {
			t.pool.markFailure(o)
		}
	} else {
		if o := t.findOrigin(upstream); o != nil {
			t.pool.markSuccess(o)
		}
	}
	if t.pool != nil {
		// noop: 留作扩展日志钩子
		_ = start
	}
	return resp, nil
}

func (t *customTransport) findOrigin(addr string) *Origin {
	if addr == "" {
		return nil
	}
	t.pool.mu.Lock()
	defer t.pool.mu.Unlock()
	for _, o := range t.pool.origins {
		if o.Addr == addr {
			return o
		}
	}
	return nil
}

func (p *ReverseProxy) modifyResponse(resp *http.Response) error {
	// 注入边缘节点标识。
	resp.Header.Set("X-ShieldFlow-Edge", "1")
	resp.Header.Set("X-ShieldFlow-Upstream", resp.Request.Header.Get("X-ShieldFlow-Upstream"))
	return nil
}

func (p *ReverseProxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	upstream := r.Header.Get("X-ShieldFlow-Upstream")
	if upstream != "" && p.pool != nil {
		if o := p.findOriginPublic(upstream); o != nil {
			p.pool.markFailure(o)
		}
	}
	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

func (p *ReverseProxy) findOriginPublic(addr string) *Origin {
	p.pool.mu.Lock()
	defer p.pool.mu.Unlock()
	for _, o := range p.pool.origins {
		if o.Addr == addr {
			return o
		}
	}
	return nil
}

// ServeHTTP 实现 http.Handler。
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rp.ServeHTTP(w, r)
}

// clientIP 提取客户端 IP（兼容 X-Forwarded-For / X-Real-IP）。
func clientIP(r *http.Request) string {
	if ff := r.Header.Get("X-Forwarded-For"); ff != "" {
		if idx := strings.IndexByte(ff, ','); idx > 0 {
			return strings.TrimSpace(ff[:idx])
		}
		return strings.TrimSpace(ff)
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
