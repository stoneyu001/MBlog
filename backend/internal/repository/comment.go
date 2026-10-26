package repository

import (
	"database/sql"
	"errors"
	"log"
	"time"
)

var ErrCommentNotFound = errors.New("评论不存在")

type Comment struct {
	ID        int       `json:"id"`
	ArticleID string    `json:"article_id"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	Status    string    `json:"status"`
	ReplyTo   *int      `json:"reply_to"`
	UserAgent string    `json:"user_agent"`
}

type CommentFilter struct {
	Status    string
	ArticleID string
	Query     string
	Limit     int
	Offset    int
}

type CommentRepo struct {
	db *sql.DB
}

func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Insert(comment *Comment) (int, error) {
	var id int
	err := r.db.QueryRow(`
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

func (r *CommentRepo) FindByArticle(articleID string) ([]Comment, error) {
	rows, err := r.db.Query(`
		SELECT id, article_id, nickname, COALESCE(email, ''), content,
			created_at AT TIME ZONE 'Asia/Shanghai' AS created_at,
			COALESCE(ip_address, ''), status, reply_to, COALESCE(user_agent, '')
		FROM comments
		WHERE article_id = $1 AND status = 'approved'
		ORDER BY created_at ASC
		LIMIT 500
	`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]Comment, 0)
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (r *CommentRepo) FindByID(id int) (Comment, error) {
	comment, err := scanComment(r.db.QueryRow(`
		SELECT id, article_id, nickname, COALESCE(email, ''), content,
			created_at AT TIME ZONE 'Asia/Shanghai' AS created_at,
			COALESCE(ip_address, ''), status, reply_to, COALESCE(user_agent, '')
		FROM comments WHERE id = $1
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, ErrCommentNotFound
	}
	return comment, err
}

func (r *CommentRepo) FindAll(filter CommentFilter) ([]Comment, int64, error) {
	var total int64
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM comments
		WHERE ($1::text = '' OR status = $1)
		  AND ($2::text = '' OR article_id ILIKE '%' || $2 || '%')
		  AND ($3::text = '' OR nickname ILIKE '%' || $3 || '%' OR content ILIKE '%' || $3 || '%')
	`, filter.Status, filter.ArticleID, filter.Query).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(`
		SELECT id, article_id, nickname, COALESCE(email, ''), content,
			created_at AT TIME ZONE 'Asia/Shanghai' AS created_at,
			COALESCE(ip_address, ''), status, reply_to, COALESCE(user_agent, '')
		FROM comments
		WHERE ($1::text = '' OR status = $1)
		  AND ($2::text = '' OR article_id ILIKE '%' || $2 || '%')
		  AND ($3::text = '' OR nickname ILIKE '%' || $3 || '%' OR content ILIKE '%' || $3 || '%')
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, filter.Status, filter.ArticleID, filter.Query, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	comments := make([]Comment, 0, filter.Limit)
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, 0, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

func (r *CommentRepo) CountByStatus() (map[string]int64, error) {
	counts := map[string]int64{"pending": 0, "approved": 0, "rejected": 0}
	rows, err := r.db.Query(`SELECT status, COUNT(*) FROM comments GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *CommentRepo) Delete(id int) error {
	result, err := r.db.Exec("DELETE FROM comments WHERE id = $1 OR reply_to = $1", id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (r *CommentRepo) UpdateStatus(id int, status string) error {
	result, err := r.db.Exec("UPDATE comments SET status = $1 WHERE id = $2", status, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

type commentScanner interface {
	Scan(dest ...any) error
}

func scanComment(scanner commentScanner) (Comment, error) {
	var comment Comment
	err := scanner.Scan(
		&comment.ID,
		&comment.ArticleID,
		&comment.Nickname,
		&comment.Email,
		&comment.Content,
		&comment.CreatedAt,
		&comment.IPAddress,
		&comment.Status,
		&comment.ReplyTo,
		&comment.UserAgent,
	)
	return comment, err
}
