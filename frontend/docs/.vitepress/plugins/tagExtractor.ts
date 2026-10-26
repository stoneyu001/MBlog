import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const chineseStopwordsPath = path.resolve(__dirname, './chineseStopwords.txt')
const defaultChineseStopwords = [
  '的', '了', '和', '是', '就', '都', '而', '及', '与', '这', '那', '有', '在', '中', '为',
  '对', '上', '到', '之', '或', '被', '所', '由', '我们', '你们', '他们', '这个', '那个',
  '这些', '那些', '一个', '一些', '因此', '因为', '如果', '虽然', '但是', '不过', '可以',
  '需要', '还有', '其实', '必须', '所以', '不同', '本文', '文章', '博客', '内容', '如下',
  '如何', '首先', '然后', '最后', '例如', '使用', '进行', '通过', '现在', '之后', '这里'
]

const fileStopwords = fs.existsSync(chineseStopwordsPath)
  ? fs.readFileSync(chineseStopwordsPath, 'utf8').split(/\r?\n/)
  : []
const chineseStopwords = new Set([...defaultChineseStopwords, ...fileStopwords].map(word => word.trim()).filter(Boolean))

function preprocessMarkdown(content: string): string {
  return content
    .replace(/^---[\s\S]*?---\s*/, '')
    .replace(/```[\s\S]*?```/g, '')
    .replace(/`[^`]+`/g, '')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
}

function extractBasicTagsFromFileName(fileName: string): string[] {
  const name = path.parse(fileName).name
    .replace(/^post_\d+_/, '')
    .replace(/^\d+[-_.]?/, '')
    .trim()
  return name.length >= 2 && name.length <= 16 ? [name] : []
}

function tokenize(text: string): string[] {
  const chunks = text.toLowerCase().match(/[\p{Script=Han}]+|[a-z][a-z0-9+.#-]{1,31}/gu) || []
  const tokens: string[] = []

  for (const chunk of chunks) {
    if (!/^[\p{Script=Han}]+$/u.test(chunk)) {
      tokens.push(chunk)
      continue
    }

    const chars = Array.from(chunk)
    for (let index = 0; index < chars.length - 1; index += 1) {
      tokens.push(chars[index] + chars[index + 1])
    }
    for (let index = 0; index < chars.length - 2; index += 1) {
      tokens.push(chars[index] + chars[index + 1] + chars[index + 2])
    }
  }

  return tokens.filter(token => !chineseStopwords.has(token) && !/^\d+$/.test(token))
}

export function extractTagsFromContent(content: string, limit = 5): string[] {
  const frequency = new Map<string, number>()
  for (const token of tokenize(preprocessMarkdown(content))) {
    frequency.set(token, (frequency.get(token) || 0) + 1)
  }

  return [...frequency.entries()]
    .filter(([token]) => token.length >= 2)
    .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0], 'zh-CN'))
    .slice(0, limit)
    .map(([token]) => token)
}

export function extractTags(content: string, fileName: string, limit = 5): string[] {
  const fileNameTags = extractBasicTagsFromFileName(fileName)
  const contentTags = extractTagsFromContent(content, Math.max(limit * 2, 8))
  return [...new Set([...fileNameTags, ...contentTags])].slice(0, limit)
}

export function getManualTags(frontmatter: Record<string, unknown> | undefined): string[] | null {
  if (!frontmatter || !Array.isArray(frontmatter.tags)) {
    return null
  }

  const tags = frontmatter.tags
    .filter((tag): tag is string => typeof tag === 'string')
    .map(tag => tag.trim())
    .filter(Boolean)

  return tags.length > 0 ? tags : null
}
