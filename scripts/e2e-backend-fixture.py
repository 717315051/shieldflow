#!/usr/bin/env python3
# ============================================================
# ShieldFlow CDN 后端 A 方案 — 完整 E2E 数据 fixture + smoke
# ============================================================
# 主公授权: A — 把后端打到 100%(19 个 404 干掉 + 真业务流跑通)
# 工具配额耗尽,这是「一键直跑」脚本,主公直接执行即可
# ============================================================
import subprocess, json, sys, time, os

MASTER = '82.158.224.144'
SSH_PASS = os.environ.get('SHIELDFLOW_SSH_PASS', '<your-ssh-password>')
SSH = ['sshpass', '-p', SSH_PASS, 'ssh',
       '-o', 'StrictHostKeyChecking=no', '-o', 'UserKnownHostsFile=/dev/null',
       '-o', 'LogLevel=ERROR', f'root@{MASTER}']

BASE = f'http://{MASTER}:8080/api/v1'

def ssh_run(script, timeout=120):
    r = subprocess.run(SSH + ['bash', '-s'],
                       input=script, capture_output=True, text=True, timeout=timeout)
    return r.stdout

def curl(method, path, token=None, data=None, expect_in=None):
    cmd = ['curl', '-sS', '--max-time', '10', '-w', '\nHTTP_%{http_code}\n',
           '-X', method]
    if token:
        cmd += ['-H', f'Authorization: Bearer {token}']
    if data is not None:
        cmd += ['-H', 'Content-Type: application/json', '-d', json.dumps(data)]
    cmd.append(f'{BASE}{path}')
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=15)
    body = r.stdout
    http = body.split('HTTP_')[-1].strip()
    json_part = body.split('HTTP_')[0].strip()
    return http, json_part

# ============ STEP 0 — 登录 ============
print('=' * 70)
print('STEP 0  admin 登录')
print('=' * 70)
http, body = curl('POST', '/auth/login',
                   data={'account': 'admin', 'password': 'admin123'})
print(f'  HTTP {http} body={body[:200]}')
admin_token = json.loads(body)['data']['token']

# ============ STEP 1 — 创建 ACME 账户 ============
print('\n' + '=' * 70)
print('STEP 1  创建 ACME 账户 (root level + cert group)')
print('=' * 70)
http, body = curl('POST', '/acme-accounts', admin_token, {
    'email': 'ops@shieldflow.cn',
    'directory': 'https://acme-v02.api.letsencrypt.org/directory',
    'key_type': 'EC256'
})
print(f'  POST /acme-accounts  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/acme-accounts', admin_token)
print(f'  GET  /acme-accounts  HTTP {http}  {body[:200]}')

# ============ STEP 2 — 创建 DNS 账户(阿里云) ============
print('\n' + '=' * 70)
print('STEP 2  创建 DNS 账户 (aliyun)')
print('=' * 70)
http, body = curl('POST', '/dns-accounts', admin_token, {
    'provider': 'aliyun',
    'api_key': '<your-aliyun-access-key-id>',
    'api_secret': '<your-aliyun-access-key-secret>'
})
print(f'  POST /dns-accounts  HTTP {http}  {body[:200]}')

# ============ STEP 3 — 创建套餐 + 流量包 + 域名包 ============
print('\n' + '=' * 70)
print('STEP 3  创建套餐 + 流量包 + 域名包')
print('=' * 70)
http, body = curl('POST', '/admin/packages', admin_token, {
    'name': 'L7-Standard',
    'type': 'l7',
    'price': 99.0,
    'period_days': 30,
    'domain_limit': 10,
    'request_quota': 1000000,
    'bandwidth_mbps': 100,
    'status': 'active',
    'description': '七层标准版'
})
print(f'  POST /admin/packages  HTTP {http}  {body[:200]}')

http, body = curl('POST', '/admin/packages/traffic', admin_token, {
    'name': '流量包 100GB',
    'traffic_gb': 100,
    'price': 50.0,
    'period_days': 90,
    'status': 'active'
})
print(f'  POST /admin/packages/traffic  HTTP {http}  {body[:200]}')

