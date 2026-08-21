package comments

import (
	"errors"
	"sort"

	"blog/internal/repository"
)

var ErrInvalidReply = errors.New("只能回复同一文章中已公开的一级评论")

type CommentService struct {
	repo *repository.CommentRepo
}

func NewCommentService(repo *repository.CommentRepo) *CommentService {
	return &CommentService{repo: repo}
}

func (cs *CommentService) AddComment(comment *Comment) (int, error) {
	if comment.ReplyTo != nil {
		parent, err := cs.repo.FindByID(*comment.ReplyTo)
		if err != nil {
			if errors.Is(err, repository.ErrCommentNotFound) {
				return 0, ErrInvalidReply
			}
			return 0, err
		}
		if parent.ArticleID != comment.ArticleID || parent.Status != "approved" || parent.ReplyTo != nil {
			return 0, ErrInvalidReply
		}
	}

	return cs.repo.Insert(toRepositoryComment(*comment))
}

func (cs *CommentService) GetCommentsByArticle(articleID string) ([]Comment, error) {
	repoComments, err := cs.repo.FindByArticle(articleID)
	if err != nil {
		return nil, err
	}
	return fromRepositoryComments(repoComments), nil
}

func (cs *CommentService) ListComments(filter repository.CommentFilter) ([]Comment, int64, error) {
	items, total, err := cs.repo.FindAll(filter)
	if err != nil {
		return nil, 0, err
	}
	return fromRepositoryComments(items), total, nil
}

func (cs *CommentService) CountByStatus() (map[string]int64, error) {
	return cs.repo.CountByStatus()
}

// BuildCommentTree 只构建两级评论。孤立回复不会被错误提升为一级评论。
func (cs *CommentService) BuildCommentTree(comments []Comment) []CommentResponse {
	commentMap := make(map[int]*CommentResponse, len(comments))
	for _, comment := range comments {
		commentMap[comment.ID] = &CommentResponse{
			ID:        comment.ID,
			ArticleID: comment.ArticleID,
			Nickname:  comment.Nickname,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
			ReplyTo:   comment.ReplyTo,
			Replies:   []CommentResponse{},
		}
	}

	roots := make([]*CommentResponse, 0)
	for _, comment := range comments {
		current := commentMap[comment.ID]
		if comment.ReplyTo == nil {
			roots = append(roots, current)
			continue
		}

		parent, exists := commentMap[*comment.ReplyTo]
		if exists && parent.ReplyTo == nil {
			parent.Replies = append(parent.Replies, *current)
		}
	}

	sort.Slice(roots, func(left, right int) bool {
		return roots[left].CreatedAt.After(roots[right].CreatedAt)
	})
	for _, root := range roots {
		sort.Slice(root.Replies, func(left, right int) bool {
			return root.Replies[left].CreatedAt.Before(root.Replies[right].CreatedAt)
		})
	}

	result := make([]CommentResponse, len(roots))
	for index, root := range roots {
		result[index] = *root
	}
	return result
}

func (cs *CommentService) DeleteComment(id int) error {
	return cs.repo.Delete(id)
}

func (cs *CommentService) UpdateCommentStatus(id int, status string) error {
	return cs.repo.UpdateStatus(id, status)
}

func toRepositoryComment(comment Comment) *repository.Comment {
	return &repository.Comment{
		ID:        comment.ID,
		ArticleID: comment.ArticleID,
		Nickname:  comment.Nickname,
		Email:     comment.Email,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
		IPAddress: comment.IPAddress,
		Status:    comment.Status,
		ReplyTo:   comment.ReplyTo,
		UserAgent: comment.UserAgent,
	}
}

func fromRepositoryComments(items []repository.Comment) []Comment {
	comments := make([]Comment, len(items))
	for index, item := range items {
		comments[index] = Comment{
			ID:        item.ID,
			ArticleID: item.ArticleID,
			Nickname:  item.Nickname,
			Email:     item.Email,
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
			IPAddress: item.IPAddress,
			Status:    item.Status,
			ReplyTo:   item.ReplyTo,
			UserAgent: item.UserAgent,
		}
	}
	return comments
}
