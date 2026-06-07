package comments

import (
	"blog/internal/repository"
)

// CommentService 处理评论业务逻辑
type CommentService struct {
	repo *repository.CommentRepo
}

// NewCommentService 创建评论服务
func NewCommentService(repo *repository.CommentRepo) *CommentService {
	return &CommentService{repo: repo}
}

// AddComment 添加评论
func (cs *CommentService) AddComment(comment *Comment) (int, error) {
	repoComment := &repository.Comment{
		ArticleID: comment.ArticleID,
		Nickname:  comment.Nickname,
		Email:     comment.Email,
		Content:   comment.Content,
		Status:    comment.Status,
		ReplyTo:   comment.ReplyTo,
		IPAddress: comment.IPAddress,
		UserAgent: comment.UserAgent,
	}
	return cs.repo.Insert(repoComment)
}

// GetCommentsByArticle 获取文章评论
func (cs *CommentService) GetCommentsByArticle(articleID string) ([]Comment, error) {
	repoComments, err := cs.repo.FindByArticle(articleID)
	if err != nil {
		return nil, err
	}
	comments := make([]Comment, len(repoComments))
	for i, rc := range repoComments {
		comments[i] = Comment{
			ID:        rc.ID,
			ArticleID: rc.ArticleID,
			Nickname:  rc.Nickname,
			Email:     rc.Email,
			Content:   rc.Content,
			CreatedAt: rc.CreatedAt,
			IPAddress: rc.IPAddress,
			Status:    rc.Status,
			ReplyTo:   rc.ReplyTo,
			UserAgent: rc.UserAgent,
		}
	}
	return comments, nil
}

// BuildCommentTree 构建评论树（纯业务逻辑，无 SQL）
func (cs *CommentService) BuildCommentTree(comments []Comment) []CommentResponse {
	commentMap := make(map[int]CommentResponse)
	var rootComments []CommentResponse

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

	for _, comment := range comments {
		if comment.ReplyTo == nil {
			rootComments = append(rootComments, commentMap[comment.ID])
		} else {
			parent, exists := commentMap[*comment.ReplyTo]
			if exists {
				parent.Replies = append(parent.Replies, commentMap[comment.ID])
				commentMap[*comment.ReplyTo] = parent
			} else {
				rootComments = append(rootComments, commentMap[comment.ID])
			}
		}
	}
	return rootComments
}

// DeleteComment 删除评论
func (cs *CommentService) DeleteComment(id int) error {
	return cs.repo.Delete(id)
}

// UpdateCommentStatus 更新评论状态
func (cs *CommentService) UpdateCommentStatus(id int, status string) error {
	return cs.repo.UpdateStatus(id, status)
}