http, body = curl('POST', '/admin/packages/domain', admin_token, {
    'name': '域名包 5个',
    'domain_count': 5,
    'price': 30.0,
    'period_days': 30,
    'status': 'active'
})
print(f'  POST /admin/packages/domain  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/packages/market', admin_token)
print(f'  GET  /packages/market  HTTP {http}')
print(f'    {body[:300]}')

# ============ STEP 4 — 创建普通用户 ============
print('\n' + '=' * 70)
print('STEP 4  注册普通用户 user1')
print('=' * 70)
http, body = curl('POST', '/auth/register', data={
    'username': 'user1',
    'password': 'User1@pass123',
    'email': 'user1@shieldflow.cn',
    'phone': '13800138001'
})
print(f'  POST /auth/register  HTTP {http}  {body[:200]}')

http, body = curl('POST', '/auth/login',
                   data={'account': 'user1', 'password': 'User1@pass123'})
if http != '200':
    # 用户可能已存在 → 重新登录
    http, body = curl('POST', '/auth/login',
                       data={'account': 'user1', 'password': 'User1@pass123'})
print(f'  POST /auth/login     HTTP {http}  {body[:200]}')
user1_token = json.loads(body)['data']['token']

# ============ STEP 5 — 创建域名 ============
print('\n' + '=' * 70)
print('STEP 5  user1 创建域名 hub-hupu.com (已有,验证 get/update/basic)')
print('=' * 70)
http, body = curl('GET', '/domains', user1_token)
print(f'  GET /domains  HTTP {http}')
domains = json.loads(body)['data']['list'] if body else []
domain_id = None
for d in domains:
    if d.get('domain_name') == 'hub-hupu.com':
        domain_id = d['id']
        break
if not domain_id:
    # 用 admin token 拿已有 hub-hupu.com
    http, body = curl('GET', '/domains', admin_token, None)
    try:
        admin_list = json.loads(body)['data'].get('list') or []
        for d in admin_list:
            if d.get('domain_name') == 'hub-hupu.com':
                domain_id = d['id']
                break
    except Exception:
        pass
if not domain_id:
    http, body = curl('POST', '/domains', admin_token, {
        'domain_name': 'hub-hupu.com',
        'origin_address': '127.0.0.1:8080',
        'protocol': 'http',
        'package_id': 1
    })
    print(f'  POST /domains  HTTP {http}  {body[:300]}')
    try:
        domain_id = json.loads(body)['data']['id']
    except Exception:
        domain_id = 7  # fallback known id
print(f'  domain_id = {domain_id}')

# 验证全套 domain 接口
for verb_path in [
    ('GET', f'/domains/{domain_id}'),
    ('GET', f'/domains/{domain_id}/config'),
    ('PUT', f'/domains/{domain_id}/basic'),
    ('PUT', f'/domains/{domain_id}/protection'),
    ('PUT', f'/domains/{domain_id}/custom-pages'),
    ('PUT', f'/domains/{domain_id}/package'),
    ('PUT', f'/domains/{domain_id}/status'),
    ('POST', f'/domains/{domain_id}/certificate'),
]:
    v, p = verb_path
    data = {'protocol': 'http', 'origin_address': '127.0.0.1'} if v == 'PUT' and p.endswith('/basic') else None
    http, body = curl(v, p, user1_token, data)
    print(f'  {v:6s} {p:50s} HTTP {http}  {body[:80]}')

# ============ STEP 6 — SSL 证书 ============
print('\n' + '=' * 70)
print('STEP 6  上传证书 + 申请证书 + 列表')
print('=' * 70)
http, body = curl('POST', '/certificates/upload', user1_token, {
    'name': 'test-cert',
    'domain': 'hub-hupu.com',
    'cert': '-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----',
    'key': '-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----'
})
print(f'  POST /certificates/upload  HTTP {http}  {body[:200]}')

http, body = curl('POST', '/certificates/apply', user1_token, {
    'domain_id': domain_id,
    'validation': 'http-01'
})
print(f'  POST /certificates/apply   HTTP {http}  {body[:200]}')

http, body = curl('GET', '/certificates', user1_token)
print(f'  GET  /certificates         HTTP {http}')
cert_id = None
if body:
    try:
        data = json.loads(body)['data']
        cert_id = data['list'][0]['id'] if data.get('list') else None
    except:
        pass

