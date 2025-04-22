# 埋点示例页面

这个页面展示了无感埋点功能的各种使用方式。

## 自动埋点示例

以下链接和按钮会自动被埋点系统捕获点击事件：

<div class="example-section">
  <a href="/markdown-examples.html">点击查看Markdown示例</a>

  <button class="primary-btn">普通按钮</button>
  
  <button class="danger-btn">危险操作</button>
</div>

## 指令式埋点示例

以下元素使用`v-track`指令进行精确埋点：

<div class="example-section">
  <button v-track="{ action: 'submit', category: 'form' }" class="success-btn">
    提交表单
  </button>
  
  <span v-track class="clickable-text">
    可点击文本
  </span>
  
  <div v-track="{ action: 'banner_click', position: 'top' }" class="banner">
    推广横幅
  </div>
</div>

## 自定义事件埋点

点击下面的元素会触发自定义埋点事件：

<div class="example-section">
  <button id="custom-event-btn" class="info-btn">触发自定义事件</button>
</div>

<script setup>
import { onMounted } from 'vue'

onMounted(() => {
  // 获取全局跟踪器实例
  const tracker = window.__tracker
  
  if (tracker) {
    // 自定义事件绑定
    const customBtn = document.getElementById('custom-event-btn')
    if (customBtn) {
      customBtn.addEventListener('click', () => {
        tracker.track({
          eventType: 'CUSTOM',
          pagePath: window.location.pathname,
          metadata: {
            customEventId: 'demo-1',
            timestamp: new Date().toISOString()
          }
        })
        
        alert('自定义事件已触发，请在控制台查看')
      })
    }
  }
})
</script>

<style>
.example-section {
  margin: 20px 0;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

button {
  margin: 5px;
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.primary-btn {
  background-color: #4569d4;
  color: white;
}

.danger-btn {
  background-color: #d9534f;
  color: white;
}

.success-btn {
  background-color: #5cb85c;
  color: white;
}

.info-btn {
  background-color: #5bc0de;
  color: white;
}

.clickable-text {
  color: #4569d4;
  text-decoration: underline;
  cursor: pointer;
  margin: 0 10px;
}

.banner {
  background-color: #ffeb3b;
  color: #333;
  padding: 10px;
  text-align: center;
  margin-top: 10px;
  cursor: pointer;
}
</style> 