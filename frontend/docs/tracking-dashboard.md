# 埋点数据看板

::: warning 开发中
埋点数据分析功能正在开发中，敬请期待。
:::

这个页面展示了收集的埋点数据的基本统计信息。

<div id="dashboard-container">
  <div class="loading-overlay" v-if="loading">
    <div class="spinner"></div>
    <div>加载中...</div>
  </div>
  
  <div class="error-message" v-if="error">
    <p>{{ error }}</p>
    <button @click="fetchData">重试</button>
  </div>
  
  <div v-if="!loading && !error" class="dashboard-content">
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-title">总事件数</div>
        <div class="stat-value">{{ stats.totalEvents }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-title">页面浏览量</div>
        <div class="stat-value">{{ stats.pageViews }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-title">访客数</div>
        <div class="stat-value">{{ stats.uniqueVisitors }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-title">点击事件数</div>
        <div class="stat-value">{{ stats.clickEvents }}</div>
      </div>
    </div>
    
    <div class="charts-container">
      <div class="chart-wrapper">
        <h3>热门页面</h3>
        <div class="bar-chart">
          <div v-for="(page, index) in stats.topPages" :key="index" class="bar-item">
            <div class="bar-label">{{ formatPath(page.pagePath) }}</div>
            <div class="bar" :style="{ width: getBarWidth(page.count, getMaxPageCount()) }">
              <span class="bar-value">{{ page.count }}</span>
            </div>
          </div>
          <div v-if="stats.topPages.length === 0" class="no-data">暂无数据</div>
        </div>
      </div>
      
      <div class="chart-wrapper">
        <h3>热门元素</h3>
        <div class="bar-chart">
          <div v-for="(element, index) in stats.topElements" :key="index" class="bar-item">
            <div class="bar-label">{{ formatElementPath(element.elementPath) }}</div>
            <div class="bar" :style="{ width: getBarWidth(element.count, getMaxElementCount()) }">
              <span class="bar-value">{{ element.count }}</span>
            </div>
          </div>
          <div v-if="stats.topElements.length === 0" class="no-data">暂无数据</div>
        </div>
      </div>
      
      <div class="chart-wrapper full-width">
        <h3>小时分布</h3>
        <div class="hour-chart">
          <div v-for="hour in 24" :key="hour-1" class="hour-bar">
            <div class="hour-label">{{ hour-1 }}时</div>
            <div class="hour-value" 
                 :style="{ height: getHourHeight(getHourCount(hour-1)) }"
                 :title="`${getHourCount(hour-1)}个事件`">
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <div class="time-range">
      <select v-model="timeRange" @change="handleTimeRangeChange">
        <option value="7">最近7天</option>
        <option value="15">最近15天</option>
        <option value="30">最近30天</option>
      </select>
    </div>
  </div>
</div>

<script setup>
import { ref, onMounted, computed } from 'vue'

// 状态变量
const loading = ref(true)
const error = ref(null)
const timeRange = ref('7')
const stats = ref({
  totalEvents: 0,
  pageViews: 0,
  uniqueVisitors: 0,
  clickEvents: 0,
  topPages: [],
  topElements: [],
  eventsByHour: []
})

// 获取数据
const fetchData = async () => {
  loading.value = true
  error.value = null
  
  try {
    // 计算日期范围
    const endDate = new Date()
    const startDate = new Date()
    startDate.setDate(endDate.getDate() - parseInt(timeRange.value))
    
    // 格式化日期
    const formatDate = (date) => {
      return date.toISOString().split('T')[0]
    }
    
    // 发送请求
    const response = await fetch(`http://localhost:3000/api/tracking/analytics/overview?start_date=${formatDate(startDate)}&end_date=${formatDate(endDate)}`)
    
    if (!response.ok) {
      throw new Error('获取数据失败')
    }
    
    const data = await response.json()
    stats.value = {
      totalEvents: data.stats.total_events || 0,
      pageViews: data.stats.page_views || 0,
      uniqueVisitors: data.stats.unique_visitors || 0,
      clickEvents: data.stats.click_events || 0,
      topPages: data.top_pages || [],
      topElements: [],  // 后端暂未实现元素统计
      eventsByHour: []  // 后端暂未实现小时统计
    }
  } catch (err) {
    error.value = err.message || '获取数据失败，请稍后重试'
    console.error(err)
  } finally {
    loading.value = false
  }
}

// 处理时间范围变化
const handleTimeRangeChange = () => {
  fetchData()
}

// 格式化页面路径
const formatPath = (path) => {
  if (!path) return '/'
  return path.length > 25 ? path.substring(0, 22) + '...' : path
}

// 格式化元素路径
const formatElementPath = (path) => {
  if (!path) return '未知元素'
  return path.length > 25 ? path.substring(0, 22) + '...' : path
}

// 获取最大页面访问数
const getMaxPageCount = () => {
  if (!stats.value.topPages || stats.value.topPages.length === 0) return 1
  return Math.max(...stats.value.topPages.map(page => page.count))
}

// 获取最大元素点击数
const getMaxElementCount = () => {
  if (!stats.value.topElements || stats.value.topElements.length === 0) return 1
  return Math.max(...stats.value.topElements.map(element => element.count))
}

// 计算柱状图宽度
const getBarWidth = (value, max) => {
  return `${Math.max(5, (value / max) * 100)}%`
}

// 获取指定小时的事件数
const getHourCount = (hour) => {
  if (!stats.value.eventsByHour) return 0
  const hourStat = stats.value.eventsByHour.find(stat => stat.hour === hour)
  return hourStat ? hourStat.count : 0
}

// 获取小时柱状图高度
const getHourHeight = (count) => {
  if (!stats.value.eventsByHour || stats.value.eventsByHour.length === 0) return '0%'
  const max = Math.max(...stats.value.eventsByHour.map(hour => hour.count))
  if (max === 0) return '0%'
  return `${Math.max(5, (count / max) * 100)}%`
}

// 页面加载时获取数据
onMounted(() => {
  fetchData()
})
</script>

<style>
#dashboard-container {
  position: relative;
  padding: 20px;
  background-color: #f9f9f9;
  border-radius: 8px;
  min-height: 500px;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(255, 255, 255, 0.8);
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  z-index: 10;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #ddd;
  border-top: 4px solid #3498db;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 10px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error-message {
  text-align: center;
  padding: 30px;
  color: #e74c3c;
}

.error-message button {
  background-color: #3498db;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  margin-top: 10px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 30px;
}

.stat-card {
  background-color: white;
  border-radius: 6px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  text-align: center;
}

.stat-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 10px;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #333;
}

.charts-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.chart-wrapper {
  background-color: white;
  border-radius: 6px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.chart-wrapper h3 {
  margin-top: 0;
  margin-bottom: 20px;
  font-size: 16px;
  color: #333;
}

.full-width {
  grid-column: 1 / -1;
}

.bar-chart {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.bar-item {
  display: flex;
  align-items: center;
}

.bar-label {
  width: 120px;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bar {
  height: 24px;
  background-color: #3498db;
  border-radius: 4px;
  display: flex;
  align-items: center;
  padding: 0 8px;
  transition: width 0.3s ease;
  min-width: 30px;
}

.bar-value {
  color: white;
  font-size: 12px;
}

.hour-chart {
  display: flex;
  height: 200px;
  align-items: flex-end;
  gap: 4px;
}

.hour-bar {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  height: 100%;
}

.hour-label {
  font-size: 10px;
  color: #666;
  margin-top: 5px;
}

.hour-value {
  width: 100%;
  background-color: #3498db;
  border-radius: 3px 3px 0 0;
  transition: height 0.3s ease;
}

.time-range {
  margin-top: 30px;
  text-align: right;
}

.time-range select {
  padding: 8px 12px;
  border-radius: 4px;
  border: 1px solid #ddd;
}

.no-data {
  text-align: center;
  padding: 30px;
  color: #888;
  font-style: italic;
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .charts-container {
    grid-template-columns: 1fr;
  }
}
</style> 