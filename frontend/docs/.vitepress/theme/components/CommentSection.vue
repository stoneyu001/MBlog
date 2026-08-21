<template>
  <section v-if="shouldShow" class="comment-section" aria-labelledby="comment-heading">
    <header class="comment-heading">
      <div>
        <p class="comment-kicker">DISCUSSION</p>
        <h2 id="comment-heading">评论与交流</h2>
      </div>
      <span v-if="!loading" class="comment-count">{{ totalComments }} 条</span>
    </header>

    <form class="comment-form" @submit.prevent="submitComment">
      <div class="comment-form__intro">
        <div>
          <h3>留下你的想法</h3>
          <p>昵称会被记住，邮箱仅用于本次提交且不会公开。</p>
        </div>
        <span>审核后显示</span>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label for="comment-nickname">昵称 <span aria-hidden="true">*</span></label>
          <input
            id="comment-nickname"
            v-model="formData.nickname"
            name="nickname"
            type="text"
            autocomplete="nickname"
            maxlength="100"
            required
            placeholder="怎么称呼你"
          />
        </div>
        <div class="form-group">
          <label for="comment-email">邮箱 <small>选填，不公开</small></label>
          <input
            id="comment-email"
            v-model="formData.email"
            name="email"
            type="email"
            autocomplete="email"
            maxlength="100"
            placeholder="name@example.com"
          />
        </div>
      </div>

      <div class="form-group">
        <div class="form-label-row">
          <label for="comment-content">评论内容 <span aria-hidden="true">*</span></label>
          <span>{{ formData.content.length }}/2000</span>
        </div>
        <textarea
          id="comment-content"
          v-model="formData.content"
          name="content"
          rows="5"
          minlength="2"
          maxlength="2000"
          required
          placeholder="围绕文章内容展开交流，会更容易得到回应。"
        ></textarea>
      </div>

      <input v-model="formData.website" class="comment-honeypot" name="website" tabindex="-1" autocomplete="off" aria-hidden="true" />

      <div class="form-actions">
        <button class="comment-button comment-button--primary" type="submit" :disabled="mainSubmitting">
          {{ mainSubmitting ? '正在提交…' : '提交评论' }}
        </button>
        <p v-if="mainError" class="form-message form-message--error" role="alert">{{ mainError }}</p>
        <p v-else-if="mainSuccess" class="form-message form-message--success" role="status">{{ mainSuccess }}</p>
      </div>
    </form>

    <div class="comments-list" :aria-busy="loading">
      <div class="comments-list__title">
        <h3>全部评论</h3>
        <button v-if="loadError" class="text-button" type="button" @click="loadComments">重新加载</button>
      </div>

      <div v-if="loading" class="comment-state" role="status">
        <span class="loading-dot" aria-hidden="true"></span>
        正在读取评论
      </div>
      <div v-else-if="loadError" class="comment-state comment-state--error" role="alert">
        {{ loadError }}
      </div>
      <div v-else-if="comments.length === 0" class="comment-state">
        暂无已公开评论。你可以成为第一位留言者。
      </div>

      <ol v-else class="comment-thread">
        <li v-for="comment in comments" :key="comment.id" class="comment-item">
          <article>
            <header class="comment-item__header">
              <div class="comment-author">
                <span class="comment-avatar" aria-hidden="true">{{ avatarText(comment.nickname) }}</span>
                <div>
                  <strong>{{ comment.nickname }}</strong>
                  <time :datetime="comment.created_at">{{ formatDate(comment.created_at) }}</time>
                </div>
              </div>
              <button class="text-button" type="button" @click="replyTo(comment)">回复</button>
            </header>
            <p class="comment-content">{{ comment.content }}</p>
          </article>

          <ol v-if="comment.replies?.length" class="reply-list">
            <li v-for="reply in comment.replies" :key="reply.id" class="reply-item">
              <header class="comment-item__header">
                <div class="comment-author">
                  <span class="comment-avatar comment-avatar--reply" aria-hidden="true">{{ avatarText(reply.nickname) }}</span>
                  <div>
                    <strong>{{ reply.nickname }}</strong>
                    <time :datetime="reply.created_at">{{ formatDate(reply.created_at) }}</time>
                  </div>
                </div>
              </header>
              <p class="comment-content">{{ reply.content }}</p>
            </li>
          </ol>

          <form v-if="replyingTo === comment.id" class="reply-form" @submit.prevent="submitReply(comment.id)">
            <div class="reply-form__heading">
              <strong>回复 {{ comment.nickname }}</strong>
              <button class="text-button" type="button" @click="cancelReply">取消</button>
            </div>
            <div class="form-row">
              <div class="form-group">
                <label :for="`reply-nickname-${comment.id}`">昵称 <span aria-hidden="true">*</span></label>
                <input
                  :id="`reply-nickname-${comment.id}`"
                  v-model="replyForm.nickname"
                  type="text"
                  autocomplete="nickname"
                  maxlength="100"
                  required
                />
              </div>
              <div class="form-group">
                <label :for="`reply-email-${comment.id}`">邮箱 <small>选填</small></label>
                <input
                  :id="`reply-email-${comment.id}`"
                  v-model="replyForm.email"
                  type="email"
                  autocomplete="email"
                  maxlength="100"
                />
              </div>
            </div>
            <div class="form-group">
              <div class="form-label-row">
                <label :for="`reply-content-${comment.id}`">回复内容 <span aria-hidden="true">*</span></label>
                <span>{{ replyForm.content.length }}/2000</span>
              </div>
              <textarea
                :id="`reply-content-${comment.id}`"
                v-model="replyForm.content"
                rows="3"
                minlength="2"
                maxlength="2000"
                required
                autofocus
              ></textarea>
            </div>
            <input v-model="replyForm.website" class="comment-honeypot" tabindex="-1" autocomplete="off" aria-hidden="true" />
            <div class="form-actions">
              <button class="comment-button comment-button--primary" type="submit" :disabled="replySubmitting">
                {{ replySubmitting ? '正在提交…' : '提交回复' }}
              </button>
              <p v-if="replyError" class="form-message form-message--error" role="alert">{{ replyError }}</p>
            </div>
          </form>
        </li>
      </ol>

      <p v-if="replySuccess" class="form-message form-message--success comments-notice" role="status">
        {{ replySuccess }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useData, useRoute } from 'vitepress'
