import { defineConfig } from 'vitepress'
import type { DefaultTheme, PageData, TransformPageContext } from 'vitepress'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'
import { extractTags, getManualTags } from './plugins/tagExtractor'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const docsRoot = path.resolve(__dirname, '..')
const collator = new Intl.Collator('zh-CN', { numeric: true, sensitivity: 'base' })

function stripFrontmatter(content: string): string {
  return content.replace(/^---[\s\S]*?---\s*/, '')
}

function getArticleTitle(content: string, fileName: string): string {
  const frontmatterTitle = content.match(/^---[\s\S]*?^title:\s*["']?(.+?)["']?\s*$[\s\S]*?^---/m)?.[1]
  const headingTitle = stripFrontmatter(content).match(/^#{1,3}\s+(.+)$/m)?.[1]
  return (frontmatterTitle || headingTitle || path.parse(fileName).name).trim()
}

function hasFrontmatterFlag(content: string, key: string): boolean {
  const frontmatter = content.match(/^---([\s\S]*?)^---/m)?.[1] || ''
  return new RegExp(`^${key}:\\s*true\\s*$`, 'mi').test(frontmatter)
}

function getArticles(articlesPath: string) {
  const basePath = path.relative(docsRoot, articlesPath).split(path.sep).join('/')

  return fs.readdirSync(articlesPath)
    .filter(file => file.endsWith('.md'))
    .flatMap(file => {
      const content = fs.readFileSync(path.join(articlesPath, file), 'utf-8')
      if (hasFrontmatterFlag(content, 'draft') || hasFrontmatterFlag(content, 'section')) {
        return []
      }
      return [{
        text: getArticleTitle(content, file),
        link: `/${basePath}/${path.parse(file).name}`
      }]
    })
    .sort((left, right) => collator.compare(left.text, right.text))
}

function plainTextFromMarkdown(content: string): string {
  return stripFrontmatter(content)
    .replace(/```[\s\S]*?```/g, '')
    .replace(/`[^`]+`/g, '')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/[*>_~-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function tokenizeSearchText(text: string): string[] {
  const chunks = text.toLowerCase().match(/[\p{Script=Han}]+|[a-z0-9][a-z0-9_+.#-]*/gu) || []
  const tokens: string[] = []

  for (const chunk of chunks) {
    if (/^[\p{Script=Han}]+$/u.test(chunk)) {
      const chars = Array.from(chunk)
      tokens.push(...chars)
      for (let index = 0; index < chars.length - 1; index += 1) {
        tokens.push(chars[index] + chars[index + 1])
      }
    } else {
      tokens.push(chunk)
    }
  }

  return tokens
}

export default defineConfig({
  lang: 'zh-CN',
  title: 'StoneYu Blog',
  description: '记录技术实践、生活观察与持续学习的个人博客。',
  lastUpdated: true,
  sitemap: {
    hostname: 'https://www.stoneyu.fun'
  },

  head: [
    ['link', { rel: 'icon', href: '/logo.webp' }],
    ['meta', { name: 'theme-color', content: '#172426' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'StoneYu Blog' }],
    ['meta', { property: 'og:locale', content: 'zh_CN' }]
  ],

  vite: {
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:3000',
          changeOrigin: true
        }
      }
    }
  },

  transformPageData(pageData: PageData & { relativePath?: string }, _ctx: TransformPageContext) {
    const relativePath = (pageData.relativePath || '').split(path.sep).join('/')
    const isArticlePage = /^(life|tech)\/.+\.md$/.test(relativePath)

    if (!isArticlePage) {
      return pageData
    }

    const sourcePath = path.resolve(docsRoot, relativePath)
    if (!fs.existsSync(sourcePath)) {
      return pageData
    }

    const rawContent = fs.readFileSync(sourcePath, 'utf-8')
    const plainText = plainTextFromMarkdown(rawContent)
    pageData.frontmatter ||= {}

    if (!getManualTags(pageData.frontmatter)) {
      try {
        pageData.frontmatter.tags = extractTags(rawContent, path.basename(relativePath), 5)
      } catch (error) {
        console.warn(`无法为 ${relativePath} 自动提取标签`, error)
      }
    }

    if (!pageData.frontmatter.description && !pageData.frontmatter.excerpt && plainText) {
      pageData.frontmatter.description = `${plainText.slice(0, 150)}${plainText.length > 150 ? '…' : ''}`
    }

    if (!pageData.frontmatter.readingTime) {
      pageData.frontmatter.readingTime = Math.max(1, Math.ceil(plainText.length / 300))
    }

    return pageData
  },

  themeConfig: {
    lastUpdated: {
      text: '最后更新'
    },
    outline: {
      level: 'deep',
      label: '本页目录'
    },
    nav: [
      { text: '首页', link: '/' },
      { text: '技术栈志', link: '/tech/💻技术栈志' },
      { text: '生活拾撷', link: '/life/🍵生活拾撷' },
      { text: '访问数据', link: '/admin', target: '_self' }
    ],
    search: {
      provider: 'local',
      options: {
        detailedView: true,
        translations: {
          button: {
            buttonText: '搜索文章',
            buttonAriaLabel: '搜索文章'
          },
          modal: {
            displayDetails: '显示详细内容',
            resetButtonTitle: '清除搜索',
            backButtonTitle: '关闭搜索',
            noResultsText: '没有找到相关内容',
            footer: {
              selectText: '打开',
              selectKeyAriaLabel: '回车',
              navigateText: '切换',
              navigateUpKeyAriaLabel: '向上',
              navigateDownKeyAriaLabel: '向下',
              closeText: '关闭',
              closeKeyAriaLabel: 'Esc'
            }
          }
        },
        miniSearch: {
          options: {
            tokenize: tokenizeSearchText,
            processTerm: term => term.toLowerCase()
          },
          searchOptions: {
            prefix: true,
            fuzzy: 0.2,
            boost: { title: 5, titles: 3, text: 1 }
          }
        }
      } as DefaultTheme.LocalSearchOptions
    },
    sidebar: {
      '/life/': [
        {
          text: '生活拾撷',
          items: getArticles(path.resolve(docsRoot, 'life'))
        }
      ],
      '/tech/': [
        {
          text: '技术栈志',
          items: getArticles(path.resolve(docsRoot, 'tech'))
        }
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/stoneyu001/MBlog' }
    ],
    docFooter: {
      prev: '上一篇',
      next: '下一篇'
    },
    returnToTopLabel: '返回顶部',
    sidebarMenuLabel: '文章目录',
    darkModeSwitchLabel: '切换主题',
    logo: { src: '/logo.webp', width: 26, height: 26 },
    footer: {
      message: '保持好奇，持续记录。',
      copyright: '© 2025–2026 StoneYu Blog'
    }
  }
})
