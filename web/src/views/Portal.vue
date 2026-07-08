<template>
  <div class="portal">
    <!-- 导航栏 -->
    <header class="nav">
      <div class="container nav-content">
        <div class="logo">
          <span class="logo-text">ShieldFlow</span>
        </div>
        <nav class="nav-links">
          <a href="#features">功能特性</a>
          <a href="#protection">安全防护</a>
          <a href="#pricing">套餐价格</a>
          <a href="#docs">文档</a>
        </nav>
        <div class="nav-actions">
          <a-button type="text" @click="$router.push('/login')">登录</a-button>
          <a-button type="primary" @click="$router.push('/register')">免费注册</a-button>
        </div>
      </div>
    </header>

    <!-- Hero -->
    <section class="hero">
      <div class="container hero-content">
        <h1>企业级自建CDN<br/>智能安全防护平台</h1>
        <p class="hero-desc">DDoS防护 · WAF语义分析 · AI智能防护 · 全球加速</p>
        <div class="hero-actions">
          <a-button type="primary" size="large" @click="$router.push('/register')">立即使用</a-button>
          <a-button size="large" @click="scrollTo('features')">了解更多</a-button>
        </div>
        <div class="hero-stats">
          <div class="stat"><span class="num">10T+</span><span class="label">DDoS防护</span></div>
          <div class="stat"><span class="num">7层</span><span class="label">安全防护</span></div>
          <div class="stat"><span class="num">99.9%</span><span class="label">可用性</span></div>
          <div class="stat"><span class="num">AI</span><span class="label">智能防护</span></div>
        </div>
      </div>
    </section>

    <!-- 功能特性 -->
    <section id="features" class="features">
      <div class="container">
        <h2 class="section-title">功能特性</h2>
        <div class="feature-grid">
          <div class="feature-card" v-for="f in features" :key="f.title">
            <div class="feature-icon">
              <component :is="f.icon" :style="{ fontSize: '32px' }" />
            </div>
            <h3>{{ f.title }}</h3>
            <p>{{ f.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- 安全防护 -->
    <section id="protection" class="protection">
      <div class="container">
        <h2 class="section-title">7层安全防护</h2>
        <div class="protection-flow">
          <div class="flow-item" v-for="(p, i) in protectionLayers" :key="p">
            <div class="flow-num">{{ i + 1 }}</div>
            <div class="flow-text">{{ p }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- 套餐价格 -->
    <section id="pricing" class="pricing">
      <div class="container">
        <h2 class="section-title">套餐价格</h2>
        <div class="pricing-grid">
          <div class="pricing-card" v-for="p in pricingPlans" :key="p.name"
               :class="{ featured: p.featured }">
            <h3>{{ p.name }}</h3>
            <div class="price"><span class="amount">{{ p.price }}</span><span class="unit">/月</span></div>
            <ul>
              <li v-for="f in p.features" :key="f">{{ f }}</li>
            </ul>
            <a-button :type="p.featured ? 'primary' : 'default'" block @click="$router.push('/register')">
              立即购买
            </a-button>
          </div>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="footer">
      <div class="container">
        <p>ShieldFlow © 2026 企业级自建CDN系统</p>
      </div>
    </footer>
  </div>
</template>

<script setup>
import {
  SafetyOutlined, FireOutlined, ThunderboltOutlined, RobotOutlined,
  GlobalOutlined, SafetyCertificateOutlined, LineChartOutlined, SyncOutlined,
} from '@ant-design/icons-vue'

const features = [
  { icon: 'SafetyOutlined', title: 'DDoS防护', desc: 'eBPF四层+七层CC防护，T级防护能力，自动封禁恶意IP' },
  { icon: 'FireOutlined', title: 'WAF语义分析', desc: 'AI驱动的语义WAF，精准识别SQL注入/XSS/RCE等17类攻击' },
  { icon: 'ThunderboltOutlined', title: 'CDN加速', desc: '二级缓存架构，智能回源，HTTP/2&3支持，Gzip/Brotli压缩' },
  { icon: 'RobotOutlined', title: 'AI智能防护', desc: '敏感词检测、语义WAF、威胁情报，多模型支持' },
  { icon: 'GlobalOutlined', title: '多DNS支持', desc: 'Cloudflare/阿里云/腾讯云DNSPod，自动CNAME同步' },
  { icon: 'SafetyCertificateOutlined', title: 'SSL证书', desc: 'ACME自动申请，Let\'s Encrypt/ZeroSSL，HTTP/DNS验证' },
  { icon: 'LineChartOutlined', title: '实时监控', desc: '访问日志/攻击日志/流量统计/地理分布，ClickHouse存储' },
  { icon: 'SyncOutlined', title: '缓存管理', desc: '文件刷新/目录刷新/文件预热，批量操作' },
]

const protectionLayers = [
  '黑白名单', 'CC防护', '访问控制', '区域限制',
  'Bot检测', '语义WAF', '转发源站'
]

const pricingPlans = [
  { name: '基础版', price: '¥99', features: ['100GB流量/月', '5个域名', '基础WAF', 'CC防护', 'HTTP/2'], featured: false },
  { name: '专业版', price: '¥299', features: ['500GB流量/月', '20个域名', 'AI-WAF', 'DDoS防护', 'HTTP/3', '缓存预热'], featured: true },
  { name: '企业版', price: '¥999', features: ['2TB流量/月', '无限域名', '全功能防护', 'eBPF DDoS', '独立日志服务器', 'VIP支持'], featured: false },
]
</script>

<style scoped>
.portal { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
.container { max-width: 1200px; margin: 0 auto; padding: 0 24px; }

.nav { position: fixed; top: 0; left: 0; right: 0; height: 64px; background: rgba(255,255,255,0.95); backdrop-filter: blur(8px); border-bottom: 1px solid #eee; z-index: 100; }
.nav-content { display: flex; align-items: center; justify-content: space-between; height: 64px; }
.logo-text { font-size: 24px; font-weight: 800; background: linear-gradient(135deg, #0066ff, #00ccff); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.nav-links { display: flex; gap: 32px; }
.nav-links a { color: #333; text-decoration: none; font-size: 15px; transition: color 0.2s; }
.nav-links a:hover { color: #0066ff; }

.hero { padding: 120px 0 80px; background: linear-gradient(135deg, #0a0e27 0%, #1a1f4e 100%); color: white; text-align: center; }
.hero h1 { font-size: 48px; font-weight: 800; margin-bottom: 16px; line-height: 1.3; }
.hero-desc { font-size: 20px; color: rgba(255,255,255,0.7); margin-bottom: 32px; }
.hero-actions { display: flex; gap: 16px; justify-content: center; margin-bottom: 64px; }
.hero-stats { display: flex; gap: 48px; justify-content: center; }
.stat { text-align: center; }
.stat .num { display: block; font-size: 32px; font-weight: 700; color: #00ccff; }
.stat .label { font-size: 14px; color: rgba(255,255,255,0.6); }

.section-title { text-align: center; font-size: 36px; font-weight: 700; margin-bottom: 48px; color: #1a1f4e; }
.features { padding: 80px 0; }
.feature-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 24px; }
.feature-card { text-align: center; padding: 32px 24px; border-radius: 12px; background: #f9fafe; transition: all 0.3s; }
.feature-card:hover { transform: translateY(-4px); box-shadow: 0 8px 24px rgba(0,0,0,0.08); }
.feature-icon { font-size: 40px; margin-bottom: 16px; }
.feature-card h3 { font-size: 18px; margin-bottom: 8px; }
.feature-card p { font-size: 14px; color: #666; line-height: 1.6; }

.protection { padding: 80px 0; background: #f5f7ff; }
.protection-flow { display: flex; flex-wrap: wrap; gap: 16px; justify-content: center; }
.flow-item { display: flex; align-items: center; gap: 8px; padding: 12px 20px; background: white; border-radius: 24px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.flow-num { width: 28px; height: 28px; border-radius: 50%; background: #0066ff; color: white; display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: 700; }
.flow-text { font-size: 15px; font-weight: 500; }

.pricing { padding: 80px 0; }
.pricing-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; max-width: 1000px; margin: 0 auto; }
.pricing-card { padding: 40px 32px; border-radius: 16px; border: 2px solid #eee; background: white; text-align: center; }
.pricing-card.featured { border-color: #0066ff; box-shadow: 0 8px 32px rgba(0,102,255,0.15); transform: scale(1.05); }
.pricing-card h3 { font-size: 24px; margin-bottom: 16px; }
.price .amount { font-size: 36px; font-weight: 800; color: #0066ff; }
.price .unit { font-size: 16px; color: #999; }
.pricing-card ul { list-style: none; padding: 24px 0; text-align: left; }
.pricing-card ul li { padding: 8px 0; color: #555; font-size: 14px; }
.pricing-card ul li::before { content: '✓'; color: #00cc66; font-weight: bold; margin-right: 8px; }

.footer { padding: 32px 0; background: #1a1f4e; color: rgba(255,255,255,0.6); text-align: center; }
</style>