import { getCommentUser, saveCommentUser } from './CommentStorage'

interface CommentItem {
  id: number
  article_id: string
  nickname: string
  content: string
  created_at: string
  reply_to?: number | null
  replies?: CommentItem[]
}

interface CommentFormData {
  nickname: string
  email: string
  content: string
  website: string
}

const route = useRoute()
const { frontmatter } = useData()
const apiBaseUrl = String(import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')

const comments = ref<CommentItem[]>([])
const loading = ref(false)
const loadError = ref('')
const mainSubmitting = ref(false)
const mainError = ref('')
const mainSuccess = ref('')
const replySubmitting = ref(false)
const replyError = ref('')
const replySuccess = ref('')
const replyingTo = ref<number | null>(null)
let activeRequest: AbortController | null = null

const emptyForm = (): CommentFormData => ({ nickname: '', email: '', content: '', website: '' })
const formData = ref<CommentFormData>(emptyForm())
const replyForm = ref<CommentFormData>(emptyForm())

const shouldShow = computed(() => {
  return frontmatter.value.comments !== false
    && frontmatter.value.section !== true
    && /^\/(life|tech)\/.+/.test(route.path)
})

const articleId = computed(() => {
  const configuredId = typeof frontmatter.value.commentId === 'string'
    ? frontmatter.value.commentId.trim()
    : ''
  if (configuredId) return Array.from(configuredId.replace(/\/+/g, '_')).slice(0, 200).join('')

  let path = route.path.replace(/\/$/, '') || '/index'
  try {
    path = decodeURIComponent(path)
  } catch {
    // 保留无法解码的原始路径，避免页面因此失去评论能力。
  }

  const legacyId = path
    .replace(/[^\w\u4e00-\u9fa5\-\/.]/g, '')
    .replace(/\/+/g, '_')
    .replace(/^_|_$/g, '') || 'index'
  return Array.from(legacyId).slice(0, 200).join('')
})

const totalComments = computed(() => comments.value.reduce(
  (total, comment) => total + 1 + (comment.replies?.length || 0),
  0
))

function loadUserInfo() {
  const savedUser = getCommentUser()
  if (savedUser?.nickname) {
    formData.value.nickname = savedUser.nickname
  }
}

function saveUserInfo(nickname: string) {
  saveCommentUser({ nickname: nickname.trim() })
}

async function readError(response: Response, fallback: string): Promise<string> {
  try {
    const body = await response.json()
    return typeof body?.error === 'string' ? body.error : fallback
  } catch {
    return fallback
  }
}

async function loadComments() {
  activeRequest?.abort()
  activeRequest = null
  if (!shouldShow.value) {
    comments.value = []
    return
  }

  const request = new AbortController()
  activeRequest = request
  loading.value = true
  loadError.value = ''

  try {
    const response = await fetch(`${apiBaseUrl}/api/comments/${encodeURIComponent(articleId.value)}`, {
      signal: request.signal,
      headers: { Accept: 'application/json' }
    })
    if (!response.ok) {
      throw new Error(await readError(response, '评论读取失败'))
    }
    const data = await response.json()
    if (!Array.isArray(data)) {
      throw new Error('评论数据格式异常')
    }
    comments.value = data
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') return
    comments.value = []
    loadError.value = error instanceof Error ? error.message : '评论读取失败，请稍后重试'
  } finally {
    if (activeRequest === request) {
      activeRequest = null
      loading.value = false
    }
  }
}

function validateForm(data: CommentFormData, action: '评论' | '回复'): string {
  if (!data.nickname.trim()) return '请填写昵称'
  if (!data.content.trim()) return `请填写${action}内容`
  if (data.content.trim().length < 2) return `${action}内容至少需要 2 个字符`
  return ''
}

async function submitComment() {
  mainError.value = validateForm(formData.value, '评论')
  mainSuccess.value = ''
  if (mainError.value) return

  mainSubmitting.value = true
  try {
    const response = await fetch(`${apiBaseUrl}/api/comments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        article_id: articleId.value,
        nickname: formData.value.nickname.trim(),
        email: formData.value.email.trim(),
        content: formData.value.content.trim(),
        website: formData.value.website
      })
    })
    if (!response.ok) {
      throw new Error(await readError(response, '评论提交失败'))
    }

    saveUserInfo(formData.value.nickname)
    formData.value.content = ''
    formData.value.website = ''
    mainSuccess.value = '评论已提交，审核通过后会显示在这里。'
  } catch (error) {
    mainError.value = error instanceof Error ? error.message : '评论提交失败，请稍后重试'
  } finally {
    mainSubmitting.value = false
  }
}

function replyTo(comment: CommentItem) {
  replyingTo.value = comment.id
  replyError.value = ''
  replySuccess.value = ''
  replyForm.value = {
    nickname: formData.value.nickname,
    email: formData.value.email,
    content: '',
    website: ''
  }
}

function cancelReply() {
  replyingTo.value = null
  replyError.value = ''
}

async function submitReply(parentId: number) {
  replyError.value = validateForm(replyForm.value, '回复')
  if (replyError.value) return

  replySubmitting.value = true
  try {
    const response = await fetch(`${apiBaseUrl}/api/comments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        article_id: articleId.value,
        nickname: replyForm.value.nickname.trim(),
        email: replyForm.value.email.trim(),
        content: replyForm.value.content.trim(),
        reply_to: parentId,
        website: replyForm.value.website
      })
    })
    if (!response.ok) {
      throw new Error(await readError(response, '回复提交失败'))
    }

    formData.value.nickname = replyForm.value.nickname.trim()
    formData.value.email = replyForm.value.email.trim()
    saveUserInfo(replyForm.value.nickname)
    replyingTo.value = null
    replySuccess.value = '回复已提交，审核通过后会显示。'
  } catch (error) {
    replyError.value = error instanceof Error ? error.message : '回复提交失败，请稍后重试'
  } finally {
    replySubmitting.value = false
  }
}

