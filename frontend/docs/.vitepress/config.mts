import { defineConfig } from 'vitepress'
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
      { text: '关于', link: '/about' }
    ],
    search: {
      provider: 'local'
    },
    sidebar: [
      {
        text: '所有文章',
        items: getArticles(path.resolve(__dirname, '../articles'))
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/stoneyu001/MBlog' }
    ]
  }
})
