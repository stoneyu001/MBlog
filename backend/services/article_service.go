package services

import (
	"database/sql"
	"fmt"
	"time"
)

type Article struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// 创建文章
func CreateArticle(db *sql.DB, title, content string) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO articles (title, content) 
		VALUES ($1, $2) 
		RETURNING id
	`, title, content).Scan(&id)
	return id, err
}

// 根据ID获取文章
func GetArticleByID(db *sql.DB, id int) (*Article, error) {
	row := db.QueryRow(`
		SELECT id, title, content, created_at 
		FROM articles 
		WHERE id = $1
	`, id)

	var article Article
	err := row.Scan(&article.ID, &article.Title, &article.Content, &article.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("文章不存在")
	}
	return &article, err
}
