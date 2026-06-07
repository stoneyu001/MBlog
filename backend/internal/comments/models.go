package comments

import (
	"time"
)

// Comment 代表一条评论
type Comment struct {
	ID        int       `json:"id"`
	ArticleID string    `json:"article_id"` // 文章标识符
	Nickname  string    `json:"nickname"`   // 评论者昵称
	Email     string    `json:"email"`      // 评论者邮箱
	Content   string    `json:"content"`    // 评论内容
	CreatedAt time.Time `json:"created_at"` // 创建时间
	IPAddress string    `json:"ip_address"` // IP地址
	Status    string    `json:"status"`     // 状态
	ReplyTo   *int      `json:"reply_to"`   // 回复的评论ID
	UserAgent string    `json:"user_agent"` // 用户代理
}

// CommentRequest 代表评论请求
type CommentRequest struct {
	ArticleID string `json:"article_id" binding:"required"` // 文章标识符
	Nickname  string `json:"nickname" binding:"required"`   // 评论者昵称
	Email     string `json:"email"`                         // 评论者邮箱，可选
	Content   string `json:"content" binding:"required"`    // 评论内容
	ReplyTo   *int   `json:"reply_to"`                      // 回复的评论ID
}

// CommentResponse 代表评论响应
type CommentResponse struct {
	ID        int               `json:"id"`
	ArticleID string            `json:"article_id"`
	Nickname  string            `json:"nickname"`
	Content   string            `json:"content"`
	CreatedAt time.Time         `json:"created_at"`
	ReplyTo   *int              `json:"reply_to"`
	Replies   []CommentResponse `json:"replies,omitempty"`
}
