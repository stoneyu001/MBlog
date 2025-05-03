import DefaultTheme from 'vitepress/theme'
import { type Theme } from 'vitepress'
import { onBeforeUnmount, h } from 'vue'
import './styles/custom.css' // 自定义样式
import createTrackingPlugin from '../plugins/tracking'
import ArticleMeta from './components/ArticleMeta.vue' // 导入标签组件

// 创建跟踪插件
const trackingPlugin = createTrackingPlugin({
  endpoint: 'http://localhost:3000/api/tracking/batch',
  batchSize: 5,         // 减小批量大小，更频繁发送
  batchInterval: 2000,  // 减少等待时间到2秒
  debug: true,
  sampling: 1, // 100%采样率
  excludePaths: [
    '/admin*', // 排除管理界面
  ],
  includeElementSelector: [
    'a', 
    'button',
    '.track-click', // 特别标记的元素
    '[data-track]',  // 带有data-track属性的元素
    'input[type="submit"]', // 添加表单提交按钮
    'form'  // 添加表单元素
  ],
  enableAutoTrack: {
    pageview: true,
    click: true,
    exposure: true  // 启用曝光追踪
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

    // 全局注册标签组件
    app.component('ArticleMeta', ArticleMeta);
  },
  setup() {
    // 在组件卸载时清理资源
    onBeforeUnmount(() => {
      trackingPlugin.dispose();
    });
  },
  // 自定义布局，在文章标题后添加标签
  Layout() {
    // 使用默认主题的布局
    return h(DefaultTheme.Layout, null, {
      'doc-before': () => h(ArticleMeta)  // 改为 doc-before 插槽
    })
  }
} satisfies Theme 