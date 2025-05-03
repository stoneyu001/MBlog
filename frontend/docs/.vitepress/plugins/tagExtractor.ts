import natural from 'natural';
const { TfIdf } = natural;
import * as path from 'node:path';
import * as fs from 'node:fs';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// 加载额外的中文停用词
const chineseStopwordsPath = path.resolve(__dirname, './chineseStopwords.txt');
const defaultChineseStopwords = [
  '的', '了', '和', '是', '就', '都', '而', '及', '与', '这', '那', '有', '在', '中', '为',
  '对', '上', '个', '到', '之', '或', '被', '所', '由', '它', '我', '你', '他', '她', '我们',
  '你们', '他们', '这个', '那个', '这些', '那些', '一个', '一些', '如此', '因此', '因为', '如果',
  '虽然', '但是', '不过', '可以', '不能', '需要', '只要', '只有', '还有', '还是', '其实', '其它',
  '一定', '必须', '所以', '每个', '一样', '一直', '一般', '一种', '一条', '不同', '不断', '不要',
  '地方', '希望', '时候', '问题', '能够', '什么', '为了', '出来', '所有', '自己', '这样',
  '本文', '文章', '博客', '内容', '如下', '一下', '如何', '可能', '首先', '然后', '最后', '例如',
  '返回', '使用', '无法', '很多', '说明', '已经', '看到', '进行', '通过', '现在', '知道', '之后',
  '不会', '这里', '那里', '知道', '发现', '觉得', '认为', '看看', '遇到'
];

// 如果自定义停用词文件不存在，创建一个
if (!fs.existsSync(chineseStopwordsPath)) {
  fs.writeFileSync(chineseStopwordsPath, defaultChineseStopwords.join('\n'), 'utf8');
}

// 读取中文停用词
const chineseStopwords = fs.existsSync(chineseStopwordsPath) 
  ? fs.readFileSync(chineseStopwordsPath, 'utf8').split('\n')
  : defaultChineseStopwords;

// 预处理文本(去除Markdown标记和特殊符号)
function preprocessMarkdown(content: string): string {
  return content
    .replace(/```[\s\S]*?```/g, '') // 移除代码块
    .replace(/`[^`]+`/g, '')        // 移除行内代码
    .replace(/\[.*?\]\(.*?\)/g, '') // 移除链接
    .replace(/#+\s/g, '')           // 移除标题标记
    .replace(/\!\[.*?\]\(.*?\)/g, '')  // 移除图片
    .replace(/[*>_~-]/g, ' ')       // 移除其他 Markdown 符号
    .replace(/\s+/g, ' ');          // 合并多个空格为一个
}

// 从文章文件名和内容自动提取前期标签
function extractBasicTagsFromFileName(fileName: string): string[] {
  // 从文件名中提取类别，如 "post_1_数据库优化.md" => "数据库优化"
  const match = fileName.match(/_([^_]+)\.md$/);
  if (match && match[1]) {
    return [match[1]];
  }
  return [];
}

// 简单的中文分词函数作为备选
function simpleCut(text: string): string[] {
  // 使用正则表达式进行简单分词
  return text
    .replace(/[^\u4e00-\u9fa5a-zA-Z0-9]/g, ' ') // 保留中文、英文和数字
    .split(/\s+/)
    .filter(word => word.length > 0);
}

/**
 * 使用TF-IDF算法提取关键词
 * TF-IDF是一种评估文档中词语重要性的统计方法
 * 词频(TF)越高，文档频率(DF)越低，得分越高
 */
function extractKeywordsByTFIDF(content: string, limit: number = 5): string[] {
  // 预处理Markdown内容
  const plainText = preprocessMarkdown(content);
  
  // 直接使用简单分词方法
  const segments = simpleCut(plainText);
  
  // 过滤停用词和单字词
  const validSegments = segments.filter(word => 
    word.length > 1 && !chineseStopwords.includes(word) && !/^\d+$/.test(word)
  );
  
  // 使用TF-IDF计算关键词
  const tfidf = new TfIdf();
  
  // 将文档添加到TF-IDF计算器
  tfidf.addDocument(validSegments);
  
  // 获取文档中每个词的TF-IDF得分
  const terms: { term: string, tfidf: number }[] = [];
  tfidf.listTerms(0).forEach(item => {
    terms.push({ term: item.term, tfidf: item.tfidf });
  });
  
  // 按TF-IDF得分排序并获取前N个词
  return terms
    .sort((a, b) => b.tfidf - a.tfidf)
    .slice(0, limit)
    .map(item => item.term);
}

/**
 * 提取标签
 * 使用TF-IDF算法提取关键词作为标签
 */
export function extractTagsFromContent(content: string, limit: number = 5): string[] {
  // 使用TF-IDF提取关键词
  return extractKeywordsByTFIDF(content, limit);
}

// 主提取函数
export function extractTags(content: string, fileName: string, limit: number = 5): string[] {
  // 先尝试从文件名提取基础标签
  const fileNameTags = extractBasicTagsFromFileName(fileName);
  
  // 然后从内容中提取更多标签
  const contentTags = extractTagsFromContent(content, limit - fileNameTags.length);
  
  // 合并去重
  const allTags = [...new Set([...fileNameTags, ...contentTags])];
  
  return allTags.slice(0, limit); // 确保最多返回limit个标签
}

// 辅助函数：从文章URL提取文件名
export function getFileNameFromUrl(url: string): string {
  const parts = url.split('/');
  return parts[parts.length - 1];
}

// 从前言中提取标签（如果存在）
export function getManualTags(frontmatter: any): string[] | null {
  if (frontmatter && Array.isArray(frontmatter.tags) && frontmatter.tags.length > 0) {
    return frontmatter.tags;
  }
  return null;
} 