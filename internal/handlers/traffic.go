package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============ 流量统计 (ClickHouse) ============

// GetTrafficStats 流量统计: 总流量/请求/带宽/缓存命中率
// GET /api/v1/traffic/stats
func GetTrafficStats(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "统计服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{
			"total_traffic":   0,
			"total_requests":  0,
			"bandwidth":       0,
			"cache_hit_ratio": 0,
		}})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" {
		startTime = time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	query := `SELECT 
		sum(bytes) as total_traffic, 
		count() as total_requests,
		sumIf(bytes, cache_status = 'HIT') / sum(bytes) as cache_hit_ratio
	FROM access_logs 
	WHERE domain IN (?) AND timestamp >= ? AND timestamp <= ?`

	var totalTraffic uint64
	var totalRequests uint64
	var cacheHitRatio float64
	if err := ch.QueryRow(context.Background(), query, domains, startTime, endTime).Scan(&totalTraffic, &totalRequests, &cacheHitRatio); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 计算平均带宽 (bps): totalTraffic * 8 / 秒数
	layout := "2006-01-02 15:04:05"
	st, _ := time.Parse(layout, startTime)
	et, _ := time.Parse(layout, endTime)
	duration := et.Sub(st).Seconds()
	var bandwidth float64
	if duration > 0 {
		bandwidth = float64(totalTraffic) * 8 / duration
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total_traffic":   totalTraffic,
			"total_requests":  totalRequests,
			"bandwidth":       bandwidth,
			"cache_hit_ratio": cacheHitRatio,
		},
	})
}

// GetTrafficRanking 指标排行: Top IP/攻击IP/URL/UA/状态码
// GET /api/v1/traffic/ranking
func GetTrafficRanking(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "统计服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{}})
		return
	}

	metric := c.DefaultQuery("metric", "ip") // ip / attack_ip / url / ua / status
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" {
		startTime = time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	var field, table string
	switch metric {
	case "ip":
		field, table = "ip", "access_logs"
	case "attack_ip":
		field, table = "ip", "attack_logs"
	case "url":
		field, table = "url", "access_logs"
	case "ua":
		field, table = "user_agent", "access_logs"
	case "status":
		field, table = "status", "access_logs"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不支持的指标"})
		return
	}

	querySQL := fmt.Sprintf("SELECT %s as val, count() as cnt FROM %s WHERE domain IN (?) AND timestamp >= ? AND timestamp <= ? GROUP BY %s ORDER BY cnt DESC LIMIT %d", field, table, field, limit)
	rows, err := ch.Query(context.Background(), querySQL, domains, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var val string
		var cnt uint64
		if err := rows.Scan(&val, &cnt); err != nil {
			continue
		}
		list = append(list, gin.H{"name": val, "count": cnt})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"metric": metric,
			"list":   list,
		},
	})
}

// GetBandwidthTrend 带宽趋势
// GET /api/v1/traffic/bandwidth
func GetBandwidthTrend(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "统计服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []interface{}{}})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	granularity := c.DefaultQuery("granularity", "5m") // 1m / 5m / 1h / 1d

	if startTime == "" {
		startTime = time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	interval := intervalForGranularity(granularity)

	querySQL := fmt.Sprintf(`SELECT 
		toStartOfInterval(timestamp, INTERVAL %s) as t, 
		sum(bytes) * 8 as bandwidth_bits 
	FROM access_logs 
	WHERE domain IN (?) AND timestamp >= ? AND timestamp <= ? 
	GROUP BY t ORDER BY t`, interval)

	rows, err := ch.Query(context.Background(), querySQL, domains, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var t time.Time
		var bw uint64
		if err := rows.Scan(&t, &bw); err != nil {
			continue
		}
		list = append(list, gin.H{
			"timestamp": t.Format("2006-01-02 15:04:05"),
			"bandwidth": bw,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// GetCacheStats 缓存命中率统计
// GET /api/v1/traffic/cache
func GetCacheStats(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "统计服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": []interface{}{}, "hit_ratio": 0}})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" {
		startTime = time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	querySQL := `SELECT 
		cache_status, 
		count() as cnt 
	FROM access_logs 
	WHERE domain IN (?) AND timestamp >= ? AND timestamp <= ? 
	GROUP BY cache_status ORDER BY cnt DESC`

	rows, err := ch.Query(context.Background(), querySQL, domains, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	var total uint64
	var hitCount uint64
	for rows.Next() {
		var status string
		var cnt uint64
		if err := rows.Scan(&status, &cnt); err != nil {
			continue
		}
		list = append(list, gin.H{"cache_status": status, "count": cnt})
		total += cnt
		if status == "HIT" {
			hitCount = cnt
		}
	}

	var hitRatio float64
	if total > 0 {
		hitRatio = float64(hitCount) / float64(total)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       list,
			"hit_ratio":  hitRatio,
			"hit_count":  hitCount,
			"total":      total,
		},
	})
}

// intervalForGranularity 将粒度字符串转为 ClickHouse INTERVAL 字符串
func intervalForGranularity(g string) string {
	switch g {
	case "1m":
		return "1 MINUTE"
	case "5m":
		return "5 MINUTE"
	case "1h":
		return "1 HOUR"
	case "1d":
		return "1 DAY"
	default:
		return "5 MINUTE"
	}
}
