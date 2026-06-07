package repository

import (
	"database/sql"
	"log"
	"time"
)

// Comment 评论数据模型
type Comment struct {
	ID        int
	ArticleID string
	Nickname  string
	Email     string
	Content   string
	CreatedAt time.Time
	IPAddress string
	Status    string
	ReplyTo   *int
	UserAgent string
}

// CommentRepo 评论数据访问层
type CommentRepo struct {
	db *sql.DB
}

// NewCommentRepo 创建评论 Repository
func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

// Insert 添加评论，返回新 ID
func (r *CommentRepo) Insert(c *Comment) (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO comments(article_id, nickname, email, content, created_at, ip_address, status, reply_to, user_agent)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, c.ArticleID, c.Nickname, c.Email, c.Content,
		time.Now(), c.IPAddress, c.Status, c.ReplyTo, c.UserAgent).Scan(&id)
	if err != nil {
		log.Printf("添加评论失败: %v", err)
		return 0, err
	}
	return id, nil
}

// FindByArticle 获取文章的评论列表
func (r *CommentRepo) FindByArticle(articleID string) ([]Comment, error) {
	log.Printf("评论查询: articleID=%s", articleID)
	rows, err := r.db.Query(`
		SELECT id, article_id, nickname, email, content, created_at, ip_address, status, reply_to, user_agent
		FROM comments
		WHERE article_id = $1 AND status = 'approved'
		ORDER BY created_at DESC
	`, articleID)
	if err != nil {
		log.Printf("查询评论失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ArticleID, &c.Nickname, &c.Email,
			&c.Content, &c.CreatedAt, &c.IPAddress, &c.Status,
			&c.ReplyTo, &c.UserAgent); err != nil {
			log.Printf("扫描评论数据失败: %v", err)
			continue
		}
		comments = append(comments, c)
	}
	log.Printf("查询完成，获取到 %d 条评论", len(comments))
	return comments, rows.Err()
}

// Delete 删除评论
func (r *CommentRepo) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM comments WHERE id = $1", id)
	if err != nil {
		log.Printf("删除评论失败: %v", err)
	}
	return err
}

// UpdateStatus 更新评论状态
func (r *CommentRepo) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec("UPDATE comments SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		log.Printf("更新评论状态失败: %v", err)
	}
	return err
}
