<template>
  <div class="comment-section">
    <h2 class="comment-title">评论</h2>
    
    <!-- 评论表单 -->
    <div class="comment-form">
      <h3>发表评论</h3>
      <div class="form-group">
        <label for="nickname">昵称 <span class="required">*</span></label>
        <input 
          type="text" 
          id="nickname" 
          v-model="formData.nickname" 
          placeholder="请输入您的昵称" 
          required
        />
      </div>
      
      <div class="form-group">
        <label for="email">邮箱</label>
        <input 
          type="email" 
          id="email" 
          v-model="formData.email" 
          placeholder="选填不公开"
        />
      </div>
      
      <div class="form-group">
        <label for="content">评论内容 <span class="required">*</span></label>
        <textarea 
          id="content" 
          v-model="formData.content" 
          placeholder="请输入评论内容" 
          rows="4" 
          required
        ></textarea>
      </div>
      
      <button 
        class="submit-btn" 
        @click="submitComment" 
        :disabled="isSubmitting"
      >
        {{ isSubmitting ? '提交中...' : '提交评论' }}
      </button>
      
      <div v-if="submitError" class="error-message">
        {{ submitError }}
      </div>
      <div v-if="submitSuccess" class="success-message">
        评论提交成功！
      </div>
    </div>
    
    <!-- 评论列表 -->
    <div class="comments-list">
      <h3>全部评论 ({{ comments.length }})</h3>
      
      <div v-if="loading" class="loading">
        正在加载评论...
      </div>
      
      <div v-else-if="loadError" class="error-message">
        {{ loadError }}
      </div>
      
      <div v-else-if="comments.length === 0" class="no-comments">
        暂无评论，快来发表第一条吧！
      </div>
      
      <div v-else>
        <div v-for="comment in comments" :key="comment.id" class="comment-item">
          <div class="comment-header">
            <strong class="comment-author">{{ comment.nickname }}</strong>
            <span class="comment-date">{{ formatDate(comment.created_at) }}</span>
          </div>
          
          <div class="comment-content">
            {{ comment.content }}
          </div>
          
          <div class="comment-actions">
            <button class="reply-btn" @click="replyTo(comment)">回复</button>
          </div>
          
          <!-- 回复评论 -->
          <div v-if="comment.replies && comment.replies.length > 0" class="replies">
            <div v-for="reply in comment.replies" :key="reply.id" class="reply-item">
              <div class="comment-header">
                <strong class="comment-author">{{ reply.nickname }}</strong>
                <span class="comment-date">{{ formatDate(reply.created_at) }}</span>
              </div>
              
              <div class="comment-content">
                {{ reply.content }}
              </div>
            </div>
          </div>
          
          <!-- 回复表单 -->
          <div v-if="replyingTo === comment.id" class="reply-form">
            <div class="form-group">
              <label for="reply-nickname">昵称 <span class="required">*</span></label>
              <input 
                type="text" 
                id="reply-nickname" 
                v-model="replyForm.nickname" 
                placeholder="请输入您的昵称" 
                required
              />
            </div>
            
            <div class="form-group">
              <label for="reply-email">邮箱</label>
              <input 
                type="email" 
                id="reply-email" 
                v-model="replyForm.email" 
                placeholder="选填不公开"
              />
            </div>
            
            <div class="form-group">
              <label for="reply-content">回复内容 <span class="required">*</span></label>
              <textarea 
                id="reply-content" 
                v-model="replyForm.content" 
                placeholder="请输入回复内容" 
                rows="3" 
                required
              ></textarea>
            </div>
            
            <div class="reply-actions">
              <button 
                class="submit-btn" 
                @click="submitReply" 
                :disabled="isSubmitting"
              >
                {{ isSubmitting ? '提交中...' : '提交回复' }}
              </button>
              <button class="cancel-btn" @click="cancelReply">取消</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue';
import { useRoute } from 'vitepress';
import { saveCommentUser, getCommentUser } from './CommentStorage';

const route = useRoute();

// 使用路径作为文章ID，确保中文路径也能正确处理
const articleId = computed(() => {
  // 移除末尾的斜杠，然后对路径进行编码转换
  const path = decodeURIComponent(route.path.replace(/\/$/, '') || '/index');
  
  // 创建一个更简单的标识符，避免特殊字符和完整路径
  // 移除所有表情符号和特殊字符，只保留基本的URL友好字符
  const simpleId = path
    .replace(/[^\w\u4e00-\u9fa5\-\/\.]/g, '') // 只保留字母、数字、中文、连字符、斜杠和点
    .replace(/\/+/g, '_')                     // 将斜杠替换为下划线
    .replace(/^_/, '')                        // 移除开头的下划线
    .replace(/_$/, '');                       // 移除结尾的下划线
  
  console.log('原始路径:', path, '简化的文章ID:', simpleId);
  return simpleId || 'index';
});

