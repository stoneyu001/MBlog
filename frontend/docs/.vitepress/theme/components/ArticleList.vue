<script setup lang="ts">
import { computed } from 'vue'
import { data as articles } from '../../data/posts.data.js'

const props = withDefaults(defineProps<{
  section?: 'all' | 'tech' | 'life'
  limit?: number
}>(), {
  section: 'all',
  limit: 0
})

const visibleArticles = computed(() => {
  const filtered = props.section === 'all'
    ? articles
    : articles.filter(article => article.section === props.section)
  return props.limit > 0 ? filtered.slice(0, props.limit) : filtered
})

function formatDate(date: string): string {
  if (!date) return ''
  const parsed = new Date(date)
  return Number.isNaN(parsed.getTime())
    ? date
    : new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' }).format(parsed)
}
</script>

<template>
  <div class="article-list" aria-label="文章列表">
    <a
      v-for="article in visibleArticles"
      :key="article.url"
      class="article-card"
      :href="article.url"
    >
      <div class="article-card__eyebrow">
        <span>{{ article.section === 'tech' ? '技术' : '生活' }}</span>
        <span>{{ article.readingTime }} 分钟阅读</span>
        <time v-if="article.date" :datetime="article.date">{{ formatDate(article.date) }}</time>
      </div>
      <h3>{{ article.title }}</h3>
      <p>{{ article.description }}</p>
      <div v-if="article.tags.length" class="article-card__tags" aria-label="文章标签">
        <span v-for="tag in article.tags" :key="tag">{{ tag }}</span>
      </div>
      <span class="article-card__more" aria-hidden="true">阅读全文 <span>→</span></span>
    </a>
    <p v-if="visibleArticles.length === 0" class="article-list__empty">这个栏目还在整理中。</p>
  </div>
</template>

<style scoped>
.article-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  margin: 1.5rem 0 2rem;
}

.article-card {
  position: relative;
  display: flex;
  min-height: 230px;
  flex-direction: column;
  padding: 1.3rem;
  color: var(--vp-c-text-1);
  background: var(--mblog-paper-soft);
  border: 1px solid var(--mblog-border);
  border-radius: 14px;
  box-shadow: var(--mblog-shadow-sm);
  text-decoration: none;
  transition: transform 180ms ease, border-color 180ms ease, box-shadow 180ms ease;
}

.article-card:hover {
  border-color: var(--vp-c-brand-1);
  box-shadow: var(--mblog-shadow-md);
  transform: translateY(-3px);
}

.article-card__eyebrow {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem 0.8rem;
  color: var(--vp-c-text-3);
  font-family: var(--mblog-font-sans);
  font-size: 0.75rem;
  letter-spacing: 0.04em;
}

.article-card h3 {
  margin: 0.8rem 0 0.55rem;
  border: 0;
  color: var(--vp-c-text-1);
  font-size: 1.15rem;
  line-height: 1.45;
}

.article-card p {
  display: -webkit-box;
  margin: 0;
  overflow: hidden;
  color: var(--vp-c-text-2);
  font-size: 0.92rem;
  line-height: 1.75;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.article-card__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.9rem;
}

.article-card__tags span {
  padding: 0.15rem 0.5rem;
  color: var(--vp-c-text-2);
  background: var(--mblog-accent-soft);
  border-radius: 999px;
  font-size: 0.72rem;
}

.article-card__more {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-top: auto;
  padding-top: 1rem;
  color: var(--vp-c-brand-1);
  font-family: var(--mblog-font-sans);
  font-size: 0.83rem;
  font-weight: 650;
}

.article-card__more span {
  transition: transform 180ms ease;
}

.article-card:hover .article-card__more span {
  transform: translateX(3px);
}

.article-list__empty {
  grid-column: 1 / -1;
  color: var(--vp-c-text-2);
}

@media (max-width: 700px) {
  .article-list {
    grid-template-columns: 1fr;
  }

  .article-card {
    min-height: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .article-card,
  .article-card__more span {
    transition: none;
  }
}
</style>
