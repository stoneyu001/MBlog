package comments

import (
	"database/sql"
	"log"
	"time"
)

// CommentService 处理评论数据的服务
type CommentService struct {
	db *sql.DB
}

// NewCommentService 创建新的评论服务
func NewCommentService(db *sql.DB) *CommentService {
	return &CommentService{
		db: db,
	}
}

// Init 初始化评论系统数据库表
func (cs *CommentService) Init() error {
	// 创建评论表
	_, err := cs.db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id SERIAL PRIMARY KEY,
			article_id VARCHAR(100) NOT NULL,
			nickname VARCHAR(100) NOT NULL,
			email VARCHAR(100),
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ip_address VARCHAR(50),
			status VARCHAR(20) DEFAULT 'approved',
			reply_to INTEGER DEFAULT NULL,
			user_agent TEXT
		)
	`)
	if err != nil {
		log.Printf("创建评论表失败: %v", err)
		return err
	}

	// 创建索引
	_, err = cs.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_comments_article_id ON comments(article_id);
		CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at);
		CREATE INDEX IF NOT EXISTS idx_comments_status ON comments(status);
		CREATE INDEX IF NOT EXISTS idx_comments_reply_to ON comments(reply_to);
	`)
	if err != nil {
		log.Printf("创建评论索引失败: %v", err)
		return err
	}

	log.Println("评论系统数据库初始化成功")
	return nil
}

// AddComment 添加一条评论
func (cs *CommentService) AddComment(comment *Comment) (int, error) {
	// 设置客户端编码为UTF8
	_, err := cs.db.Exec("SET client_encoding = 'UTF8'")
	if err != nil {
		log.Printf("设置客户端编码失败: %v", err)
	}

	var id int
	err = cs.db.QueryRow(`
		INSERT INTO comments(article_id, nickname, email, content, created_at, ip_address, status, reply_to, user_agent)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, comment.ArticleID, comment.Nickname, comment.Email, comment.Content,
		time.Now(), comment.IPAddress, comment.Status, comment.ReplyTo, comment.UserAgent).Scan(&id)

	if err != nil {
		log.Printf("添加评论失败: %v", err)
		return 0, err
	}

	return id, nil
}

// GetCommentsByArticle 获取文章的评论
func (cs *CommentService) GetCommentsByArticle(articleID string) ([]Comment, error) {
	// 设置客户端编码为UTF8
	_, err := cs.db.Exec("SET client_encoding = 'UTF8'")
	if err != nil {
		log.Printf("设置客户端编码失败: %v", err)
	}

	log.Printf("执行查询，参数: articleID=%s", articleID)

	// 构建SQL
	query := `
		SELECT id, article_id, nickname, email, content, created_at, ip_address, status, reply_to, user_agent
		FROM comments
		WHERE article_id = $1 AND status = 'approved'
		ORDER BY created_at DESC
	`

	// 执行查询
	rows, err := cs.db.Query(query, articleID)
	if err != nil {
		log.Printf("查询评论失败: %v, SQL: %s, 参数: articleID=%s", err, query, articleID)
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(
			&comment.ID, &comment.ArticleID, &comment.Nickname, &comment.Email,
			&comment.Content, &comment.CreatedAt, &comment.IPAddress, &comment.Status,
			&comment.ReplyTo, &comment.UserAgent,
		)
		if err != nil {
			log.Printf("扫描评论数据失败: %v", err)
			continue
		}
		comments = append(comments, comment)
	}

	if err = rows.Err(); err != nil {
		log.Printf("遍历评论结果集时出错: %v", err)
	}

	log.Printf("查询完成，获取到 %d 条评论", len(comments))
	return comments, nil
}

// BuildCommentTree 构建评论树
func (cs *CommentService) BuildCommentTree(comments []Comment) []CommentResponse {
	commentMap := make(map[int]CommentResponse)
	var rootComments []CommentResponse

	// 首先转换所有评论到响应格式
	for _, comment := range comments {
		resp := CommentResponse{
			ID:        comment.ID,
			ArticleID: comment.ArticleID,
			Nickname:  comment.Nickname,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
			ReplyTo:   comment.ReplyTo,
			Replies:   []CommentResponse{},
		}
		commentMap[comment.ID] = resp
	}

	// 然后构建树结构
	for _, comment := range comments {
		if comment.ReplyTo == nil {
			// 这是一个根评论
			rootComments = append(rootComments, commentMap[comment.ID])
		} else {
			// 这是一个回复
			parent, exists := commentMap[*comment.ReplyTo]
			if exists {
				parent.Replies = append(parent.Replies, commentMap[comment.ID])
				commentMap[*comment.ReplyTo] = parent
			} else {
				// 如果父评论不存在（可能被删除或审核未通过），作为根评论处理
				rootComments = append(rootComments, commentMap[comment.ID])
			}
		}
	}

	return rootComments
}

// DeleteComment 删除评论
func (cs *CommentService) DeleteComment(id int) error {
	_, err := cs.db.Exec("DELETE FROM comments WHERE id = $1", id)
	if err != nil {
		log.Printf("删除评论失败: %v", err)
		return err
	}
	return nil
}

// UpdateCommentStatus 更新评论状态
func (cs *CommentService) UpdateCommentStatus(id int, status string) error {
	_, err := cs.db.Exec("UPDATE comments SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		log.Printf("更新评论状态失败: %v", err)
		return err
	}
	return nil
}
