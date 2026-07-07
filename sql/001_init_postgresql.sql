-- ============================================================================
-- ShieldFlow PostgreSQL 数据库初始化脚本
-- Database: shieldflow_cdn
-- Engine: PostgreSQL 13+
-- Description: ShieldFlow 企业级自建CDN系统 - 关系型数据存储
-- ============================================================================

-- 创建数据库（如不存在）
-- 注意：该语句需在连接默认数据库（如 postgres）时执行
-- SELECT 'CREATE DATABASE shieldflow_cdn'
--   WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'shieldflow_cdn')\gexec
-- \c shieldflow_cdn

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 1. 用户表 users
--    系统用户信息，包括普通用户和管理员
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(64)  NOT NULL UNIQUE,
    email           VARCHAR(128) UNIQUE,
    phone           VARCHAR(32)  UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(16)  NOT NULL DEFAULT 'user'
                    CHECK (role IN ('user', 'admin')),
    status          SMALLINT     NOT NULL DEFAULT 1
                    CHECK (status IN (0, 1, 2)),  -- 0=禁用 1=正常 2=锁定
    real_name       VARCHAR(64),
    id_card         VARCHAR(32),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  users                 IS '用户表';
COMMENT ON COLUMN users.role            IS '角色：user=普通用户 admin=管理员';
COMMENT ON COLUMN users.status          IS '状态：0=禁用 1=正常 2=锁定';
COMMENT ON COLUMN users.password_hash   IS 'bcrypt 加密的密码哈希';

CREATE INDEX idx_users_email      ON users (email);
CREATE INDEX idx_users_phone      ON users (phone);
CREATE INDEX idx_users_role       ON users (role);
CREATE INDEX idx_users_status     ON users (status);
CREATE INDEX idx_users_created_at ON users (created_at);

