import { createContentLoader } from 'vitepress'

export interface ArticleSummary {
  url: string
  title: string
  description: string
  date: string
  readingTime: number
  tags: string[]
  section: 'tech' | 'life'
}

function cleanMarkdown(source: string): string {
  return source
    .replace(/^---[\s\S]*?---\s*/, '')
    .replace(/```[\s\S]*?```/g, '')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/^#{1,6}\s*/gm, '')
    .replace(/[*>_~-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function normalizeDate(value: unknown): string {
  if (!value) return ''
  const parsed = new Date(String(value))
  return Number.isNaN(parsed.getTime()) ? String(value) : parsed.toISOString().slice(0, 10)
}

export default createContentLoader(['tech/*.md', 'life/*.md'], {
  includeSrc: true,
  transform(entries): ArticleSummary[] {
    return entries
      .filter(entry => entry.frontmatter.draft !== true && entry.frontmatter.section !== true)
      .map(entry => {
        const source = entry.src || ''
        const plainText = cleanMarkdown(source)
        const heading = source.replace(/^---[\s\S]*?---\s*/, '').match(/^#{1,3}\s*(.+)$/m)?.[1]
        const title = String(entry.frontmatter.title || heading || entry.url.split('/').pop() || '未命名文章')
        const description = String(entry.frontmatter.description || plainText.slice(0, 120))
        const tags = Array.isArray(entry.frontmatter.tags)
          ? entry.frontmatter.tags.filter((tag): tag is string => typeof tag === 'string').slice(0, 4)
          : []

        return {
          url: entry.url,
          title,
          description: `${description}${plainText.length > 120 && !entry.frontmatter.description ? '…' : ''}`,
          date: normalizeDate(entry.frontmatter.date),
          readingTime: Math.max(1, Math.ceil(plainText.length / 300)),
          tags,
          section: entry.url.startsWith('/life/') ? 'life' : 'tech'
        }
      })
      .sort((left, right) => {
        if (left.date !== right.date) {
          return right.date.localeCompare(left.date)
        }
        return left.title.localeCompare(right.title, 'zh-CN', { numeric: true })
      })
  }
})
