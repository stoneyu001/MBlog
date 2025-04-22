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
      return {
        text: path.parse(file).name,
        link: `/articles/${file}`
      }
    })
}

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "StoneYu Blog",
  description: "share and learn",
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: '文章', link: '/articles/' },
      { text: '埋点示例', link: '/tracking-example' },
      { text: '关于', link: '/about' }
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
    sidebar: [
      {
        text: '所有文章',
        items: getArticles(path.resolve(__dirname, '../articles'))
      },
      {
        text: '功能演示',
        items: [
          { text: '埋点示例', link: '/tracking-example' },
          { text: 'Markdown示例', link: '/markdown-examples' }
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/stoneyu001/MBlog' }
    ]
  }
})