// API基础URL配置
const apiBaseUrl = 'http://localhost:3000'; // 开发环境
// const apiBaseUrl = ''; // 生产环境使用相对路径

const loading = ref(false);
const loadError = ref('');
const comments = ref([]);
const isSubmitting = ref(false);
const submitError = ref('');
const submitSuccess = ref(false);
const replyingTo = ref(null);

// 评论表单和回复表单数据
const formData = ref({
  nickname: '',
  email: '',
  content: ''
});

const replyForm = ref({
  nickname: '',
  email: '',
  content: ''
});

// 加载用户信息从本地存储
function loadUserInfo() {
  const savedUser = getCommentUser();
  if (savedUser) {
    formData.value.nickname = savedUser.nickname || '';
    formData.value.email = savedUser.email || '';
  }
}

// 保存用户信息到本地存储
function saveUserInfo() {
  if (formData.value.nickname) {
    saveCommentUser({
      nickname: formData.value.nickname,
      email: formData.value.email
    });
  }
}

// 加载评论
async function loadComments() {
  loading.value = true;
  loadError.value = '';
  
  try {
    console.log('获取评论，文章ID:', articleId.value);
    const encodedId = encodeURIComponent(articleId.value);
    console.log('编码后的文章ID:', encodedId, '请求URL:', `${apiBaseUrl}/api/comments/${encodedId}`);
    
    const response = await fetch(`${apiBaseUrl}/api/comments/${encodedId}`);
    
    console.log('评论API响应状态:', response.status, response.statusText);
    if (!response.ok) {
      const errorText = await response.text();
      console.error('评论API错误响应:', errorText);
      throw new Error('获取评论失败');
    }
    
    const data = await response.json();
    console.log('获取到评论数据:', data);
    console.log('获取到评论数据类型:', typeof data, '是否数组:', Array.isArray(data));
    
    // 处理null或undefined响应，转换为空数组
    if (data === null || data === undefined) {
      console.log('响应数据为null或undefined，使用空数组');
      comments.value = [];
    } else {
      comments.value = data;
    }
    
    console.log('获取到评论数量:', comments.value.length);
  } catch (error) {
    console.error('加载评论出错详细信息:', error);
    console.error('错误堆栈:', error.stack);
    loadError.value = '加载评论失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}

// 提交评论
async function submitComment() {
  if (!formData.value.nickname || !formData.value.content) {
    submitError.value = '请填写昵称和评论内容';
    return;
  }
  
  isSubmitting.value = true;
  submitError.value = '';
  submitSuccess.value = false;
  
  try {
    console.log('提交评论，文章ID:', articleId.value);
    
    const commentData = {
      article_id: articleId.value,
      nickname: formData.value.nickname,
      email: formData.value.email,
      content: formData.value.content
    };
    
    console.log('评论提交数据:', commentData);
    
    const response = await fetch(`${apiBaseUrl}/api/comments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(commentData)
    });
    
    console.log('评论提交响应状态:', response.status, response.statusText);
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error('评论提交错误响应:', errorText);
      throw new Error('提交评论失败');
    }
    
    const responseData = await response.json();
    console.log('评论提交响应数据:', responseData);
    
    // 保存用户信息
    saveUserInfo();
    
    // 清空内容留下昵称和邮箱
    formData.value.content = '';
    
    submitSuccess.value = true;
    
    // 重新加载评论
    await loadComments();
    
    // 3秒后隐藏成功提示
    setTimeout(() => {
      submitSuccess.value = false;
    }, 3000);
  } catch (error) {
    console.error('提交评论出错详细信息:', error);
    console.error('错误堆栈:', error.stack);
    submitError.value = '提交评论失败，请稍后重试';
  } finally {
    isSubmitting.value = false;
  }
}

// 回复评论
function replyTo(comment) {
  replyingTo.value = comment.id;
  replyForm.value = {
    nickname: formData.value.nickname,
    email: formData.value.email,
    content: ''
  };
}

// 取消回复
function cancelReply() {
  replyingTo.value = null;
}

// 提交回复
async function submitReply() {
  if (!replyForm.value.nickname || !replyForm.value.content) {
    submitError.value = '请填写昵称和回复内容';
    return;
  }
  
  isSubmitting.value = true;
  submitError.value = '';
  
  try {
    console.log('提交回复，文章ID:', articleId.value, '回复ID:', replyingTo.value);
    const response = await fetch(`${apiBaseUrl}/api/comments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        article_id: articleId.value,
        nickname: replyForm.value.nickname,
        email: replyForm.value.email,
        content: replyForm.value.content,
        reply_to: replyingTo.value
      })
    });
    
    if (!response.ok) {
      throw new Error('提交回复失败');
    }
    
    // 清除回复状态
    replyingTo.value = null;
    
    // 将回复表单的昵称和邮箱同步到主表单
    formData.value.nickname = replyForm.value.nickname;
    formData.value.email = replyForm.value.email;
    
    // 保存到本地存储
    saveUserInfo();
    
    // 重新加载评论
    await loadComments();
  } catch (error) {
    console.error('提交回复出错:', error);
    submitError.value = '提交回复失败，请稍后重试';
  } finally {
    isSubmitting.value = false;
  }
}