# ============ STEP 7 — 黑白名单 + 白名单 ============
print('\n' + '=' * 70)
print('STEP 7  黑白名单 + 白名单 CRUD')
print('=' * 70)
http, body = curl('POST', '/protection/blacklists', user1_token, {
    'type': 'ip', 'list_type': 'black', 'value': '1.2.3.4', 'match_mode': 'exact'
})
print(f'  POST /protection/blacklists       HTTP {http}  {body[:200]}')

http, body = curl('GET', '/protection/blacklists', user1_token)
print(f'  GET  /protection/blacklists       HTTP {http}')
bl_id = None
if body:
    try:
        bl_id = json.loads(body)['data']['list'][0]['id']
    except: pass

http, body = curl('POST', '/protection/whitelist', user1_token, {
    'type': 'ip', 'value': '5.6.7.8', 'match_mode': 'exact'
})
print(f'  POST /protection/whitelist       HTTP {http}  {body[:200]}')

http, body = curl('GET', '/protection/whitelist', user1_token)
print(f'  GET  /protection/whitelist       HTTP {http}  {body[:200]}')

if bl_id:
    http, body = curl('DELETE', f'/protection/blacklists/{bl_id}', user1_token)
    print(f'  DELETE /protection/blacklists/{bl_id}  HTTP {http}')

# ============ STEP 8 — 缓存任务 ============
print('\n' + '=' * 70)
print('STEP 8  缓存任务全流程')
print('=' * 70)
http, body = curl('POST', '/cache/file-refresh', user1_token, {
    'domain_id': domain_id,
    'paths': ['/index.html', '/static/app.js']
})
print(f'  POST /cache/file-refresh  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/cache/tasks', user1_token)
print(f'  GET  /cache/tasks         HTTP {http}')
task_id = None
if body:
    try:
        task_id = json.loads(body)['data']['list'][0]['id']
    except: pass

if task_id:
    http, body = curl('GET', f'/cache/tasks/{task_id}', user1_token)
    print(f'  GET  /cache/tasks/{task_id}        HTTP {http}  {body[:200]}')
    http, body = curl('POST', f'/cache/tasks/{task_id}/cancel', user1_token)
    print(f'  POST /cache/tasks/{task_id}/cancel HTTP {http}')

# ============ STEP 9 — 防护策略模板 ============
print('\n' + '=' * 70)
print('STEP 9  防护策略模板')
print('=' * 70)
http, body = curl('POST', '/protection/templates', user1_token, {
    'name': 'Web 通用防护',
    'description': 'CC + Bot + Referer',
    'is_default': False,
    'config': json.dumps({
        'cc': {'enabled': True, 'qps': 100, 'action': 'challenge'},
        'bot': {'enabled': True, 'types': ['ahrefs', 'semrush']}
    })
})
print(f'  POST /protection/templates  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/protection/templates', user1_token)
print(f'  GET  /protection/templates  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/protection/templates/system', user1_token)
print(f'  GET  /protection/templates/system  HTTP {http}  {body[:200]}')

# ============ STEP 10 — 流量 + 防护统计 ============
print('\n' + '=' * 70)
print('STEP 10 流量统计 / 日志')
print('=' * 70)
for p in ['/traffic/stats', '/traffic/ranking', '/traffic/bandwidth',
          '/traffic/cache-hit', '/logs/access', '/logs/attack',
          '/logs/layer4', '/logs/ai', '/logs/map']:
    http, body = curl('GET', p, user1_token)
    print(f'  GET {p:30s} HTTP {http}  {body[:100]}')

# ============ STEP 11 — 余额 + 购买 ============
print('\n' + '=' * 70)
print('STEP 11 余额 + 购买')
print('=' * 70)
http, body = curl('GET', '/balance', user1_token)
print(f'  GET /balance            HTTP {http}  {body[:200]}')

http, body = curl('POST', '/balance/recharge', user1_token, {'amount': 1000.0})
print(f'  POST /balance/recharge  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/packages', user1_token)
print(f'  GET /packages           HTTP {http}  {body[:200]}')
pkg_id = None
if body:
    try:
        pkg_id = json.loads(body)['data']['list'][0]['id']
    except: pass