function formatDate(dateValue: string): string {
  const date = new Date(dateValue)
  if (Number.isNaN(date.getTime())) return dateValue
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function avatarText(nickname: string): string {
  return Array.from(nickname.trim())[0]?.toUpperCase() || '访'
}

watch(() => route.path, () => {
  replyingTo.value = null
  mainError.value = ''
  mainSuccess.value = ''
  replyError.value = ''
  replySuccess.value = ''
  loadComments()
})

onMounted(() => {
  loadUserInfo()
  loadComments()
})

onBeforeUnmount(() => activeRequest?.abort())
</script>

<style scoped>
.comment-section {
  margin-top: 3rem;
  padding-top: 2.25rem;
  border-top: 1px solid var(--mblog-border-strong);
  font-family: var(--mblog-font-sans);
}

.comment-heading,
.comment-form__intro,
.comment-item__header,
.comments-list__title,
.reply-form__heading,
.form-label-row,
.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.comment-heading h2 {
  margin: 0.1rem 0 0;
  border: 0;
  font-family: var(--mblog-font-display);
  font-size: clamp(1.55rem, 4vw, 2rem);
}

.comment-kicker {
  margin: 0;
  color: var(--vp-c-brand-1);
  font-size: 0.7rem;
  font-weight: 750;
  letter-spacing: 0.18em;
}

.comment-count {
  padding: 0.25rem 0.65rem;
  color: var(--vp-c-text-2);
  background: var(--mblog-accent-soft);
  border-radius: 999px;
  font-size: 0.76rem;
}

.comment-form,
.reply-form {
  margin-top: 1.35rem;
  padding: clamp(1rem, 3vw, 1.45rem);
  background: var(--mblog-paper-soft);
  border: 1px solid var(--mblog-border);
  border-radius: 14px;
  box-shadow: var(--mblog-shadow-sm);
}

.comment-form__intro {
  align-items: flex-start;
  margin-bottom: 1.2rem;
}

.comment-form__intro h3,
.comments-list__title h3 {
  margin: 0;
  border: 0;
  color: var(--vp-c-text-1);
  font-family: var(--mblog-font-display);
  font-size: 1.05rem;
}

.comment-form__intro p {
  margin: 0.25rem 0 0;
  color: var(--vp-c-text-3);
  font-size: 0.78rem;
}

.comment-form__intro > span {
  flex: 0 0 auto;
  color: var(--vp-c-brand-1);
  font-size: 0.72rem;
}

.form-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.9rem;
}

