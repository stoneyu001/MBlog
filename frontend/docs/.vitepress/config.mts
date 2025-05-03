import { defineConfig } from 'vitepress'
import type { DefaultTheme } from 'vitepress'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// 扫描文章目录获取所有 .md 文件
function getArticles(articlesPath: string) {
  const files = fs.readdirSync(articlesPath)
  return files
    .filter(file => file.endsWith('.md'))
    .map(file => {
      const basePath = path.relative(path.resolve(__dirname, '..'), articlesPath)
      return {
        text: path.parse(file).name,
        link: `/${basePath}/${file}`
      }
    })
}

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "StoneYu Blog",
  description: "share and learn",
  lastUpdated: true,
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: '🍵生活拾撷', link: '/life/🍵生活拾撷' },
      { text: '💻技术栈志', link: '/tech/💻技术栈志' },
    ],
    search: {
      provider: 'local',
      options: {
        detailedView: true, // 显示完整结果
        locales: {
          zh: {
            translations: {
              button: {
                buttonText: '搜索文档',
                buttonAriaLabel: '搜索文档'
              },
              modal: {
                noResultsText: '未找到相关结果',
                resetButtonTitle: '清除查询条件',
                footer: {
                  selectText: '选择',
                  navigateText: '切换'
                }
              }
            }
          }
        },
        // @ts-ignore
        fields: ['title', 'content'], // 索引字段
        storeFields: ['title', 'href'], // 返回字段
        searchOptions: {
          prefix: true, // 前缀匹配
          fuzzy: 0.2, // 模糊匹配容错率
          boost: { title: 4, content: 1 } // 权重配置
        },
        // 中文分词优化
        tokenize: (text) => {
          return text
            .split(/[\s\-，。；：！？、]+/) // 基本中文分词
            .filter(term => term.length > 1) // 过滤短词
        }
      } as DefaultTheme.LocalSearchOptions
    },
    sidebar: {
      // 当用户在 `life` 目录或其子目录下时，显示这个侧边栏
      '/life/': [
        {
          text: '🍵生活拾撷',
          // collapsed: true, // 默认折叠
          items: getArticles(path.resolve(__dirname, '../life'))
        }
      ],
      // 当用户在 `tech` 目录或其子目录下时，显示这个侧边栏
      '/tech/': [
        {
          text: '💻技术栈志',
          // collapsed: true, // 默认折叠
          items: getArticles(path.resolve(__dirname, '../tech'))
        }
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/stoneyu001/MBlog' }
    ]
  }
})
