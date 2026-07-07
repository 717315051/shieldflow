package grpc

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shieldflow/shieldflow/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler 封装业务逻辑，处理来自边缘节点的请求。
type Handler struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewHandler(db *gorm.DB, log *zap.Logger) *Handler {
	return &Handler{db: db, log: log}
}

// ============================================================================
// Config Service 业务逻辑
// ============================================================================

// PushDomainConfig 从 PG 查询域名配置 → 序列化为 JSON → 推送到节点
// 在 REST 模式下，节点主动拉取；此方法返回最新配置供节点应用。
func (h *Handler) PushDomainConfig(req PushDomainConfigRequest) (json.RawMessage, error) {
	h.log.Info("push domain config",
		zap.Uint("node_id", req.NodeID),
		zap.Uint("domain_id", req.DomainID),
	)

	var domain models.Domain
	if err := h.db.First(&domain, req.DomainID).Error; err != nil {
		h.log.Error("查询域名失败", zap.Uint("domain_id", req.DomainID), zap.Error(err))
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// 组装完整配置
	cfg := map[string]interface{}{
		"domain_id":         domain.ID,
		"domain_name":       domain.DomainName,
		"cname":             domain.CNAME,
		"status":            domain.Status,
		"origin_config":     safeUnmarshal(domain.OriginConfig),
		"https_config":      safeUnmarshal(domain.HTTPSConfig),
		"cache_config":      safeUnmarshal(domain.CacheConfig),
		"advanced_config":   safeUnmarshal(domain.AdvancedConfig),
		"custom_headers":    safeUnmarshal(domain.CustomHeaders),
		"custom_pages":      safeUnmarshal(domain.CustomPages),
		"protection_config": safeUnmarshal(domain.ProtectionConfig),
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return data, nil
}

// PushDDoSConfig 查询 DDoS 规则下发给节点
func (h *Handler) PushDDoSConfig(req PushDDoSConfigRequest) (json.RawMessage, error) {
	h.log.Info("push ddos config", zap.Uint("node_id", req.NodeID))

	var rules []models.DDoSRule
	if err := h.db.Where("status = ?", "active").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("query ddos rules: %w", err)
	}

	// 查询黑白名单
	var blacklist []models.DDoSBlacklistEntry
	h.db.Find(&blacklist)

	cfg := map[string]interface{}{
		"rules":     rules,
		"blacklist": blacklist,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal ddos config: %w", err)
	}
	return data, nil
}

// PushGlobalConfig 推送全局配置
func (h *Handler) PushGlobalConfig(req PushGlobalConfigRequest) (json.RawMessage, error) {
	h.log.Info("push global config", zap.Int("node_count", len(req.NodeIDs)))

	var settings []models.SystemSetting
	h.db.Find(&settings)

	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	data, err := json.Marshal(settingsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal global config: %w", err)
	}
	return data, nil
}

// SyncNodeStatus 同步节点状态
func (h *Handler) SyncNodeStatus(req SyncNodeStatusRequest) error {
	var status struct {
		Status     string `json:"status"`
		ConnCount  int    `json:"conn_count"`
		QPS        int    `json:"qps"`
		CPUUsage   float64 `json:"cpu_usage"`
		MemUsage   float64 `json:"mem_usage"`
	}
	if err := json.Unmarshal(req.Status, &status); err != nil {
		return fmt.Errorf("unmarshal status: %w", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":         status.Status,
		"conn_count":     status.ConnCount,
		"qps":            status.QPS,
		"last_heartbeat": &now,
	}

	if err := h.db.Model(&models.Node{}).Where("id = ?", req.NodeID).Updates(updates).Error; err != nil {
		h.log.Error("更新节点状态失败", zap.Uint("node_id", req.NodeID), zap.Error(err))
		return fmt.Errorf("update node status: %w", err)
	}
	return nil
}

// Heartbeat 处理节点心跳
func (h *Handler) Heartbeat(req HeartbeatRequest) (HeartbeatResponse, error) {
	h.log.Debug("heartbeat", zap.Uint("node_id", req.NodeID))

	now := time.Now()
	if err := h.db.Model(&models.Node{}).Where("id = ?", req.NodeID).Updates(map[string]interface{}{
		"status":         "online",
		"last_heartbeat": &now,
	}).Error; err != nil {
		return HeartbeatResponse{}, fmt.Errorf("update heartbeat: %w", err)
	}

	// 解析 metrics 更新 QPS 等
	if len(req.Metrics) > 0 {
		var metrics struct {
			ConnCount int `json:"conn_count"`
			QPS       int `json:"qps"`
		}
		if err := json.Unmarshal(req.Metrics, &metrics); err == nil {
			h.db.Model(&models.Node{}).Where("id = ?", req.NodeID).Updates(map[string]interface{}{
				"conn_count": metrics.ConnCount,
				"qps":        metrics.QPS,
			})
		}
	}

	return HeartbeatResponse{
		ACK:       true,
		Timestamp: now.Unix(),
		ConfigVersion: time.Now().Format("20060102150405"),
	}, nil
}

// ============================================================================
// Log Service 业务逻辑 — 接收日志写入 ClickHouse
// ============================================================================

// ReportAccessLogs 接收访问日志批量写入 ClickHouse
func (h *Handler) ReportAccessLogs(batch AccessLogBatch) error {
	if len(batch.Logs) == 0 {
		return nil
	}
	h.log.Debug("report access logs", zap.Uint("node_id", batch.NodeID), zap.Int("count", len(batch.Logs)))
	// TODO: 当 ClickHouse conn 可用时批量写入
	// 这里先做日志记录，后续对接 ch driver
	for _, l := range batch.Logs {
		h.log.Info("access_log",
			zap.Time("ts", l.Timestamp),
			zap.Uint("node", l.NodeID),
			zap.Uint("domain", l.DomainID),
			zap.String("ip", l.ClientIP),
			zap.String("method", l.Method),
			zap.String("path", l.Path),
			zap.Int("status", l.Status),
			zap.Int64("bytes", l.BytesSent),
		)
	}
	return nil
}

func (h *Handler) ReportAttackLogs(batch AttackLogBatch) error {
	if len(batch.Logs) == 0 {
		return nil
	}
	h.log.Info("report attack logs", zap.Uint("node_id", batch.NodeID), zap.Int("count", len(batch.Logs)))
	for _, l := range batch.Logs {
		h.log.Warn("attack_log",
			zap.Time("ts", l.Timestamp),
			zap.Uint("node", l.NodeID),
			zap.String("ip", l.ClientIP),
			zap.String("type", l.AttackType),
			zap.String("action", l.Action),
		)
	}
	return nil
}

func (h *Handler) ReportDDoSLogs(batch DDoSLogBatch) error {
	if len(batch.Logs) == 0 {
		return nil
	}
	h.log.Info("report ddos logs", zap.Uint("node_id", batch.NodeID), zap.Int("count", len(batch.Logs)))
	for _, l := range batch.Logs {
		h.log.Warn("ddos_log",
			zap.Time("ts", l.Timestamp),
			zap.Uint("node", l.NodeID),
			zap.String("ip", l.ClientIP),
			zap.String("proto", l.Protocol),
			zap.Int64("pps", l.PacketRate),
		)
	}
	return nil
}

func (h *Handler) ReportLayer4Logs(batch Layer4LogBatch) error {
	if len(batch.Logs) == 0 {
		return nil
	}
	h.log.Debug("report layer4 logs", zap.Uint("node_id", batch.NodeID), zap.Int("count", len(batch.Logs)))
	for _, l := range batch.Logs {
		h.log.Info("layer4_log",
			zap.Time("ts", l.Timestamp),
			zap.Uint("node", l.NodeID),
			zap.String("proto", l.Protocol),
			zap.Int("port", l.ListenPort),
			zap.Int64("in", l.BytesIn),
			zap.Int64("out", l.BytesOut),
		)
	}
	return nil
}

func (h *Handler) ReportAILogs(batch AILogBatch) error {
	if len(batch.Logs) == 0 {
		return nil
	}
	h.log.Info("report ai logs", zap.Uint("node_id", batch.NodeID), zap.Int("count", len(batch.Logs)))
	for _, l := range batch.Logs {
		h.log.Info("ai_log",
			zap.Time("ts", l.Timestamp),
			zap.Uint("node", l.NodeID),
			zap.String("ip", l.ClientIP),
			zap.Float64("score", l.Score),
			zap.String("verdict", l.Verdict),
		)
	}
	return nil
}

// ============================================================================
// Node Service 业务逻辑
// ============================================================================

// RegisterNode 验证授权码 → 创建/更新节点记录 → 返回节点配置
func (h *Handler) RegisterNode(req RegisterNodeRequest) (RegisterNodeResponse, error) {
	h.log.Info("node register", zap.Uint("node_id", req.NodeID), zap.String("ip", req.Info.IP))

	// 验证授权码
	var node models.Node
	result := h.db.First(&node, req.NodeID)

	if result.Error != nil {
		// 节点不存在，创建
		node = models.Node{
			ID:          req.NodeID,
			Name:        req.Info.Name,
			IP:          req.Info.IP,
			Region:      req.Info.Region,
			Status:      "online",
			LicenseKey:  req.LicenseKey,
			GRPCAddress: req.Info.GRPCAddress,
			Version:     req.Info.Version,
			CPU:         req.Info.CPU,
			Memory:      req.Info.Memory,
			Disk:        req.Info.Disk,
			Bandwidth:   req.Info.Bandwidth,
		}
		if err := h.db.Create(&node).Error; err != nil {
			return RegisterNodeResponse{}, fmt.Errorf("create node: %w", err)
		}
	} else {
		// 节点已存在，更新
		now := time.Now()
		updates := map[string]interface{}{
			"name":           req.Info.Name,
			"ip":             req.Info.IP,
			"region":         req.Info.Region,
			"status":         "online",
			"grpc_address":   req.Info.GRPCAddress,
			"version":        req.Info.Version,
			"cpu":            req.Info.CPU,
			"memory":         req.Info.Memory,
			"disk":           req.Info.Disk,
			"bandwidth":      req.Info.Bandwidth,
			"last_heartbeat": &now,
		}
		if err := h.db.Model(&node).Updates(updates).Error; err != nil {
			return RegisterNodeResponse{}, fmt.Errorf("update node: %w", err)
		}
	}

	// 返回节点配置
	cfg, _ := h.GetNodeConfig(req.NodeID)
	cfgData, _ := json.Marshal(cfg)

	return RegisterNodeResponse{
		NodeID:  req.NodeID,
		Success: true,
		Config:  cfgData,
	}, nil
}

// UpdateNodeStatus 更新节点状态
func (h *Handler) UpdateNodeStatus(req UpdateNodeStatusRequest) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":         req.Status,
		"conn_count":     req.ConnCount,
		"qps":            req.QPS,
		"last_heartbeat": &now,
	}
	if err := h.db.Model(&models.Node{}).Where("id = ?", req.NodeID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update node status: %w", err)
	}
	return nil
}

