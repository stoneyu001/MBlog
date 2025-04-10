import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "StoneYu Blog",
  description: "share and learn",
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: '文章1', link: '/articles/' },
      { text: '关于', link: '/about' }
    ],
    search: {
      provider: 'local'
    },
    sidebar: [                     //动态侧边栏？
      {
        text: '目录',
        items: [
          { text: 'test', link: '/articles/test.md' },
          { text: '文明6', link: '/articles/文明6.md' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/stoneyu001/MBlog' }
    ]
  }
})
