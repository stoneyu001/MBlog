import { defineConfig } from 'vitepress'
import type { DefaultTheme, PageData, TransformPageContext } from 'vitepress'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'
import { extractTags, getManualTags, getFileNameFromUrl } from './plugins/tagExtractor'

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
  lang: 'zh-CN',
  title: "StoneYu Blog",
  description: "share and learn",
  lastUpdated: true,
  
  // 添加自动标签提取和其他元数据处理
  transformPageData(pageData: PageData & { relativePath?: string }, ctx: TransformPageContext) {
    // 检查是否是文章页面（跳过索引页和其他特殊页面）
    const relativePath = pageData.relativePath || '';
    const isArticlePage = !relativePath.includes('index') && relativePath !== '';
    
    // 获取原始Markdown内容
    const rawContent = fs.readFileSync(
      path.resolve(__dirname, '..', relativePath),
      'utf-8'
    );
    
    if (isArticlePage && rawContent) {
      // 检查是否已经有手动指定的标签
      const hasManualTags = getManualTags(pageData.frontmatter);
      
      // 如果没有手动指定的标签，则自动提取
      if (!hasManualTags) {
        try {
          const fileName = path.basename(relativePath);
          const autoTags = extractTags(rawContent, fileName, 5);
          
          // 确保 frontmatter 对象存在
          if (!pageData.frontmatter) {
            pageData.frontmatter = {};
          }
          
          // 添加自动提取的标签
          pageData.frontmatter.tags = autoTags;
          
          // 自动生成摘要（如果没有手动提供）
          if (!pageData.frontmatter.description && !pageData.frontmatter.excerpt) {
            const plainText = rawContent
              .replace(/```[\s\S]*?```/g, '')
              .replace(/`[^`]+`/g, '')
              .replace(/\[.*?\]\(.*?\)/g, '')
              .replace(/#+\s/g, '')
              .replace(/\!\[.*?\]\(.*?\)/g, '')
              .replace(/[*>_~-]/g, ' ')
              .replace(/\s+/g, ' ');
            
            pageData.frontmatter.description = plainText.slice(0, 150) + (plainText.length > 150 ? '...' : '');
          }
          
          // 计算阅读时间（如果没有手动提供）
          if (!pageData.frontmatter.readingTime) {
            const wordsPerMinute = 200; // 中文约200字/分钟
            const contentLength = rawContent.length;
            pageData.frontmatter.readingTime = Math.ceil(contentLength / wordsPerMinute);
          }
        } catch (e) {
          console.error(`Error extracting tags for ${relativePath}:`, e);
        }
      }
    }
    
    return pageData;
  },
  
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    lastUpdated: {
      text: '最后更新于'
    },
    nav: [
      { text: '主页', link: '/' },
      { text: '🍵生活拾撷', link: '/life/🍵生活拾撷' },
      { text: '💻技术栈志', link: '/tech/💻技术栈志' }
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
        fields: ['title', 'content', 'tags'], // 添加标签到索引字段
        storeFields: ['title', 'href', 'tags'], // 返回字段也包含标签
        searchOptions: {
          prefix: true, // 前缀匹配
          fuzzy: 0.2, // 模糊匹配容错率
          boost: { title: 4, content: 1, tags: 3 } // 权重配置，标签权重高
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
    ],
    docFooter: {
      prev: "上一页", //Next page
      next: "下一页", //Previous page
    },
    //当前页面 On this page
    outlineTitle: "页面导航",
  }
})
