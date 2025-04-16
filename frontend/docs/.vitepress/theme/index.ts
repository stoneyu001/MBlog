import DefaultTheme from 'vitepress/theme'
import { type Theme } from 'vitepress'
import './styles/custom.css' // 自定义样式

export default {
  extends: DefaultTheme,
  // 这里可以添加自定义主题配置
  enhanceApp({ app, router, siteData }) {
    // 注册全局组件或其他增强功能
  }
} satisfies Theme 