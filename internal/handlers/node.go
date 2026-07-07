package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// NodeHandler 节点管理 handler 集合
type NodeHandler struct{}

// NewNodeHandler 构造 NodeHandler
func NewNodeHandler() *NodeHandler {
	return &NodeHandler{}
}

// ListNodes GET /api/v1/admin/nodes
func (h *NodeHandler) ListNodes(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	status := strings.TrimSpace(c.Query("status"))
	region := strings.TrimSpace(c.Query("region"))
	groupID := strings.TrimSpace(c.Query("group_id"))

	query := db.Model(&models.Node{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ?", like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if region != "" {
		query = query.Where("region = ?", region)
	}
	if groupID != "" {
		query = query.Where("group_id = ?", groupID)
	}

	var total int64
	query.Count(&total)

	var nodes []models.Node
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "查询失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      nodes,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// CreateNode POST /api/v1/admin/nodes
func (h *NodeHandler) CreateNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Name        string `json:"name"`
		IP          string `json:"ip"`
		Region      string `json:"region"`
		GroupID     *uint  `json:"group_id"`
		GRPCAddress string `json:"grpc_address"`
		CPU         int    `json:"cpu"`
		Memory      int    `json:"memory"`
		Disk        int    `json:"disk"`
		Bandwidth   int    `json:"bandwidth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if req.Name == "" || req.IP == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "节点名称和 IP 不能为空", "data": nil})
		return
	}

	var cnt int64
	db.Model(&models.Node{}).Where("ip = ?", req.IP).Count(&cnt)
	if cnt > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 409, "message": "IP 已存在", "data": nil})
		return
	}

	// 生成 LicenseKey
	licenseKey := generateLicenseKey()

	node := models.Node{
		Name:        req.Name,
		IP:          req.IP,
		Region:      req.Region,
		GroupID:     req.GroupID,
		Status:      "offline",
		LicenseKey:  licenseKey,
		GRPCAddress: req.GRPCAddress,
		CPU:         req.CPU,
		Memory:      req.Memory,
		Disk:        req.Disk,
		Bandwidth:   req.Bandwidth,
	}
	if err := db.Create(&node).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "创建节点失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": node})
}

// DeleteNode DELETE /api/v1/admin/nodes/:id
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}
	if err := db.Delete(&models.Node{}, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "删除失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// GetNode GET /api/v1/admin/nodes/:id
func (h *NodeHandler) GetNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "节点不存在", "data": nil})
		return
	}
	// 附带分组信息
	var group models.NodeGroup
	if node.GroupID != nil {
		db.First(&group, *node.GroupID)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"node":  node,
			"group": group,
		},
	})
}

// UpdateNode PUT /api/v1/admin/nodes/:id
func (h *NodeHandler) UpdateNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}

	var req struct {
		Name        string `json:"name"`
		IP          string `json:"ip"`
		Region      string `json:"region"`
		GroupID     *uint  `json:"group_id"`
		GRPCAddress string `json:"grpc_address"`
		CPU         int    `json:"cpu"`
		Memory      int    `json:"memory"`
		Disk        int    `json:"disk"`
		Bandwidth   int    `json:"bandwidth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.IP != "" {
		updates["ip"] = req.IP
	}
	if req.Region != "" {
		updates["region"] = req.Region
	}
	updates["group_id"] = req.GroupID
	if req.GRPCAddress != "" {
		updates["grpc_address"] = req.GRPCAddress
	}
	updates["cpu"] = req.CPU
	updates["memory"] = req.Memory
	updates["disk"] = req.Disk
	updates["bandwidth"] = req.Bandwidth

	if err := db.Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// InstallNode POST /api/v1/admin/nodes/:id/install
func (h *NodeHandler) InstallNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "节点不存在", "data": nil})
		return
	}

	// 构造一键安装命令（示例）
	installCmd := fmt.Sprintf(
		`curl -fsSL https://install.shieldflow.net/node.sh | bash -s -- --license %s --grpc %s`,
		node.LicenseKey,
		c.GetString("grpc_address"),
	)
	if node.GRPCAddress != "" {
		installCmd = fmt.Sprintf(
			`curl -fsSL https://install.shieldflow.net/node.sh | bash -s -- --license %s --grpc %s`,
			node.LicenseKey,
			node.GRPCAddress,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"install_command": installCmd,
			"license_key":     node.LicenseKey,
		},
	})
}

// SSHInstallNode POST /api/v1/admin/nodes/:id/ssh-install
func (h *NodeHandler) SSHInstallNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "节点不存在", "data": nil})
		return
	}

	var req struct {
		SSHPort   int    `json:"ssh_port"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		PrivateKey string `json:"private_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if req.Username == "" || (req.Password == "" && req.PrivateKey == "") {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "用户名和密码/私钥不能为空", "data": nil})
		return
	}

	// TODO: 通过 golang.org/x/crypto/ssh 实际连接节点并执行安装命令
	// 此处先返回任务标识，实际安装异步执行
	taskID := fmt.Sprintf("ssh-install-%d-%d", node.ID, time.Now().UnixNano())
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"task_id":  taskID,
			"node_id":  node.ID,
			"node_ip":  node.IP,
			"status":   "running",
		},
	})
}

