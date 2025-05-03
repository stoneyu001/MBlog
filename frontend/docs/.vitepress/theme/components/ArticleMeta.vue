<script setup lang="ts">
import { useData } from 'vitepress'

const { page } = useData()
</script>

<template>
  <div class="article-tags" v-if="page.frontmatter.tags && page.frontmatter.tags.length">
    <span class="tag" v-for="tag in page.frontmatter.tags" :key="tag">
      {{ tag }}
    </span>
  </div>
</template>

<style scoped>
.article-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin: 0.5rem 0 1.5rem;
}

.tag {
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
  padding: 0.2rem 0.5rem;
  border-radius: 3px;
  background-color: var(--vp-c-bg-soft);
}
</style> 