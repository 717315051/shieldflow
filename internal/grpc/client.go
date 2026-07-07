package grpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 是主控后端用来调用"gRPC"(REST) 接口的客户端。
// 通过 HTTP+JSON 向边缘节点的 REST 端点推送指令。
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient 创建客户端实例
// baseURL 示例: "http://node-1.shieldflow.internal:50051"
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// doRequest 发送 HTTP 请求
func (c *Client) doRequest(method, path string, body interface{}) (*Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Node-Token", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	return &result, nil
}

// PushConfigToNode 向指定节点推送域名配置
func (c *Client) PushConfigToNode(nodeID uint, domainID uint, config interface{}) (*Response, error) {
	req := PushDomainConfigRequest{
		NodeID:   nodeID,
		DomainID: domainID,
	}
	if data, err := json.Marshal(config); err == nil {
		req.Config = data
	}
	return c.doRequest("POST", "/grpc/v1/config/push", req)
}

// PushDDoSConfigToNode 向节点推送 DDoS 配置
func (c *Client) PushDDoSConfigToNode(nodeID uint, config interface{}) (*Response, error) {
	req := PushDDoSConfigRequest{
		NodeID: nodeID,
	}
	if data, err := json.Marshal(config); err == nil {
		req.Config = data
	}
	return c.doRequest("POST", "/grpc/v1/config/ddos", req)
}

// PurgeCacheOnNode 在指定节点上刷新缓存
func (c *Client) PurgeCacheOnNode(nodeID uint, domainID uint, urls []string) (*Response, error) {
	req := PurgeCacheRequest{
		NodeID:   nodeID,
		DomainID: domainID,
		URLs:     urls,
	}
	return c.doRequest("POST", "/grpc/v1/cache/purge", req)
}

// PreheatCacheOnNode 在指定节点上预热缓存
func (c *Client) PreheatCacheOnNode(nodeID uint, domainID uint, urls []string) (*Response, error) {
	req := PreheatCacheRequest{
		NodeID:   nodeID,
		DomainID: domainID,
		URLs:     urls,
	}
	return c.doRequest("POST", "/grpc/v1/cache/preheat", req)
}

// UpgradeNode 向节点下发升级指令（通过 config push 通道传递版本号）
func (c *Client) UpgradeNode(nodeID uint, version string) (*Response, error) {
	upgradeCfg := map[string]interface{}{
		"action":  "upgrade",
		"version": version,
	}
	return c.PushConfigToNode(nodeID, 0, upgradeCfg)
}

// SendHeartbeat 发送心跳（通常由边缘节点调用，这里提供反向调用能力）
func (c *Client) SendHeartbeat(nodeID uint, metrics interface{}) (*Response, error) {
	req := HeartbeatRequest{
		NodeID: nodeID,
	}
	if data, err := json.Marshal(metrics); err == nil {
		req.Metrics = data
	}
	return c.doRequest("POST", "/grpc/v1/node/heartbeat", req)
}

// GetNodeConfig 拉取节点配置
func (c *Client) GetNodeConfig(nodeID uint) (*Response, error) {
	return c.doRequest("GET", fmt.Sprintf("/grpc/v1/node/config/%d", nodeID), nil)
}

// VerifyLicense 验证授权码
func (c *Client) VerifyLicense(licenseKey string, nodeID uint, nodeIP string) (*Response, error) {
	req := VerifyLicenseRequest{
		LicenseKey: licenseKey,
		NodeID:     nodeID,
		NodeIP:     nodeIP,
	}
	return c.doRequest("POST", "/grpc/v1/auth/verify-license", req)
}

// ReportAccessLogs 上报访问日志
func (c *Client) ReportAccessLogs(batch AccessLogBatch) (*Response, error) {
	return c.doRequest("POST", "/grpc/v1/logs/access", batch)
}