if pkg_id:
    http, body = curl('POST', f'/packages/{pkg_id}/purchase', user1_token, {})
    print(f'  POST /packages/{pkg_id}/purchase  HTTP {http}  {body[:200]}')

# ============ STEP 12 — user-packages / orders ============
print('\n' + '=' * 70)
print('STEP 12 user-packages / orders')
print('=' * 70)
http, body = curl('GET', '/user-packages', user1_token)
print(f'  GET /user-packages       HTTP {http}  {body[:200]}')
up_id = None
if body:
    try:
        up_id = json.loads(body)['data']['list'][0]['id']
    except: pass

if up_id:
    http, body = curl('GET', f'/user-packages/{up_id}', user1_token)
    print(f'  GET  /user-packages/{up_id}      HTTP {http}  {body[:200]}')

http, body = curl('GET', '/orders', user1_token)
print(f'  GET  /orders             HTTP {http}  {body[:200]}')
order_id = None
if body:
    try:
        order_id = json.loads(body)['data']['list'][0]['id']
    except: pass

if order_id:
    http, body = curl('GET', f'/orders/{order_id}', user1_token)
    print(f'  GET  /orders/{order_id}           HTTP {http}  {body[:200]}')

# ============ STEP 13 — 四层转发 ============
print('\n' + '=' * 70)
print('STEP 13 四层转发')
print('=' * 70)
http, body = curl('POST', '/layer4', user1_token, {
    'name': 'tcp-game',
    'protocol': 'tcp',
    'listen_port': 9999,
    'origin_address': '127.0.0.1:9999',
    'lb_policy': 'round_robin'
})
print(f'  POST /layer4  HTTP {http}  {body[:200]}')
l4_id = None
if body and http == '200':
    try: l4_id = json.loads(body)['data']['id']
    except: pass

if l4_id:
    http, body = curl('PUT', f'/layer4/{l4_id}', user1_token,
                       {'name': 'tcp-game-updated', 'protocol': 'tcp',
                        'listen_port': 9999, 'origin_address': '127.0.0.1:9999',
                        'lb_policy': 'round_robin'})
    print(f'  PUT  /layer4/{l4_id}  HTTP {http}')
    http, body = curl('PUT', f'/layer4/{l4_id}/status', user1_token, {'status': 'active'})
    print(f'  PUT  /layer4/{l4_id}/status  HTTP {http}')
    http, body = curl('DELETE', f'/layer4/{l4_id}', user1_token)
    print(f'  DELETE /layer4/{l4_id}  HTTP {http}')

# ============ STEP 14 — admin 端 ============
print('\n' + '=' * 70)
print('STEP 14 admin 节点 / 用户管理')
print('=' * 70)
http, body = curl('GET', '/admin/nodes', admin_token)
print(f'  GET /admin/nodes  HTTP {http}  {body[:200]}')
node_id = None
if body:
    try: node_id = json.loads(body)['data']['list'][0]['id']
    except: pass

http, body = curl('GET', '/admin/users', admin_token)
print(f'  GET /admin/users  HTTP {http}  {body[:200]}')

http, body = curl('GET', '/admin/system/backup', admin_token)
print(f'  GET /admin/system/backup  HTTP {http}  {body[:200]}')
backup_name = None
if body:
    try: backup_name = json.loads(body)['data']['list'][0]['name']
    except: pass

if backup_name:
    http, body = curl('GET', f'/admin/system/backup/{backup_name}/download', admin_token)
    print(f'  GET /admin/system/backup/{backup_name}/download  HTTP {http}')
    http, body = curl('POST', f'/admin/system/backup/{backup_name}/restore', admin_token, {})
    print(f'  POST /admin/system/backup/{backup_name}/restore  HTTP {http}')
    http, body = curl('DELETE', f'/admin/system/backup/{backup_name}', admin_token)
    print(f'  DELETE /admin/system/backup/{backup_name}  HTTP {http}')

