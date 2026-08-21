---
layout: home

hero:
  name: "StoneYu Blog"
  text: "把实践写成可再次抵达的路径"
  tagline: 记录技术现场、生活观察，以及那些值得反复翻阅的思考。
  actions:
    - theme: brand
      text: 阅读技术笔记
      link: /tech/💻技术栈志
    - theme: alt
      text: 翻阅生活随笔
      link: /life/🍵生活拾撷

features:
  - title: 工程札记
    details: 记录实现过程、故障复盘与技术取舍，让一次解决问题的经验能够被下一次检索到。
  - title: 生活切片
    details: 收集阅读、游戏、睡眠与日常体验。它们不追求标准答案，只保留当时真实的观察。
  - title: 可持续的小站
    details: 站点由 VitePress、Go、PostgreSQL 与 Docker 组成；发布和运行相互隔离，后台专注访问分析与评论管理。

---

<section class="home-journal">
  <p class="home-journal__kicker">RECENT NOTES</p>
  <div class="home-journal__heading">
    <div>
      <h2>最近整理</h2>
      <p>从归档里挑几篇，作为进入这座小站的入口。</p>
    </div>
    <a href="/tech/💻技术栈志">查看全部文章 →</a>
  </div>
  <ArticleList :limit="4" />
</section>