// 格式化日期
function formatDate(dateStr) {
  try {
    // 移除末尾的Z并替换T为空格
    const localTimeStr = dateStr.replace('Z', '').replace('T', ' ');
    
    // 直接从字符串中提取时间部分
    const [datePart, timePart] = localTimeStr.split(' ');
    const [hours, minutes] = timePart.split(':');
    
    // 只返回到分钟的格式
    return `${datePart} ${hours}:${minutes}`;
  } catch (error) {
    console.error('格式化日期出错:', error);
    return dateStr;
  }
}

// 监听路由变化
watch(() => route.path, () => {
  console.log('路由发生变化，重新加载评论');
  loadComments();
}, { immediate: true });

// 组件挂载时加载评论和用户信息
onMounted(() => {
  loadUserInfo();
  console.log('组件已挂载，当前路径:', route.path);
  console.log('处理后的文章ID:', articleId.value);
});
</script>

<style scoped>
.comment-section {
  margin-top: 2rem;
  padding-top: 2rem;
  border-top: 1px solid var(--vp-c-divider);
  max-width: 100%;
}

.comment-title {
  font-size: 1.5rem;
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
}

.comment-form {
  background-color: var(--vp-c-bg-soft);
  padding: 1.5rem;
  border-radius: 8px;
  margin-bottom: 2rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: var(--vp-c-text-1);
}

.required {
  color: var(--vp-c-danger);
}

input, textarea {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 4px;
  font-size: 1rem;
  transition: border-color 0.2s;
  background-color: var(--vp-c-bg);
  color: var(--vp-c-text-1);
}

input:focus, textarea:focus {
  outline: none;
  border-color: var(--vp-c-brand);
}

.submit-btn, .reply-btn, .cancel-btn {
  padding: 0.5rem 1rem;
  border-radius: 4px;
  font-size: 0.9rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.submit-btn {
  background-color: var(--vp-c-brand);
  color: white;
  border: none;
}

.submit-btn:hover {
  background-color: var(--vp-c-brand-dark);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.cancel-btn {
  background-color: var(--vp-c-bg-soft);
  color: var(--vp-c-text-2);
  border: 1px solid var(--vp-c-divider);
  margin-left: 0.5rem;
}

.cancel-btn:hover {
  background-color: var(--vp-c-bg-mute);
}

.reply-btn {
  background: none;
  border: none;
  color: var(--vp-c-brand);
  padding: 0;
  font-size: 0.9rem;
}

.reply-btn:hover {
  color: var(--vp-c-brand-dark);
  text-decoration: underline;
}

.error-message {
  color: var(--vp-c-danger);
  margin-top: 0.5rem;
}

.success-message {
  color: var(--vp-c-success);
  margin-top: 0.5rem;
}

.comments-list {
  margin-top: 2rem;
}

.comment-item {
  padding: 1rem 0;
  border-bottom: 1px solid var(--vp-c-divider);
}

.comment-item:last-child {
  border-bottom: none;
}

.comment-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.comment-author {
  color: var(--vp-c-text-1);
}

.comment-date {
  color: var(--vp-c-text-3);
  font-size: 0.9rem;
}

.comment-content {
  line-height: 1.6;
  margin-bottom: 0.5rem;
  white-space: pre-wrap;
  color: var(--vp-c-text-1);
}

.comment-actions {
  margin-top: 0.5rem;
}

.replies {
  margin-left: 2rem;
  margin-top: 1rem;
  padding-left: 1rem;
  border-left: 2px solid var(--vp-c-divider);
}

.reply-item {
  padding: 0.5rem 0;
}

.reply-form {
  margin-top: 1rem;
  margin-left: 2rem;
  padding: 1rem;
  background-color: var(--vp-c-bg-soft);
  border-radius: 4px;
}

.reply-actions {
  display: flex;
}

.loading {
  text-align: center;
  color: var(--vp-c-text-3);
  padding: 1rem 0;
}

.no-comments {
  text-align: center;
  color: var(--vp-c-text-3);
  padding: 2rem 0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .replies {
    margin-left: 1rem;
    padding-left: 0.5rem;
  }
  
  .reply-form {
    margin-left: 1rem;
  }
}
</style> 