// 评论存储服务
const STORAGE_KEY = 'mblog_comment_user';

/**
 * 保存评论者信息到本地存储
 * @param {object} userInfo - 包含昵称和邮箱的对象
 */
export function saveCommentUser(userInfo) {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(userInfo));
    }
  } catch (error) {
    console.error('保存评论用户信息失败:', error);
  }
}

/**
 * 从本地存储获取评论者信息
 * @returns {object|null} 包含昵称和邮箱的对象，如果没有则返回null
 */
export function getCommentUser() {
  try {
    if (typeof localStorage !== 'undefined') {
      const data = localStorage.getItem(STORAGE_KEY);
      return data ? JSON.parse(data) : null;
    }
  } catch (error) {
    console.error('获取评论用户信息失败:', error);
  }
  return null;
} 