// UpgradeNode POST /api/v1/admin/nodes/:id/upgrade
func (h *NodeHandler) UpgradeNode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "节点不存在", "data": nil})
		return
	}
	var req struct {
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 兼容无 body 调用
		req.Version = ""
	}

	targetVersion := req.Version
	if targetVersion == "" {
		targetVersion = "latest"
	}

	// TODO: 通过 gRPC 调用节点 Upgrade 接口
	// 此处仅返回任务标识
	taskID := fmt.Sprintf("upgrade-%d-%d", node.ID, time.Now().UnixNano())
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"task_id":  taskID,
			"node_id":  node.ID,
			"version":  targetVersion,
			"status":   "running",
		},
	})
}

// BatchUpgradeNodes POST /api/v1/admin/nodes/batch-upgrade
func (h *NodeHandler) BatchUpgradeNodes(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		NodeIDs []uint `json:"node_ids"`
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if len(req.NodeIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "node_ids 不能为空", "data": nil})
		return
	}
	targetVersion := req.Version
	if targetVersion == "" {
		targetVersion = "latest"
	}

	successCount := 0
	var errs []string
	for _, nid := range req.NodeIDs {
		var node models.Node
		if err := db.First(&node, nid).Error; err != nil {
			errs = append(errs, fmt.Sprintf("节点 ID %d 不存在", nid))
			continue
		}
		// TODO: 通过 gRPC 调用节点 Upgrade 接口
		successCount++
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"success_count": successCount,
			"errors":        errs,
			"version":       targetVersion,
		},
	})
}

// GetNodeStatus GET /api/v1/admin/nodes/:id/status
func (h *NodeHandler) GetNodeStatus(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的节点 ID", "data": nil})
		return
	}
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "节点不存在", "data": nil})
		return
	}

	// 判断心跳是否过期
	isOnline := node.Status == "online"
	if node.LastHeartbeat != nil {
		// 心跳超过 60s 视为离线
		if time.Since(*node.LastHeartbeat) > 60*time.Second {
			isOnline = false
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"node_id":        node.ID,
			"status":         node.Status,
			"online":         isOnline,
			"version":        node.Version,
			"cpu":            node.CPU,
			"memory":         node.Memory,
			"disk":           node.Disk,
			"bandwidth":      node.Bandwidth,
			"conn_count":     node.ConnCount,
			"qps":            node.QPS,
			"last_heartbeat": node.LastHeartbeat,
		},
	})
}

// --- 节点分组 ---

// ListNodeGroups GET /api/v1/admin/node-groups
func (h *NodeHandler) ListNodeGroups(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var groups []models.NodeGroup
	if err := db.Order("id DESC").Find(&groups).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "查询失败: " + err.Error(), "data": nil})
		return
	}

	// 附带每个分组下的节点数量
	type GroupWithCount struct {
		models.NodeGroup
		NodeCount int64 `json:"node_count"`
	}
	result := make([]GroupWithCount, 0, len(groups))
	for _, g := range groups {
		var cnt int64
		db.Model(&models.Node{}).Where("group_id = ?", g.ID).Count(&cnt)
		result = append(result, GroupWithCount{
			NodeGroup: g,
			NodeCount: cnt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// CreateNodeGroup POST /api/v1/admin/node-groups
func (h *NodeHandler) CreateNodeGroup(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "分组名称不能为空", "data": nil})
		return
	}

	var cnt int64
	db.Model(&models.NodeGroup{}).Where("name = ?", req.Name).Count(&cnt)
	if cnt > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 409, "message": "分组名称已存在", "data": nil})
		return
	}

	group := models.NodeGroup{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := db.Create(&group).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "创建失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": group})
}

// UpdateNodeGroup PUT /api/v1/admin/node-groups/:id
func (h *NodeHandler) UpdateNodeGroup(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的分组 ID", "data": nil})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["description"] = req.Description

	if err := db.Model(&models.NodeGroup{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// DeleteNodeGroup DELETE /api/v1/admin/node-groups/:id
func (h *NodeHandler) DeleteNodeGroup(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的分组 ID", "data": nil})
		return
	}

	// 解除分组下节点的关联
	db.Model(&models.Node{}).Where("group_id = ?", id).Update("group_id", nil)

	if err := db.Delete(&models.NodeGroup{}, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "删除失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// --- 辅助函数 ---

// generateLicenseKey 生成节点 LicenseKey
func generateLicenseKey() string {
	// 简易生成: 时间戳 + 随机字符 (复用 auth.go 中 rand)
	// 为避免与 auth.go 冲突，使用独立实现
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := newRand()
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return fmt.Sprintf("zy-%s-%d", string(b), time.Now().Unix())
}

// newRand 返回一个新的 rand.Rand (此处使用 math/rand 标准库)
func newRand() *randReader {
	return &randReader{}
}

// randReader 封装 math/rand 以避免和 auth.go 中 rand 冲突
type randReader struct{}

func (r *randReader) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	// 使用标准 math/rand 全局函数
	return rand.Intn(n)
}

// 避免 json 未使用告警
var _ = json.Marshal