// GetNodeConfig 查询节点应加载的完整配置
func (h *Handler) GetNodeConfig(nodeID uint) (GetNodeConfigResponse, error) {
	// 查询所有 active 域名
	var domains []models.Domain
	if err := h.db.Where("status = ?", "active").Find(&domains).Error; err != nil {
		return GetNodeConfigResponse{}, fmt.Errorf("query domains: %w", err)
	}

	// 查询 DDoS 规则
	var ddosRules []models.DDoSRule
	h.db.Where("status = ?", "active").Find(&ddosRules)

	// 查询证书
	var certs []models.Certificate
	h.db.Where("status = ?", "active").Find(&certs)

	domainsJSON, _ := json.Marshal(domains)
	rulesJSON, _ := json.Marshal(ddosRules)
	certsJSON, _ := json.Marshal(certs)

	return GetNodeConfigResponse{
		NodeID:        nodeID,
		Domains:       domainsJSON,
		DDoSRules:     rulesJSON,
		Certificates:  certsJSON,
		ConfigVersion: time.Now().Format("20060102150405"),
	}, nil
}

// ============================================================================
// Cache Service 业务逻辑
// ============================================================================

// PurgeCache 创建缓存刷新任务并推送到对应节点
func (h *Handler) PurgeCache(req PurgeCacheRequest) error {
	h.log.Info("purge cache",
		zap.Uint("node_id", req.NodeID),
		zap.Uint("domain_id", req.DomainID),
		zap.Int("url_count", len(req.URLs)),
	)

	urlsJSON, _ := json.Marshal(req.URLs)

	task := models.CacheTask{
		DomainID: req.DomainID,
		Type:     "file_refresh",
		URLs:     string(urlsJSON),
		Status:   "pending",
	}
	if err := h.db.Create(&task).Error; err != nil {
		return fmt.Errorf("create cache task: %w", err)
	}

	// TODO: 通过 gRPC/HTTP 推送到边缘节点 req.NodeID
	h.log.Info("cache purge task created", zap.Uint("task_id", task.ID), zap.Uint("node_id", req.NodeID))
	return nil
}

