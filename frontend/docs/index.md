---
layout: home

hero:
  name: "StoneYu Blog"
  text: "写代码，也记录生活。"
  tagline: 你好，我是 StoneYu。这里收藏我的技术实践、日常观察，和一路走来的思考。
  actions:
    - theme: brand
      text: 阅读技术笔记
      link: /tech/💻技术栈志
    - theme: alt
      text: 翻阅生活随笔
      link: /life/🍵生活拾撷

features:
  - title: 工程札记
    link: /tech/💻技术栈志
    linkText: 阅读技术笔记
    details: 记录实现过程、故障复盘与技术取舍，留下解决问题的思路。
  - title: 生活切片
    link: /life/🍵生活拾撷
    linkText: 翻阅生活随笔
    details: 收集阅读、游戏、睡眠与日常体验。它们不追求标准答案，只保留当时真实的观察。
  - title: 开源作品
    details: 从实际需求出发，做一些自己用得上的工具。代码与持续更新的项目，都放在 GitHub。
    link: https://github.com/stoneyu001
    linkText: 看看我的项目

---

<section class="home-journal">
  <p class="home-journal__kicker">RECENT NOTES</p>
  <div class="home-journal__heading">
    <div>
      <h2>最近文章</h2>
      <p>技术与日常，都在这里慢慢记录。</p>
    </div>
    <a href="/tech/💻技术栈志">查看全部文章 →</a>
  </div>
  <ArticleList :limit="4" />
</section>
