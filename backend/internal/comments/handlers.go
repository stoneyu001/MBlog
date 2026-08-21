package comments

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"blog/internal/middleware"
	"blog/internal/repository"

	"github.com/gin-gonic/gin"
)

const (
	maxCommentBodyBytes = 32 * 1024
	maxArticleIDLength  = 200
	maxNicknameLength   = 100
	maxEmailLength      = 100
	maxContentLength    = 2000
)

func (cs *CommentService) RegisterHandlers(r *gin.Engine) {
	r.POST("/api/comments", middleware.RateLimit(6, time.Minute), cs.handleAddComment)
	r.GET("/api/comments/:articleId", cs.handleGetCommentsByArticle)

	admin := r.Group("/api/admin/comments", middleware.RequireAuth())
	admin.Use(middleware.NoStore())
	{
		admin.GET("", cs.handleListComments)
		admin.DELETE("/:id", cs.handleDeleteComment)
		admin.PUT("/:id/status", cs.handleUpdateCommentStatus)
	}
}

func (cs *CommentService) handleAddComment(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCommentBodyBytes)
	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请完整填写昵称和评论内容"})
		return
	}

	req.ArticleID = strings.TrimSpace(req.ArticleID)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Email = strings.TrimSpace(req.Email)
	req.Content = strings.TrimSpace(req.Content)
	if req.Website != "" {
		// 对自动填写隐藏字段的请求返回普通成功，避免暴露反垃圾规则。
		c.JSON(http.StatusCreated, gin.H{"message": "评论已提交审核"})
		return
	}
	if err := validateCommentRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := &Comment{
		ArticleID: req.ArticleID,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Content:   req.Content,
		Status:    "pending",
		ReplyTo:   req.ReplyTo,
		IPAddress: c.ClientIP(),
		UserAgent: truncateRunes(c.Request.UserAgent(), 512),
	}

	id, err := cs.AddComment(comment)
	if err != nil {
		if errors.Is(err, ErrInvalidReply) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("添加评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论保存失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"status":  "pending",
		"message": "评论已提交审核",
	})
}

func (cs *CommentService) handleGetCommentsByArticle(c *gin.Context) {
	articleID := strings.TrimSpace(c.Param("articleId"))
	if articleID == "" || utf8.RuneCountInString(articleID) > maxArticleIDLength || hasInvalidControl(articleID, false) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文章标识无效"})
		return
	}

	comments, err := cs.GetCommentsByArticle(articleID)
	if err != nil {
		log.Printf("获取评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论读取失败"})
		return
	}
	c.JSON(http.StatusOK, cs.BuildCommentTree(comments))
}

func (cs *CommentService) handleListComments(c *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "pending")))
	if status == "all" {
		status = ""
	}
	if status != "" && status != "pending" && status != "approved" && status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论状态筛选无效"})
		return
	}

	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 50 {
		pageSize = 50
	}
	articleID := truncateRunes(strings.TrimSpace(c.Query("article_id")), maxArticleIDLength)
	query := truncateRunes(strings.TrimSpace(c.Query("query")), 100)

	items, total, err := cs.ListComments(repository.CommentFilter{
		Status:    status,
		ArticleID: articleID,
		Query:     query,
		Limit:     pageSize,
		Offset:    (page - 1) * pageSize,
	})
	if err != nil {
		log.Printf("读取评论管理列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论列表读取失败"})
		return
	}
	counts, err := cs.CountByStatus()
	if err != nil {
		log.Printf("读取评论状态统计失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论状态统计失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"status_counts": counts,
	})
}

func (cs *CommentService) handleDeleteComment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论 ID 无效"})
		return
	}

	if err := cs.DeleteComment(id); err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
			return
		}
		log.Printf("删除评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论已删除"})
}

func (cs *CommentService) handleUpdateCommentStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论 ID 无效"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择评论状态"})
		return
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "approved" && status != "pending" && status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论状态无效"})
		return
	}

	if err := cs.UpdateCommentStatus(id, status); err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
			return
		}
		log.Printf("更新评论状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论状态更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "评论状态已更新"})
}

func validateCommentRequest(req CommentRequest) error {
	if req.ArticleID == "" || utf8.RuneCountInString(req.ArticleID) > maxArticleIDLength {
		return errors.New("文章标识无效")
	}
	if req.Nickname == "" || utf8.RuneCountInString(req.Nickname) > maxNicknameLength {
		return errors.New("昵称应为 1 到 100 个字符")
	}
	contentLength := utf8.RuneCountInString(req.Content)
	if contentLength < 2 || contentLength > maxContentLength {
		return errors.New("评论内容应为 2 到 2000 个字符")
	}
	if utf8.RuneCountInString(req.Email) > maxEmailLength {
		return errors.New("邮箱地址过长")
	}
	if req.Email != "" {
		address, err := mail.ParseAddress(req.Email)
		if err != nil || !strings.EqualFold(address.Address, req.Email) {
			return errors.New("邮箱格式不正确")
		}
	}
	if req.ReplyTo != nil && *req.ReplyTo <= 0 {
		return errors.New("回复目标无效")
	}
	if hasInvalidControl(req.ArticleID, false) || hasInvalidControl(req.Nickname, false) ||
		hasInvalidControl(req.Email, false) || hasInvalidControl(req.Content, true) {
		return errors.New("评论包含不支持的控制字符")
	}
	return nil
}

func hasInvalidControl(value string, allowFormatting bool) bool {
	for _, char := range value {
		if !unicode.IsControl(char) {
			continue
		}
		if allowFormatting && (char == '\n' || char == '\r' || char == '\t') {
			continue
		}
		return true
	}
	return false
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
