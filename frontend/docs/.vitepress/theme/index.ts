import DefaultTheme from 'vitepress/theme'
import { type Theme } from 'vitepress'
import { onBeforeUnmount, h } from 'vue'
import './styles/custom.css'
import createTrackingPlugin from '../plugins/tracking'
import { getTrackingEndpoint } from '../plugins/config'
import ArticleMeta from './components/ArticleMeta.vue'
import ArticleList from './components/ArticleList.vue'
import CommentSection from './components/CommentSection.vue'

// 创建跟踪插件
const trackingPlugin = createTrackingPlugin({
  endpoint: getTrackingEndpoint(),
  batchSize: 10,
  batchInterval: 5000,
  debug: false,
  sampling: 1,
  excludePaths: [
    '/admin*',
    '/login*',
    '/analytics*'
  ],
  includeElementSelector: [
    'a',
    'button',
    '[role="button"]',
    '[data-track]'
  ],
  enableAutoTrack: {
    pageview: true,
    click: true,
    exposure: false
  }
});

export default {
  extends: DefaultTheme,
  enhanceApp({ app, router }) {
    if (typeof window !== 'undefined') {
      const scheduleTracking = (callback: () => void) => {
        if ('requestIdleCallback' in window) {
          window.requestIdleCallback(callback)
        } else {
          window.setTimeout(callback, 1000)
        }
      }
      scheduleTracking(() => trackingPlugin.install(router));
    }

    app.directive('track', {
      mounted(el, binding) {
        el.setAttribute('data-track', '');
        if (binding.value) {
          el.setAttribute('data-track-metadata', JSON.stringify(binding.value));
        }
      }
    });

    app.component('ArticleMeta', ArticleMeta);
    app.component('ArticleList', ArticleList);
    app.component('CommentSection', CommentSection);
  },
  setup() {
    onBeforeUnmount(() => {
      trackingPlugin.dispose();
    });
  },
  Layout() {
    return h(DefaultTheme.Layout, null, {
      'doc-before': () => h(ArticleMeta),
      'doc-after': () => h(CommentSection)
    })
  }
} satisfies Theme 