.form-group {
  margin-bottom: 0.95rem;
}

.form-group label,
.form-label-row {
  color: var(--vp-c-text-2);
  font-size: 0.78rem;
  font-weight: 650;
}

.form-group label {
  display: block;
  margin-bottom: 0.4rem;
}

.form-label-row label {
  margin: 0;
}

.form-label-row > span,
.form-group small {
  color: var(--vp-c-text-3);
  font-size: 0.7rem;
  font-weight: 400;
}

.form-group input,
.form-group textarea {
  width: 100%;
  box-sizing: border-box;
  padding: 0.72rem 0.82rem;
  color: var(--vp-c-text-1);
  background: var(--mblog-input-bg);
  border: 1px solid var(--mblog-border-strong);
  border-radius: 9px;
  font: inherit;
  font-size: 0.9rem;
  line-height: 1.55;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.form-group textarea {
  resize: vertical;
}

.form-group input:focus,
.form-group textarea:focus {
  border-color: var(--vp-c-brand-1);
  box-shadow: 0 0 0 3px var(--mblog-focus-ring);
  outline: none;
}

.comment-honeypot {
  position: absolute !important;
  left: -10000px !important;
  width: 1px !important;
  height: 1px !important;
  overflow: hidden !important;
}

.form-actions {
  justify-content: flex-start;
  min-height: 42px;
}

.comment-button {
  min-height: 42px;
  padding: 0.65rem 1.15rem;
  border: 1px solid transparent;
  border-radius: 9px;
  cursor: pointer;
  font: inherit;
  font-size: 0.83rem;
  font-weight: 700;
}

.comment-button--primary {
  color: #fffaf2;
  background: var(--vp-c-brand-1);
  box-shadow: 0 5px 14px color-mix(in srgb, var(--vp-c-brand-1) 25%, transparent);
}

.comment-button--primary:hover:not(:disabled) {
  background: var(--vp-c-brand-2);
}

.comment-button:disabled {
  cursor: wait;
  opacity: 0.65;
}

.form-message {
  margin: 0;
  font-size: 0.78rem;
}

.form-message--error,
.comment-state--error {
  color: var(--vp-c-danger-1);
}

.form-message--success {
  color: var(--vp-c-success-1);
}

.comments-list {
  margin-top: 2.2rem;
}

.comments-list__title {
  padding-bottom: 0.7rem;
  border-bottom: 1px solid var(--mblog-border);
}

.text-button {
  padding: 0.25rem 0;
  color: var(--vp-c-brand-1);
  background: transparent;
  border: 0;
  cursor: pointer;
  font: inherit;
  font-size: 0.78rem;
  font-weight: 650;
}

.text-button:hover {
  text-decoration: underline;
  text-underline-offset: 3px;
}

.comment-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.55rem;
  min-height: 130px;
  color: var(--vp-c-text-3);
  font-size: 0.85rem;
  text-align: center;
}