-- ============================================================================
-- 2. 节点分组表 node_groups
--    CDN 边缘节点的逻辑分组
-- ============================================================================
CREATE TABLE IF NOT EXISTS node_groups (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(64)  NOT NULL UNIQUE,
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE node_groups IS '节点分组表';

-- ============================================================================
-- 3. 节点表 nodes
--    CDN 边缘节点信息
-- ============================================================================
CREATE TABLE IF NOT EXISTS nodes (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL UNIQUE,
    ip              INET         NOT NULL,
    region          VARCHAR(64),                     -- 区域，如华东、华北
    group_id        BIGINT       REFERENCES node_groups(id) ON DELETE SET NULL,
    status          VARCHAR(16)  NOT NULL DEFAULT 'offline'
                    CHECK (status IN ('online', 'offline', 'maintain')),
    license_key     VARCHAR(255),                    -- 授权密钥
    grpc_address    VARCHAR(255),                    -- gRPC 通信地址 host:port
    version         VARCHAR(32),                     -- 节点版本号
    cpu             INT,                             -- CPU 核数
    memory          BIGINT,                          -- 内存 (MB)
    disk            BIGINT,                          -- 磁盘 (GB)
    bandwidth       BIGINT,                          -- 带宽上限 (Mbps)
    last_heartbeat  TIMESTAMPTZ,                     -- 最近心跳时间
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  nodes                IS 'CDN 边缘节点表';
COMMENT ON COLUMN nodes.status         IS '状态：online=在线 offline=离线 maintain=维护中';
COMMENT ON COLUMN nodes.bandwidth      IS '带宽上限，单位 Mbps';
COMMENT ON COLUMN nodes.memory         IS '内存，单位 MB';
COMMENT ON COLUMN nodes.disk           IS '磁盘，单位 GB';

CREATE INDEX idx_nodes_ip        ON nodes (ip);
CREATE INDEX idx_nodes_region    ON nodes (region);
CREATE INDEX idx_nodes_group_id  ON nodes (group_id);
CREATE INDEX idx_nodes_status    ON nodes (status);
CREATE INDEX idx_nodes_heartbeat ON nodes (last_heartbeat);

-- ============================================================================
-- 4. 套餐表 packages
--    CDN 服务套餐定义（L7 七层 / L4 四层）
-- ============================================================================
CREATE TABLE IF NOT EXISTS packages (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    type            VARCHAR(8)   NOT NULL DEFAULT 'l7'
                    CHECK (type IN ('l7', 'l4')),  -- l7=七层 l4=四层
    traffic_limit   BIGINT       NOT NULL DEFAULT 0,  -- 流量限额 (GB)，0=不限
    bandwidth_limit BIGINT       NOT NULL DEFAULT 0,  -- 带宽限额 (Mbps)，0=不限
    domain_limit    INT          NOT NULL DEFAULT 0,  -- 域名数量上限，0=不限
    price           DECIMAL(10,2) NOT NULL DEFAULT 0,
    duration_days   INT          NOT NULL DEFAULT 30,  -- 套餐时长（天）
    status          SMALLINT     NOT NULL DEFAULT 1
                    CHECK (status IN (0, 1)),  -- 0=下架 1=上架
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  packages              IS '套餐表';
COMMENT ON COLUMN packages.type         IS '类型：l7=七层 l4=四层';
COMMENT ON COLUMN packages.traffic_limit IS '流量限额 (GB)，0 表示不限';
COMMENT ON COLUMN packages.bandwidth_limit IS '带宽限额 (Mbps)，0 表示不限';

CREATE INDEX idx_packages_type   ON packages (type);
CREATE INDEX idx_packages_status ON packages (status);

-- ============================================================================
-- 5. 用户套餐表 user_packages
--    用户已购买的套餐实例
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_packages (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_id      BIGINT       NOT NULL REFERENCES packages(id) ON DELETE RESTRICT,
    instance_no     VARCHAR(64)  NOT NULL UNIQUE,      -- 实例编号
    traffic_used    BIGINT       NOT NULL DEFAULT 0,   -- 已用流量 (Bytes)
    traffic_limit   BIGINT       NOT NULL DEFAULT 0,   -- 流量限额 (Bytes)
    domain_count    INT          NOT NULL DEFAULT 0,   -- 已绑定域名数
    status          SMALLINT     NOT NULL DEFAULT 1
                    CHECK (status IN (0, 1, 2, 3)),  -- 0=已取消 1=正常 2=已过期 3=已用尽
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  user_packages              IS '用户套餐表';
COMMENT ON COLUMN user_packages.status       IS '状态：0=已取消 1=正常 2=已过期 3=已用尽';
COMMENT ON COLUMN user_packages.traffic_used IS '已用流量，单位 Bytes';

CREATE INDEX idx_user_packages_user_id    ON user_packages (user_id);
CREATE INDEX idx_user_packages_package_id ON user_packages (package_id);
CREATE INDEX idx_user_packages_status     ON user_packages (status);
CREATE INDEX idx_user_packages_expires_at ON user_packages (expires_at);

-- ============================================================================
-- 6. 域名表 domains
--    用户接入的加速域名及其完整配置
-- ============================================================================
CREATE TABLE IF NOT EXISTS domains (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_name       VARCHAR(255) NOT NULL UNIQUE,
    package_id        BIGINT       REFERENCES user_packages(id) ON DELETE SET NULL,
    cname             VARCHAR(255),                       -- 系统分配的 CNAME
    status            SMALLINT     NOT NULL DEFAULT 0
                      CHECK (status IN (0, 1, 2, 3, 4)), -- 0=待配置 1=正常 2=已暂停 3=配置中 4=已删除
    origin_config     JSONB        NOT NULL DEFAULT '{}',  -- 源站配置
    https_config      JSONB        NOT NULL DEFAULT '{}',  -- HTTPS 配置
    cache_config      JSONB        NOT NULL DEFAULT '{}',  -- 缓存配置
    advanced_config   JSONB        NOT NULL DEFAULT '{}',  -- 高级配置
    custom_headers    JSONB        NOT NULL DEFAULT '[]',  -- 自定义响应头
    custom_pages      JSONB        NOT NULL DEFAULT '{}',  -- 自定义错误页
    protection_config JSONB        NOT NULL DEFAULT '{}',  -- 防护配置
    cname_domain      VARCHAR(255),                        -- 用户需解析的 CNAME 域名
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  domains                  IS '域名表';
COMMENT ON COLUMN domains.status           IS '状态：0=待配置 1=正常 2=已暂停 3=配置中 4=已删除';
COMMENT ON COLUMN domains.origin_config     IS '源站配置 JSON：{origin_list:[{ip,port,weight,host}],origin_type,origin_protocol,...}';
COMMENT ON COLUMN domains.https_config      IS 'HTTPS 配置 JSON：{enabled,force_https,http2,certificate_id,...}';
COMMENT ON COLUMN domains.cache_config      IS '缓存配置 JSON：{rules:[{match,ttl,enabled}],default_ttl,...}';
COMMENT ON COLUMN domains.advanced_config   IS '高级配置 JSON：{follow_redirect,websocket,ip_v6,...}';
COMMENT ON COLUMN domains.custom_headers    IS '自定义响应头 JSON 数组';
COMMENT ON COLUMN domains.custom_pages      IS '自定义错误页 JSON';
COMMENT ON COLUMN domains.protection_config IS '防护配置 JSON：{waf,cc,bot,template_id,...}';

CREATE INDEX idx_domains_user_id      ON domains (user_id);
CREATE INDEX idx_domains_package_id   ON domains (package_id);
CREATE INDEX idx_domains_status       ON domains (status);
CREATE INDEX idx_domains_created_at   ON domains (created_at);
-- GIN 索引加速 JSONB 查询
CREATE INDEX idx_domains_origin_config     ON domains USING GIN (origin_config);
CREATE INDEX idx_domains_https_config      ON domains USING GIN (https_config);
CREATE INDEX idx_domains_cache_config      ON domains USING GIN (cache_config);
CREATE INDEX idx_domains_protection_config ON domains USING GIN (protection_config);

-- ============================================================================
-- 7. SSL 证书表 certificates
--    域名 SSL 证书存储
-- ============================================================================
CREATE TABLE IF NOT EXISTS certificates (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id       BIGINT       REFERENCES domains(id) ON DELETE SET NULL,
    cert_pem        TEXT         NOT NULL,              -- 证书 PEM
    key_pem         TEXT         NOT NULL,              -- 私钥 PEM
    issuer          VARCHAR(255),                        -- 颁发机构
    common_name     VARCHAR(255),                        -- 通用名称
    san             TEXT,                                -- Subject Alternative Names（逗号分隔）
    not_before      TIMESTAMPTZ,                         -- 生效时间
    not_after       TIMESTAMPTZ,                         -- 过期时间
    status          SMALLINT     NOT NULL DEFAULT 1
                    CHECK (status IN (0, 1, 2)),  -- 0=已吊销 1=有效 2=已过期
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  certificates          IS 'SSL 证书表';
COMMENT ON COLUMN certificates.status   IS '状态：0=已吊销 1=有效 2=已过期';

CREATE INDEX idx_certs_user_id    ON certificates (user_id);
CREATE INDEX idx_certs_domain_id  ON certificates (domain_id);
CREATE INDEX idx_certs_not_after  ON certificates (not_after);
CREATE INDEX idx_certs_status     ON certificates (status);

-- ============================================================================
-- 8. ACME 账户表 acme_accounts
--    Let''s Encrypt 等 ACME 自动签发账户
-- ============================================================================
CREATE TABLE IF NOT EXISTS acme_accounts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email           VARCHAR(128) NOT NULL,
    key             TEXT         NOT NULL,              -- ACME 账户密钥 PEM
    ca_url          VARCHAR(255) NOT NULL DEFAULT 'https://acme-v02.api.letsencrypt.org/directory',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE acme_accounts IS 'ACME 账户表（自动证书签发）';

CREATE INDEX idx_acme_user_id ON acme_accounts (user_id);

-- ============================================================================
-- 9. DNS 账户表 dns_accounts
--    第三方 DNS 服务商账户，用于自动 CNAME 验证
-- ============================================================================
CREATE TABLE IF NOT EXISTS dns_accounts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        VARCHAR(32)  NOT NULL,              -- 服务商：cloudflare/aliyun/tencent/dnspod/...
    api_key         VARCHAR(255) NOT NULL,
    api_secret      VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE dns_accounts IS 'DNS 服务商账户表';

CREATE INDEX idx_dns_user_id   ON dns_accounts (user_id);
CREATE INDEX idx_dns_provider  ON dns_accounts (provider);

-- ============================================================================
-- 10. 黑白名单表 blacklists
--     针对 IP / URL / 域名的访问控制名单
-- ============================================================================
CREATE TABLE IF NOT EXISTS blacklists (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id       BIGINT       REFERENCES domains(id) ON DELETE CASCADE,
    type            VARCHAR(16)  NOT NULL
                    CHECK (type IN ('ip', 'url', 'domain')),
    list_type       VARCHAR(16)  NOT NULL
                    CHECK (list_type IN ('black', 'white')),
    value           VARCHAR(512) NOT NULL,             -- 名单值
    match_mode      VARCHAR(16)  NOT NULL DEFAULT 'exact'
                    CHECK (match_mode IN ('exact', 'prefix', 'suffix', 'regex', 'cidr')),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  blacklists             IS '黑白名单表';
COMMENT ON COLUMN blacklists.type        IS '类型：ip=IP名单 url=URL名单 domain=域名名单';
COMMENT ON COLUMN blacklists.list_type   IS '名单类型：black=黑名单 white=白名单';
COMMENT ON COLUMN blacklists.match_mode  IS '匹配模式：exact/prefix/suffix/regex/cidr';

CREATE INDEX idx_bl_user_id   ON blacklists (user_id);
CREATE INDEX idx_bl_domain_id ON blacklists (domain_id);
CREATE INDEX idx_bl_type      ON blacklists (type);
CREATE INDEX idx_bl_list_type ON blacklists (list_type);

-- ============================================================================
-- 11. 防护模板表 protection_templates
--     WAF / CC / Bot 等防护规则模板
-- ============================================================================
CREATE TABLE IF NOT EXISTS protection_templates (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       REFERENCES users(id) ON DELETE CASCADE,  -- NULL 表示系统模板
    name            VARCHAR(128) NOT NULL,
    description     TEXT,
    is_default      BOOLEAN      NOT NULL DEFAULT FALSE,
    config          JSONB        NOT NULL DEFAULT '{}',  -- 防护配置
    is_system       BOOLEAN      NOT NULL DEFAULT FALSE,  -- 是否系统内置
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  protection_templates        IS '防护模板表';
COMMENT ON COLUMN protection_templates.config IS '防护配置 JSON：{waf,cc,bot,ip_reputation,...}';

CREATE INDEX idx_pt_user_id    ON protection_templates (user_id);
CREATE INDEX idx_pt_is_default ON protection_templates (is_default);
CREATE INDEX idx_pt_is_system  ON protection_templates (is_system);
CREATE INDEX idx_pt_config     ON protection_templates USING GIN (config);

-- ============================================================================
-- 12. 四层转发表 layer4_forwards
--     TCP/UDP 四层负载均衡转发规则
-- ============================================================================
CREATE TABLE IF NOT EXISTS layer4_forwards (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id       BIGINT       REFERENCES domains(id) ON DELETE SET NULL,
    package_id      BIGINT       REFERENCES user_packages(id) ON DELETE SET NULL,
    protocol        VARCHAR(8)   NOT NULL
                    CHECK (protocol IN ('tcp', 'udp')),
    listen_port     INT          NOT NULL,             -- 监听端口
    lb_strategy     VARCHAR(32)  NOT NULL DEFAULT 'round_robin'
                    CHECK (lb_strategy IN ('round_robin', 'least_conn', 'ip_hash', 'weighted')),
    origins         JSONB        NOT NULL DEFAULT '[]', -- 源站列表 [{ip,port,weight}]
    advanced        JSONB        NOT NULL DEFAULT '{}', -- 高级配置 {proxy_protocol,health_check,...}
    status          SMALLINT     NOT NULL DEFAULT 1
                    CHECK (status IN (0, 1, 2)),  -- 0=已停用 1=正常 2=配置中
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  layer4_forwards            IS '四层转发表';
COMMENT ON COLUMN layer4_forwards.protocol   IS '协议：tcp/udp';
COMMENT ON COLUMN layer4_forwards.lb_strategy IS '负载均衡策略';
COMMENT ON COLUMN layer4_forwards.origins    IS '源站列表 JSON 数组';
COMMENT ON COLUMN layer4_forwards.advanced   IS '高级配置 JSON';

CREATE INDEX idx_l4_user_id   ON layer4_forwards (user_id);
CREATE INDEX idx_l4_domain_id ON layer4_forwards (domain_id);
CREATE INDEX idx_l4_protocol  ON layer4_forwards (protocol);
CREATE INDEX idx_l4_port      ON layer4_forwards (listen_port);
CREATE INDEX idx_l4_status    ON layer4_forwards (status);

-- ============================================================================
-- 13. 缓存刷新任务表 cache_tasks
--     文件刷新 / 目录刷新 / 文件预热任务
-- ============================================================================
CREATE TABLE IF NOT EXISTS cache_tasks (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id       BIGINT       REFERENCES domains(id) ON DELETE SET NULL,
    type            VARCHAR(32)  NOT NULL
                    CHECK (type IN ('file_refresh', 'dir_refresh', 'file_preheat')),
    urls            TEXT[]       NOT NULL,             -- URL 列表
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'processing', 'success', 'failed', 'partial')),
    progress        INT          NOT NULL DEFAULT 0
                    CHECK (progress >= 0 AND progress <= 100),  -- 进度百分比
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  cache_tasks        IS '缓存刷新任务表';
COMMENT ON COLUMN cache_tasks.type   IS '类型：file_refresh=文件刷新 dir_refresh=目录刷新 file_preheat=文件预热';
COMMENT ON COLUMN cache_tasks.status IS '状态：pending/processing/success/failed/partial';
COMMENT ON COLUMN cache_tasks.progress IS '进度百分比 0-100';

CREATE INDEX idx_ct_user_id   ON cache_tasks (user_id);
CREATE INDEX idx_ct_domain_id ON cache_tasks (domain_id);
CREATE INDEX idx_ct_type      ON cache_tasks (type);
CREATE INDEX idx_ct_status    ON cache_tasks (status);
CREATE INDEX idx_ct_created   ON cache_tasks (created_at);

-- ============================================================================
-- 14. 系统设置表 system_settings
--     全局键值配置
-- ============================================================================
CREATE TABLE IF NOT EXISTS system_settings (
    id              BIGSERIAL PRIMARY KEY,
    key             VARCHAR(128) NOT NULL UNIQUE,
    value           TEXT,
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_settings IS '系统设置表';

-- ============================================================================
-- 15. DDoS 规则表 ddos_rules
--     DDoS 防护规则配置
-- ============================================================================
CREATE TABLE IF NOT EXISTS ddos_rules (
    id                      BIGSERIAL PRIMARY KEY,
    name                    VARCHAR(128) NOT NULL,
    scope                   VARCHAR(16)  NOT NULL DEFAULT 'global'
                            CHECK (scope IN ('global', 'node', 'domain')),
    node_ids                BIGINT[],                   -- 适用的节点ID列表（scope=node时）
    max_connections_per_ip  INT          NOT NULL DEFAULT 0,  -- 单IP最大连接数，0=不限
    max_pps                 INT          NOT NULL DEFAULT 0,  -- 最大包速率 pps，0=不限
    auto_ban                BOOLEAN      NOT NULL DEFAULT TRUE,  -- 是否自动封禁
    ban_duration            INT          NOT NULL DEFAULT 3600,  -- 封禁时长（秒）
    status                  SMALLINT     NOT NULL DEFAULT 1
                            CHECK (status IN (0, 1)),  -- 0=禁用 1=启用
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  ddos_rules                          IS 'DDoS 防护规则表';
COMMENT ON COLUMN ddos_rules.scope                    IS '范围：global=全局 node=指定节点 domain=指定域名';
COMMENT ON COLUMN ddos_rules.max_connections_per_ip   IS '单IP最大连接数，0 表示不限';
COMMENT ON COLUMN ddos_rules.max_pps                  IS '最大包速率 pps，0 表示不限';
COMMENT ON COLUMN ddos_rules.ban_duration             IS '封禁时长，单位秒';

CREATE INDEX idx_ddos_rules_scope   ON ddos_rules (scope);
CREATE INDEX idx_ddos_rules_status  ON ddos_rules (status);

-- ============================================================================
-- 16. DDoS 黑名单表 ddos_blacklist
-- ============================================================================
CREATE TABLE IF NOT EXISTS ddos_blacklist (
    id              BIGSERIAL PRIMARY KEY,
    ip              INET,
    cidr            CIDR,                              -- CIDR 网段（与 ip 二选一）
    type            VARCHAR(16)  NOT NULL DEFAULT 'black'
                    CHECK (type IN ('black', 'white')),
    reason          VARCHAR(512),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ddos_blacklist IS 'DDoS IP 黑白名单表';

CREATE INDEX idx_ddos_bl_ip   ON ddos_blacklist (ip);
CREATE INDEX idx_ddos_bl_cidr ON ddos_blacklist (cidr);
CREATE INDEX idx_ddos_bl_type ON ddos_blacklist (type);

-- ============================================================================
-- 17. DDoS 白名单表 ddos_whitelist
--     独立白名单表，便于高频查询
-- ============================================================================
CREATE TABLE IF NOT EXISTS ddos_whitelist (
    id              BIGSERIAL PRIMARY KEY,
    ip              INET,
    cidr            CIDR,
    type            VARCHAR(16)  NOT NULL DEFAULT 'white'
                    CHECK (type IN ('black', 'white')),
    reason          VARCHAR(512),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ddos_whitelist IS 'DDoS IP 白名单表';

CREATE INDEX idx_ddos_wl_ip   ON ddos_whitelist (ip);
CREATE INDEX idx_ddos_wl_cidr ON ddos_whitelist (cidr);

-- ============================================================================
-- 18. 日志服务器配置表 log_server_config
--     分布式架构下日志服务器/主节点的通信配置
-- ============================================================================
CREATE TABLE IF NOT EXISTS log_server_config (
    id              BIGSERIAL PRIMARY KEY,
    mode            VARCHAR(16)  NOT NULL DEFAULT 'master'
                    CHECK (mode IN ('master', 'log_server')),
    address         VARCHAR(255),                      -- 日志服务器地址 host:port
    token           VARCHAR(255),                      -- 通信认证 Token
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE log_server_config IS '日志服务器配置表';
COMMENT ON COLUMN log_server_config.mode IS '模式：master=主节点 log_server=日志服务器';

-- ============================================================================
-- 19. AI 配置表 ai_config
--     AI 分析功能所用大模型配置
-- ============================================================================
CREATE TABLE IF NOT EXISTS ai_config (
    id              BIGSERIAL PRIMARY KEY,
    provider        VARCHAR(64)  NOT NULL,              -- 服务商：openai/azure/ollama/...
    model           VARCHAR(128) NOT NULL,              -- 模型名称
    api_key         VARCHAR(255),                       -- API Key
    enabled         BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ai_config IS 'AI 大模型配置表';

-- ============================================================================
-- 20. 流量包表 traffic_packages
--     可单独购买的流量补充包
-- ============================================================================
CREATE TABLE IF NOT EXISTS traffic_packages (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    traffic_limit   BIGINT       NOT NULL,             -- 流量额度 (GB)
    price           DECIMAL(10,2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  traffic_packages              IS '流量包表';
COMMENT ON COLUMN traffic_packages.traffic_limit IS '流量额度，单位 GB';

-- ============================================================================
-- 21. 域名包表 domain_packages
--     可单独购买的域名额度补充包
-- ============================================================================
CREATE TABLE IF NOT EXISTS domain_packages (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    domain_limit    INT          NOT NULL,             -- 域名额度数
    price           DECIMAL(10,2) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  domain_packages              IS '域名包表';
COMMENT ON COLUMN domain_packages.domain_limit IS '域名额度数';

-- ============================================================================
-- 22. 购买记录表 orders
--     用户所有消费订单记录
-- ============================================================================
CREATE TABLE IF NOT EXISTS orders (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_no        VARCHAR(64)  NOT NULL UNIQUE,      -- 订单编号
    product_type    VARCHAR(32)  NOT NULL,             -- 产品类型：package/traffic/domain/...
    product_name    VARCHAR(128) NOT NULL,
    amount          DECIMAL(10,2) NOT NULL DEFAULT 0,
    channel         VARCHAR(32),                       -- 支付渠道：alipay/wechat/balance/...
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'paid', 'cancelled', 'refunded', 'failed')),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  orders             IS '购买记录表';
COMMENT ON COLUMN orders.status      IS '状态：pending/paid/cancelled/refunded/failed';

CREATE INDEX idx_orders_user_id    ON orders (user_id);
CREATE INDEX idx_orders_order_no   ON orders (order_no);
CREATE INDEX idx_orders_status     ON orders (status);
CREATE INDEX idx_orders_created_at ON orders (created_at);

-- ============================================================================
-- 23. 操作日志表 operation_logs
--     用户/管理员操作审计日志
-- ============================================================================
CREATE TABLE IF NOT EXISTS operation_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT,                             -- 操作者ID（系统操作可为 NULL）
    action          VARCHAR(64)  NOT NULL,              -- 操作动作
    target          VARCHAR(255),                       -- 操作对象
    detail          TEXT,                               -- 详情
    ip              INET,                               -- 操作IP
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE operation_logs IS '操作日志表';

CREATE INDEX idx_op_logs_user_id    ON operation_logs (user_id);
CREATE INDEX idx_op_logs_action     ON operation_logs (action);
CREATE INDEX idx_op_logs_created_at ON operation_logs (created_at);

-- ============================================================================
-- 24. 余额表 balances
--     用户账户余额
-- ============================================================================
CREATE TABLE IF NOT EXISTS balances (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT       NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance         DECIMAL(12,2) NOT NULL DEFAULT 0,  -- 可用余额
    frozen          DECIMAL(12,2) NOT NULL DEFAULT 0,  -- 冻结余额
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  balances          IS '用户余额表';
COMMENT ON COLUMN balances.balance  IS '可用余额';
COMMENT ON COLUMN balances.frozen   IS '冻结余额';

-- ============================================================================
-- 触发器：自动更新 updated_at
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为所有含 updated_at 列的表创建触发器
CREATE TRIGGER trg_users_updated_at          BEFORE UPDATE ON users               FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_nodes_updated_at          BEFORE UPDATE ON nodes               FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_domains_updated_at        BEFORE UPDATE ON domains             FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_protection_templates_upd  BEFORE UPDATE ON protection_templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_log_server_config_upd     BEFORE UPDATE ON log_server_config   FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_ai_config_updated_at      BEFORE UPDATE ON ai_config           FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_balances_updated_at       BEFORE UPDATE ON balances            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_system_settings_upd       BEFORE UPDATE ON system_settings     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 初始化默认数据
-- ============================================================================

-- 默认管理员（密码需在使用时重置）
INSERT INTO users (username, email, role, status, real_name, password_hash)
VALUES ('admin', 'admin@shieldflow.local', 'admin', 1, '系统管理员', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68lJ1dIqpjUOm')
ON CONFLICT (username) DO NOTHING;

-- 默认节点分组
INSERT INTO node_groups (name, description) VALUES ('default', '默认分组')
ON CONFLICT (name) DO NOTHING;

-- 默认系统设置
INSERT INTO system_settings (key, value) VALUES
    ('system_name',    'ShieldFlow'),
    ('system_version', '1.0.0'),
    ('default_dns',    'shieldflow.example.com')
ON CONFLICT (key) DO NOTHING;

-- ============================================================================
-- 执行完成
-- ============================================================================
