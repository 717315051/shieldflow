-- ============================================================================
-- ShieldFlow ClickHouse 数据库初始化脚本
-- Database: shieldflow_cdn
-- Engine: ClickHouse 22+
-- Description: ShieldFlow 企业级自建CDN系统 - 日志与统计数据存储
--
-- 说明：ClickHouse 不支持 IF NOT EXISTS 创建数据库的常规语法，
--       请先手动执行：CREATE DATABASE IF NOT EXISTS shieldflow_cdn
-- ============================================================================

CREATE DATABASE IF NOT EXISTS shieldflow_cdn;

-- ============================================================================
-- 1. 访问日志表 access_logs
--    七层 HTTP/HTTPS 访问日志
--    分区：按天 | TTL：30天
-- ============================================================================
CREATE TABLE IF NOT EXISTS shieldflow_cdn.access_logs
(
    `domain`        String          COMMENT '加速域名',
    `method`        String          COMMENT 'HTTP 方法',
    `url`           String          COMMENT '请求 URL（含 query）',
    `status_code`   UInt16          COMMENT 'HTTP 状态码',
    `client_ip`     IPv4            COMMENT '客户端 IP',
    `response_time` Float32         COMMENT '响应时间（毫秒）',
    `cache_hit`     UInt8           COMMENT '缓存命中：1=命中 0=未命中',
    `bytes_sent`    UInt64          COMMENT '发送字节数',
    `user_agent`    String          COMMENT 'User-Agent',
    `referer`       String          COMMENT 'Referer',
    `timestamp`     DateTime        COMMENT '请求时间'
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (domain, timestamp, client_ip)
TTL timestamp + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

COMMENT ON TABLE shieldflow_cdn.access_logs IS 'CDN 七层访问日志表，TTL 30天';

-- ============================================================================
-- 2. 攻击日志表 attack_logs
--    WAF / CC / Bot 等安全防护事件日志
--    分区：按月 | TTL：90天
-- ============================================================================
CREATE TABLE IF NOT EXISTS shieldflow_cdn.attack_logs
(
    `domain`           String          COMMENT '加速域名',
    `attack_type`      String          COMMENT '攻击类型：sqli/xss/cc/bot/scan/... ',
    `protection_type`  String          COMMENT '防护类型：waf/cc/bot/ip_reputation/... ',
    `method`           String          COMMENT 'HTTP 方法',
    `client_ip`        IPv4            COMMENT '攻击源 IP',
    `url`              String          COMMENT '请求 URL',
    `action`           String          COMMENT '处置动作：block/captcha/log/pass',
    `detail`           String          COMMENT '详细信息（JSON）',
    `timestamp`        DateTime        COMMENT '事件时间'
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (domain, timestamp, client_ip)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

COMMENT ON TABLE shieldflow_cdn.attack_logs IS '安全防护攻击日志表，TTL 90天';

-- ============================================================================
-- 3. DDoS 日志表 ddos_logs
--    DDoS 防护事件日志
--    分区：按月 | TTL：90天
-- ============================================================================
CREATE TABLE IF NOT EXISTS shieldflow_cdn.ddos_logs
(
    `domain`        String          COMMENT '加速域名（四层可为空）',
    `protocol`      String          COMMENT '协议：tcp/udp',
    `listen_port`   UInt16          COMMENT '监听端口',
    `client_ip`     IPv4            COMMENT '源 IP',
    `action`        String          COMMENT '处置动作：block/limit/pass/ban',
    `reason`        String          COMMENT '触发原因',
    `timestamp`     DateTime        COMMENT '事件时间'
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (domain, timestamp, client_ip)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

COMMENT ON TABLE shieldflow_cdn.ddos_logs IS 'DDoS 防护日志表，TTL 90天';

-- ============================================================================
-- 4. 四层日志表 layer4_logs
--    TCP/UDP 四层转发事件日志
--    分区：按天 | TTL：30天
-- ============================================================================
CREATE TABLE IF NOT EXISTS shieldflow_cdn.layer4_logs
(
    `domain`        String          COMMENT '加速域名（四层可为空）',
    `event`         String          COMMENT '事件类型：connect/disconnect/timeout/... ',
    `protocol`      String          COMMENT '协议：tcp/udp',
    `src_ip`        IPv4            COMMENT '源 IP',
    `src_port`      UInt16          COMMENT '源端口',
    `dst_ip`        IPv4            COMMENT '目标 IP（边缘节点）',
    `dst_port`      UInt16          COMMENT '目标端口',
    `edge_ip`       IPv4            COMMENT '边缘节点 IP',
    `timestamp`     DateTime        COMMENT '事件时间'
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (domain, timestamp, src_ip)
TTL timestamp + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

COMMENT ON TABLE shieldflow_cdn.layer4_logs IS 'CDN 四层转发日志表，TTL 30天';

-- ============================================================================
-- 5. AI 日志表 ai_logs
--    AI 分析功能调用记录
--    分区：按月 | TTL：365天
-- ============================================================================
CREATE TABLE IF NOT EXISTS shieldflow_cdn.ai_logs
(
    `domain`        String          COMMENT '加速域名',
    `feature`       String          COMMENT 'AI 功能：log_analysis/threat_detect/... ',
    `token_count`   UInt32          COMMENT '消耗 Token 数',
    `cost`          Decimal(10,4)   COMMENT '费用（元）',
    `timestamp`     DateTime        COMMENT '调用时间'
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (domain, timestamp)
TTL timestamp + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

COMMENT ON TABLE shieldflow_cdn.ai_logs IS 'AI 分析调用日志表，TTL 365天';

-- ============================================================================
-- 6. 带宽统计表 bandwidth_stats
--    按域名维度的带宽/流量/请求数统计（预聚合）
--    分区：按月 | TTL：365天
-- ============================================================================
CREATE TABLE IF NOT EXISTS shieldflow_cdn.bandwidth_stats
(
    `domain`         String          COMMENT '加速域名',
    `timestamp`      DateTime        COMMENT '统计时间点',
    `request_count`  UInt64          COMMENT '请求数',
    `traffic_bytes`  UInt64          COMMENT '流量字节数',
    `bandwidth_bps`  UInt64          COMMENT '带宽 (bps)'
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (domain, timestamp)
TTL timestamp + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

COMMENT ON TABLE shieldflow_cdn.bandwidth_stats IS '带宽统计表（预聚合），TTL 365天';

-- ============================================================================
-- 执行完成
-- ============================================================================
