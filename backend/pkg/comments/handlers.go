
	"github.com/gin-gonic/gin"
)

// RegisterHandlers 注册评论系统相关的路由处理函数
func (cs *CommentService) RegisterHandlers(r *gin.Engine) {
	// 创建评论
	r.POST("/api/comments", cs.handleAddComment)

	// 获取文章评论
	r.GET("/api/comments/:articleId", cs.handleGetCommentsByArticle)

	// 管理API，可以增加授权中间件
	admin := r.Group("/api/admin/comments")
	{
		admin.DELETE("/:id", cs.handleDeleteComment)
		admin.PUT("/:id/status", cs.handleUpdateCommentStatus)
	}
}

func (cs *CommentService) handleAddComment(c *gin.Context) {
	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("绑定评论请求失败: %v", err)

		// 提供详细的字段验证错误
		errorMsg := "请求参数错误"
		var missingFields []string

		if req.ArticleID == "" {
			missingFields = append(missingFields, "article_id")
		}
		if req.Nickname == "" {
			missingFields = append(missingFields, "nickname")
		}
		if req.Content == "" {
			missingFields = append(missingFields, "content")
		}

		if len(missingFields) > 0 {
			errorMsg = fmt.Sprintf("缺少必填字段: %s", strings.Join(missingFields, ", "))
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   errorMsg,
			"details": err.Error(),
		})
		return
	}

	// 记录文章ID，便于调试中文问题
	log.Printf("收到评论请求，文章ID: %s", req.ArticleID)

	// 创建评论对象
	comment := &Comment{
		ArticleID: req.ArticleID,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Content:   req.Content,
		Status:    "approved", // 默认状态为已批准
		ReplyTo:   req.ReplyTo,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}

	// 添加评论
	id, err := cs.AddComment(comment)
	if err != nil {
		log.Printf("添加评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论保存失败"})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"message": "评论已成功添加",
	})
}

// handleGetCommentsByArticle 处理获取文章评论的请求
func (cs *CommentService) handleGetCommentsByArticle(c *gin.Context) {
	// 获取并处理articleId参数
	articleID := c.Param("articleId")
	if articleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文章ID不能为空"})
		return
	}

	// 添加请求URL日志
	log.Printf("评论请求URL: %s, 参数: %+v", c.Request.URL.String(), c.Request.URL.Query())

	// 尝试解码URL编码的articleID
	decodedID, err := url.QueryUnescape(articleID)
	if err == nil && decodedID != articleID {
		articleID = decodedID
		log.Printf("文章ID已解码: %s", articleID)
	}

	// 记录所有转换前后的值，帮助调试
	log.Printf("文章ID处理过程: 原始=%s, 解码后=%s", c.Param("articleId"), articleID)

	// 清理articleID，确保不包含非法字符
	// 这一步可能在前端已经做了，但为了安全起见再做一次
	cleanID := strings.TrimSpace(articleID)
	if cleanID != articleID {
		log.Printf("文章ID已清理: %s -> %s", articleID, cleanID)
		articleID = cleanID
	}

	log.Printf("最终使用的文章ID: %s", articleID)
	log.Printf("查询SQL将使用: SELECT * FROM comments WHERE article_id = '%s' AND status = 'approved'", articleID)

	// 获取评论
	comments, err := cs.GetCommentsByArticle(articleID)
	if err != nil {
		log.Printf("获取评论失败: %v, SQL参数: articleID=%s", err, articleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论失败"})
		return
	}

	log.Printf("成功获取评论，文章ID: %s，评论数量: %d", articleID, len(comments))

	// 构建评论树
	commentTree := cs.BuildCommentTree(comments)

	// 返回评论树
	c.JSON(http.StatusOK, commentTree)
}

// handleDeleteComment 处理删除评论的请求
func (cs *CommentService) handleDeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID"})
		return
	}

	if err := cs.DeleteComment(id); err != nil {
		log.Printf("删除评论失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除评论失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论已成功删除"})
}

// handleUpdateCommentStatus 处理更新评论状态的请求
func (cs *CommentService) handleUpdateCommentStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 验证状态值
	status := strings.ToLower(req.Status)
	if status != "approved" && status != "pending" && status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态值"})
		return
	}

	if err := cs.UpdateCommentStatus(id, status); err != nil {
		log.Printf("更新评论状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新评论状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论状态已更新"})
}