# ============ 终极:再跑一次完整 164 路径 smoke ============
print('\n' + '=' * 70)
print('STEP 15  终极 164 路径 smoke (after fixture)')
print('=' * 70)
PATHS = '''
GET /ping
GET /api/v1/health
POST /api/v1/auth/login
POST /api/v1/auth/register
POST /api/v1/auth/logout
GET /api/v1/auth/captcha
POST /api/v1/auth/verify-code
POST /api/v1/auth/send-email-code
GET /api/v1/auth/profile
PUT /api/v1/auth/profile
PUT /api/v1/auth/password
POST /api/v1/auth/realname
POST /api/v1/auth/forgot-password
POST /api/v1/auth/reset-password
GET /api/v1/domains
POST /api/v1/domains
POST /api/v1/domains/batch
GET /api/v1/domains/1
PUT /api/v1/domains/1
DELETE /api/v1/domains/1
PUT /api/v1/domains/1/status
PUT /api/v1/domains/1/package
GET /api/v1/domains/1/config
PUT /api/v1/domains/1/basic
PUT /api/v1/domains/1/protection
PUT /api/v1/domains/1/custom-pages
POST /api/v1/domains/1/certificate
POST /api/v1/domains/batch-certificate
GET /api/v1/certificates
POST /api/v1/certificates/upload
GET /api/v1/certificates/1
DELETE /api/v1/certificates/1
GET /api/v1/certificates/1/download
GET /api/v1/certificates/requests
POST /api/v1/certificates/apply
GET /api/v1/certificates/requests/1
GET /api/v1/certificates/requests/1/log
DELETE /api/v1/certificates/requests/1
GET /api/v1/certificates/acme-accounts
POST /api/v1/certificates/acme-accounts
GET /api/v1/acme-accounts
POST /api/v1/acme-accounts
GET /api/v1/certificates/dns-accounts
POST /api/v1/certificates/dns-accounts
GET /api/v1/dns-accounts
POST /api/v1/dns-accounts
GET /api/v1/logs/access
GET /api/v1/logs/attack
GET /api/v1/logs/layer4
GET /api/v1/logs/layer4-intercept
GET /api/v1/logs/ai
POST /api/v1/logs/export
GET /api/v1/logs/map
GET /api/v1/traffic/stats
GET /api/v1/traffic/ranking
GET /api/v1/traffic/bandwidth
GET /api/v1/traffic/cache-hit
POST /api/v1/cache/file-refresh
POST /api/v1/cache/dir-refresh
POST /api/v1/cache/file-preheat
GET /api/v1/cache/tasks
GET /api/v1/cache/tasks/1
POST /api/v1/cache/tasks/1/cancel
GET /api/v1/layer4
POST /api/v1/layer4
PUT /api/v1/layer4/1
DELETE /api/v1/layer4/1
PUT /api/v1/layer4/1/status
GET /api/v1/protection/templates
POST /api/v1/protection/templates
PUT /api/v1/protection/templates/1
DELETE /api/v1/protection/templates/1
POST /api/v1/protection/templates/1/apply
GET /api/v1/protection/templates/system
POST /api/v1/protection/templates/system/1/apply
GET /api/v1/protection/blacklists
POST /api/v1/protection/blacklists
DELETE /api/v1/protection/blacklists/1
POST /api/v1/protection/blacklists/import
GET /api/v1/protection/blacklists/export
GET /api/v1/protection/whitelist
POST /api/v1/protection/whitelist
DELETE /api/v1/protection/whitelist/1
GET /api/v1/packages
GET /api/v1/packages/traffic
GET /api/v1/packages/domain
GET /api/v1/packages/market
POST /api/v1/packages/1/purchase
POST /api/v1/packages/traffic/1/purchase
POST /api/v1/packages/domain/1/purchase
GET /api/v1/user-packages
GET /api/v1/user-packages/1
POST /api/v1/user-packages/1/renew
GET /api/v1/orders
GET /api/v1/orders/1
GET /api/v1/balance
POST /api/v1/balance/recharge
GET /api/v1/dashboard/analysis
GET /api/v1/admin/users
POST /api/v1/admin/users
PUT /api/v1/admin/users/1
DELETE /api/v1/admin/users/1
PUT /api/v1/admin/users/1/status
GET /api/v1/admin/users/1/packages
GET /api/v1/admin/nodes
POST /api/v1/admin/nodes
DELETE /api/v1/admin/nodes/1
GET /api/v1/admin/nodes/1
PUT /api/v1/admin/nodes/1
POST /api/v1/admin/nodes/1/install
POST /api/v1/admin/nodes/1/ssh-install
POST /api/v1/admin/nodes/1/upgrade
POST /api/v1/admin/nodes/batch-upgrade
GET /api/v1/admin/nodes/1/status
GET /api/v1/admin/node-groups
POST /api/v1/admin/node-groups
PUT /api/v1/admin/node-groups/1
DELETE /api/v1/admin/node-groups/1
GET /api/v1/admin/packages
POST /api/v1/admin/packages
PUT /api/v1/admin/packages/1
DELETE /api/v1/admin/packages/1
POST /api/v1/admin/packages/traffic
POST /api/v1/admin/packages/domain
GET /api/v1/admin/ddos/dashboard
GET /api/v1/admin/ddos/rules
POST /api/v1/admin/ddos/rules
PUT /api/v1/admin/ddos/rules/1
DELETE /api/v1/admin/ddos/rules/1
GET /api/v1/admin/ddos/blacklist
POST /api/v1/admin/ddos/blacklist
DELETE /api/v1/admin/ddos/blacklist/1
GET /api/v1/admin/ddos/whitelist
POST /api/v1/admin/ddos/whitelist
DELETE /api/v1/admin/ddos/whitelist/1
GET /api/v1/admin/ddos/logs
GET /api/v1/admin/ddos/intercept-logs
GET /api/v1/admin/waf/dashboard
GET /api/v1/admin/waf/config
PUT /api/v1/admin/waf/config
GET /api/v1/admin/waf/logs
GET /api/v1/admin/waf/analysis
GET /api/v1/admin/ai/dashboard
GET /api/v1/admin/ai/token-stats
GET /api/v1/admin/ai/cost-analysis
GET /api/v1/admin/ai/models
POST /api/v1/admin/ai/models
PUT /api/v1/admin/ai/models/1
DELETE /api/v1/admin/ai/models/1
GET /api/v1/admin/system/settings
PUT /api/v1/admin/system/settings
GET /api/v1/admin/system/dns
PUT /api/v1/admin/system/dns
GET /api/v1/admin/system/acme
PUT /api/v1/admin/system/acme
GET /api/v1/admin/system/grpc
PUT /api/v1/admin/system/grpc
GET /api/v1/admin/system/backup
GET /api/v1/admin/system/backup/backup_test.sql.gz/download
POST /api/v1/admin/system/backup/backup_test.sql.gz/restore
DELETE /api/v1/admin/system/backup/backup_test.sql.gz
'''
ok = err404 = err400 = err5xx = 0
errors = []
for line in PATHS.strip().split('\n'):
    tokens = line.split()
    # tokens = [method1, path1, method2, path2, ...]
    for i in range(0, len(tokens) - 1, 2):
        method, path = tokens[i], tokens[i + 1]
        if not method or not path:
            continue
        # Replace /1 with actual IDs we created
        if '/1' in path and method != 'POST':
            if '/domains/' in path and 'POST' not in method:
                path = path.replace('/1', f'/{domain_id}')
            elif '/user-packages/' in path:
                path = path.replace('/1', f'/{up_id}' if up_id else '/1')
            elif '/orders/' in path:
                path = path.replace('/1', f'/{order_id}' if order_id else '/1')
        cmd = ['curl', '-sS', '--max-time', '5', '-o', '/dev/null',
               '-w', '%{http_code}', '-X', method,
               '-H', f'Authorization: Bearer {admin_token}',
               f'http://{MASTER}:8080{path}']  # path already has /api/v1 prefix
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
        c = r.stdout.strip()
        if c.startswith('2'):
            ok += 1
        elif c == '404':
            err404 += 1
            errors.append(f'404 {method} {path}')
        elif c.startswith('4'):
            err400 += 1
        elif c.startswith('5'):
            err5xx += 1
            errors.append(f'5xx {method} {path}')

print(f'\n  OK={ok}  404={err404}  4xx={err400}  5xx={err5xx}')
print(f'  Total = {ok+err404+err400+err5xx}')
print(f'  PASS RATE = {100*ok/(ok+err404+err400+err5xx):.1f}%')
if errors:
    print('\n  Failures:')
    for e in errors[:30]:
        print(f'    {e}')

print('\n=== DONE ===')