.loading-dot {
  width: 7px;
  height: 7px;
  background: var(--vp-c-brand-1);
  border-radius: 50%;
  animation: comment-pulse 1.2s ease-in-out infinite;
}

.comment-thread,
.reply-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.comment-item {
  padding: 1.35rem 0;
  border-bottom: 1px solid var(--mblog-border);
}

.comment-author {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.comment-avatar {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  color: var(--vp-c-brand-1);
  background: var(--mblog-accent-soft);
  border: 1px solid var(--mblog-border-strong);
  border-radius: 50%;
  font-family: var(--mblog-font-display);
  font-size: 0.85rem;
  font-weight: 700;
}

.comment-avatar--reply {
  width: 29px;
  height: 29px;
  font-size: 0.75rem;
}

.comment-author strong,
.comment-author time {
  display: block;
}

.comment-author strong {
  color: var(--vp-c-text-1);
  font-size: 0.85rem;
}

.comment-author time {
  margin-top: 0.15rem;
  color: var(--vp-c-text-3);
  font-size: 0.68rem;
}

.comment-content {
  margin: 0.75rem 0 0 2.75rem;
  color: var(--vp-c-text-2);
  font-family: var(--mblog-font-display);
  font-size: 0.95rem;
  line-height: 1.8;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.reply-list {
  margin: 1rem 0 0 2.75rem;
  padding-left: 1rem;
  border-left: 2px solid var(--mblog-border-strong);
}

.reply-item {
  padding: 0.8rem 0;
}

.reply-item + .reply-item {
  border-top: 1px dashed var(--mblog-border);
}

.reply-form {
  margin: 1rem 0 0 2.75rem;
  box-shadow: none;
}

.reply-form__heading {
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
  font-size: 0.82rem;
}

.comments-notice {
  margin-top: 1rem;
}

@keyframes comment-pulse {
  50% { opacity: 0.35; transform: scale(0.75); }
}

@media (max-width: 640px) {
  .comment-form__intro,
  .form-row {
    display: block;
  }

  .comment-form__intro > span {
    display: inline-block;
    margin-top: 0.55rem;
  }

  .comment-item__header {
    align-items: flex-start;
  }

  .comment-content {
    margin-left: 0;
  }

  .reply-list,
  .reply-form {
    margin-left: 0.6rem;
  }

  .form-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .comment-button {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .loading-dot {
    animation: none;
  }
}
</style>
