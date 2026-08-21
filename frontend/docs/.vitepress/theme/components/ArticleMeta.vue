<script setup lang="ts">
import { computed } from 'vue'
import { useData } from 'vitepress'

const { page, frontmatter } = useData()

const tags = computed<string[]>(() => Array.isArray(frontmatter.value.tags)
  ? frontmatter.value.tags.filter((tag): tag is string => typeof tag === 'string' && tag.trim().length > 0)
  : [])

const readingTime = computed(() => {
  const value = Number(frontmatter.value.readingTime)
  return Number.isFinite(value) && value > 0 ? Math.ceil(value) : 0
})

const showMeta = computed(() => {
  const relativePath = page.value.relativePath || ''
  return frontmatter.value.articleMeta !== false
    && /^(life|tech)\/.+\.md$/.test(relativePath)
    && (tags.value.length > 0 || readingTime.value > 0)
})
</script>

<template>
  <aside v-if="showMeta" class="article-meta" aria-label="文章信息">
    <span v-if="readingTime" class="article-meta__reading">约 {{ readingTime }} 分钟阅读</span>
    <span v-if="readingTime && tags.length" class="article-meta__divider" aria-hidden="true"></span>
    <ul v-if="tags.length" class="article-meta__tags" aria-label="文章标签">
      <li v-for="tag in tags" :key="tag">{{ tag }}</li>
    </ul>
  </aside>
</template>

<style scoped>
.article-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
  margin: -0.2rem 0 1.8rem;
  color: var(--vp-c-text-3);
  font-family: var(--mblog-font-sans);
  font-size: 0.8rem;
}

.article-meta__divider {
  width: 1px;
  height: 0.9rem;
  background: var(--mblog-border-strong);
}

.article-meta__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.article-meta__tags li {
  padding: 0.15rem 0.55rem;
  color: var(--vp-c-text-2);
  background: var(--mblog-accent-soft);
  border: 1px solid color-mix(in srgb, var(--vp-c-brand-1) 18%, transparent);
  border-radius: 999px;
  line-height: 1.5;
}

@media (max-width: 520px) {
  .article-meta__divider {
    display: none;
  }

  .article-meta__reading {
    flex-basis: 100%;
  }
}
</style>
