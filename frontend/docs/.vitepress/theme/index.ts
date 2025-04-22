import DefaultTheme from 'vitepress/theme'
import { type Theme } from 'vitepress'
import { onBeforeUnmount } from 'vue'
import './styles/custom.css' // 自定义样式
import createTrackingPlugin from '../plugins/tracking'

// 创建跟踪插件
const trackingPlugin = createTrackingPlugin({
  endpoint: '/api/tracking/batch',
  batchSize: 10,
  batchInterval: 5000,
  debug: process.env.NODE_ENV === 'development', // 开发环境下启用调试
  sampling: 1.0, // 100%采样
  excludePaths: [
    '/admin*', // 排除管理界面
  ],
  includeElementSelector: [
    'a', 
    'button',
    '.track-click', // 特别标记的元素
    '[data-track]'  // 带有data-track属性的元素
  ],
  enableAutoTrack: {
    pageview: true,
    click: true,
    exposure: false
  }
});

export default {
  extends: DefaultTheme,
  // 这里可以添加自定义主题配置
  enhanceApp({ app, router, siteData }) {
    // 注册跟踪插件
    trackingPlugin.install(router);
    
    // 为特定元素添加指令式埋点
    app.directive('track', {
      mounted(el, binding) {
        // 添加标记表示这个元素应该被追踪
        el.setAttribute('data-track', '');
        
        // 添加自定义元数据
        if (binding.value) {
          el.setAttribute('data-track-metadata', JSON.stringify(binding.value));
        }
      }
    });
  },
  setup() {
    // 在组件卸载时清理资源
    onBeforeUnmount(() => {
      trackingPlugin.dispose();
    });
  }
} satisfies Theme 