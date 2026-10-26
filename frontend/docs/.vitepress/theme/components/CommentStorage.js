const STORAGE_KEY = 'mblog_comment_user_v2'
const LEGACY_STORAGE_KEY = 'mblog_comment_user'

function normalizeUser(value) {
  if (!value || typeof value !== 'object' || typeof value.nickname !== 'string') {
    return null
  }

  const nickname = value.nickname.trim().slice(0, 100)
  return nickname ? { nickname } : null
}

export function saveCommentUser(userInfo) {
  if (typeof localStorage === 'undefined') return

  try {
    const user = normalizeUser(userInfo)
    if (user) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(user))
    }
    localStorage.removeItem(LEGACY_STORAGE_KEY)
  } catch {
    // 隐私模式或存储配额不足时，评论本身仍可正常提交。
  }
}

export function getCommentUser() {
  if (typeof localStorage === 'undefined') return null

  try {
    const current = normalizeUser(JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null'))
    if (current) return current

    const legacy = normalizeUser(JSON.parse(localStorage.getItem(LEGACY_STORAGE_KEY) || 'null'))
    if (legacy) {
      saveCommentUser(legacy)
      return legacy
    }

    localStorage.removeItem(STORAGE_KEY)
    localStorage.removeItem(LEGACY_STORAGE_KEY)
  } catch {
    localStorage.removeItem(STORAGE_KEY)
    localStorage.removeItem(LEGACY_STORAGE_KEY)
  }

  return null
}
