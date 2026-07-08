import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import ECharts from 'vue-echarts'
import * as echarts from 'echarts'
import 'echarts/lib/chart/map'
import 'echarts/lib/chart/line'
import 'echarts/lib/chart/bar'
import 'echarts/lib/chart/pie'
import 'echarts/lib/component/legend'
import 'echarts/lib/component/title'
import 'echarts/lib/component/tooltip'
import 'echarts/lib/component/grid'
import 'echarts/lib/component/dataZoom'
import 'echarts/lib/component/visualMap'
import 'echarts/lib/component/geo'
// (Dashboard.vue 用 Pie 替代 Map,省去 china.json 依赖)

import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(Antd)
app.component('VChart', ECharts)
app.mount('#app')