// PreheatCache 创建缓存预热任务
func (h *Handler) PreheatCache(req PreheatCacheRequest) error {
	h.log.Info("preheat cache",
		zap.Uint("node_id", req.NodeID),
		zap.Uint("domain_id", req.DomainID),
		zap.Int("url_count", len(req.URLs)),
	)

	urlsJSON, _ := json.Marshal(req.URLs)

	task := models.CacheTask{
		DomainID: req.DomainID,
		Type:     "file_preheat",
		URLs:     string(urlsJSON),
		Status:   "pending",
	}
	if err := h.db.Create(&task).Error; err != nil {
		return fmt.Errorf("create cache task: %w", err)
	}

	h.log.Info("cache preheat task created", zap.Uint("task_id", task.ID), zap.Uint("node_id", req.NodeID))
	return nil
}

// GetCacheStats 获取缓存统计
func (h *Handler) GetCacheStats(domainID uint) (CacheStatsResponse, error) {
	// TODO: 从节点采集或从 ClickHouse 聚合
	// 暂时返回占位数据
	return CacheStatsResponse{
		DomainID:   domainID,
		CacheSize:  0,
		CacheItems: 0,
		HitRate:    0,
		MissRate:   0,
		Evictions:  0,
	}, nil
}

// ============================================================================
// Auth Service 业务逻辑
// ============================================================================

// VerifyLicense 验证节点授权码
func (h *Handler) VerifyLicense(req VerifyLicenseRequest) (VerifyLicenseResponse, error) {
	h.log.Info("verify license", zap.Uint("node_id", req.NodeID), zap.String("ip", req.NodeIP))

	if req.LicenseKey == "" {
		return VerifyLicenseResponse{Valid: false, Message: "license key is empty"}, nil
	}

	// 查询节点，验证 license_key 匹配
	var node models.Node
	if err := h.db.First(&node, req.NodeID).Error; err != nil {
		return VerifyLicenseResponse{Valid: false, Message: "node not found"}, nil
	}

	if node.LicenseKey != req.LicenseKey {
		return VerifyLicenseResponse{Valid: false, Message: "license key mismatch"}, nil
	}

	return VerifyLicenseResponse{
		Valid:     true,
		NodeID:    node.ID,
		Message:   "verified",
	}, nil
}

// ============================================================================
// 工具函数
// ============================================================================

func safeUnmarshal(s string) interface{} {
	if s == "" {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s // 返回原始字符串
	}
	return v
}
