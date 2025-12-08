package service

import (
	mysql "blog/dao/mysql"
	redis "blog/dao/redis"
	"blog/model"
	"blog/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrCommentNotFound           = errors.New("评论不存在")
	ErrCommentInvalidContent     = errors.New("评论不能为空")
	ErrPostIsDeleted             = errors.New("评论的文章已被删除")
	ErrCommentAlreadyLiked       = errors.New("已经点赞过此评论")
	ErrCommentNotLiked           = errors.New("还没有点赞此评论")
	ErrReplyToNonexistentComment = errors.New("回复的评论不存在")
	ErrUnauthorized              = errors.New("未授权操作")
)

type CommentService interface {
	// 评论基础功能
	CreateComment(ctx context.Context, req *CreateCommentRequest) (*model.Comment, error)
	GetComment(ctx context.Context, id uint) (*model.Comment, error)
	DeleteComment(ctx context.Context, id uint) error
	ListCommentsByPost(ctx context.Context, postID uint, page, size int) ([]*model.Comment, int64, error)
	ListCommentsByUser(ctx context.Context, userID uint, page, size int) ([]*model.Comment, int64, error)

	// 评论点赞功能
	LikeComment(ctx context.Context, commentID uint) error
	UnlikeComment(ctx context.Context, commentID uint) error
	GetCommentLikes(ctx context.Context, commentID uint) (uint, error)
	IsCommentLiked(ctx context.Context, commentID uint) (bool, error)

	// 评论回复功能
	CreateReply(ctx context.Context, req *CreateReplyRequest) (*model.Comment, error)
	ListReplies(ctx context.Context, commentID uint, page, size int) ([]*model.Comment, int64, error)
}

// 请求结构体
type CreateCommentRequest struct {
	PostID  uint   `json:"post_id" binding:"required"`
	Content string `json:"content" binding:"required,min=1,max=1000"`
}

type UpdateCommentRequest struct {
	Content *string `json:"content" binding:"omitempty,min=1,max=1000"`
}

type CreateReplyRequest struct {
	ParentID uint   `json:"parent_id" binding:"required"`
	PostID   uint   `json:"post_id" binding:"required"`
	Content  string `json:"content" binding:"required,min=1,max=1000"`
}

type commentService struct {
	// MySQL DAO
	commentSQL     mysql.CommentSQL     // 评论CRUD
	postSQL        mysql.PostSQL        // 更新帖子评论数
	userSQL        mysql.UserSQL        // 获取用户信息
	commentLikeSQL mysql.CommentLikeSQL // 评论点赞

	// Redis缓存
	commentCache redis.CommentCache // 评论计数和点赞缓存

	// 数据库
	db *gorm.DB // 事务管理
}

func NewCommentService(
	commentSQL mysql.CommentSQL,
	postSQL mysql.PostSQL,
	userSQL mysql.UserSQL,
	commentLikeSQL mysql.CommentLikeSQL,
	commentCache redis.CommentCache,
	db *gorm.DB,
) CommentService {
	return &commentService{
		commentSQL:     commentSQL,
		postSQL:        postSQL,
		userSQL:        userSQL,
		commentLikeSQL: commentLikeSQL,
		commentCache:   commentCache,
		db:             db,
	}
}

// getCommentWithUser 获取评论及其用户信息（只获取昵称和头像）
func (s *commentService) getCommentWithUser(ctx context.Context, commentID uint) (*model.Comment, error) {
	var comment model.Comment
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			// 只获取用户的基本公开信息
			return db.Select("id, name, avatar_url")
		}).
		First(&comment, commentID).Error

	if err != nil {
		return nil, ErrCommentNotFound
	}

	return &comment, nil
}

// getCommentWithFullInfo 获取评论及其所有关联数据（如果需要完整信息）
func (s *commentService) getCommentWithFullInfo(ctx context.Context, commentID uint) (*model.Comment, error) {
	var comment model.Comment
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url, bio")
		}).
		Preload("Post", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title, slug, author_name")
		}).
		Preload("Parent", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, content, user_id")
		}).
		First(&comment, commentID).Error

	if err != nil {
		return nil, ErrCommentNotFound
	}

	return &comment, nil
}

// getCommentWithReplies 获取评论及其回复（用于嵌套评论）
func (s *commentService) getCommentWithReplies(ctx context.Context, commentID uint) (*model.Comment, error) {
	var comment model.Comment
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.
				Preload("User", func(db *gorm.DB) *gorm.DB {
					return db.Select("id, name, avatar_url")
				}).
				Order("created_at ASC")
		}).
		First(&comment, commentID).Error

	if err != nil {
		return nil, ErrCommentNotFound
	}
	return &comment, nil
}
func (s *commentService) getCurrentUser(ctx context.Context) (*model.User, error) {
	userID, err := utils.GetCurrentUserIDFromContext(ctx)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// 从数据库获取用户信息
	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

func (s *commentService) CreateComment(ctx context.Context, req *CreateCommentRequest) (*model.Comment, error) {
	// 1. 验证评论内容
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrCommentInvalidContent
	}

	// 2. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, req.PostID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostIsDeleted
		}
		return nil, fmt.Errorf("获取帖子失败: %w", err)
	}

	// 4. 创建评论对象
	comment := &model.Comment{
		Content:   content,
		PostID:    req.PostID,
		UserID:    currentUser.ID,
		Status:    "published", // 默认状态为已发布
		Level:     0,           // 顶级评论级别为0
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 5. 开启事务
	var createdComment *model.Comment
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 保存评论到数据库
		if err := s.commentSQL.InsertComment(ctx, comment); err != nil {
			return fmt.Errorf("保存评论失败: %w", err)
		}

		// 更新帖子评论数
		// 或者直接更新帖子表
		updates := map[string]interface{}{
			"comment_numbers": post.CommentNumbers + 1,
			"updated_at":      time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, req.PostID, updates); err != nil {
			return fmt.Errorf("更新帖子评论数失败: %w", err)
		}

		//  更新Redis缓存
		if err := s.commentCache.IncrCommentCount(ctx, req.PostID); err != nil {
			// Redis操作失败不影响主流程，记录日志即可
			fmt.Printf("Redis评论数缓存失败: %v\n", err)
		}

		// 获取完整的评论信息
		createdComment, err = s.getCommentWithUser(ctx, comment.ID)
		if err != nil {
			return fmt.Errorf("获取评论详情失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdComment, nil
}
func (s *commentService) GetComment(ctx context.Context, id uint) (*model.Comment, error) {
	//获取评论
	comment, err := s.getCommentWithUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return comment, nil
}
func (s *commentService) DeleteComment(ctx context.Context, id uint) error {
	//  获取现有评论
	comment, err := s.commentSQL.GetCommentByID(ctx, id)
	if err != nil {
		return ErrCommentNotFound
	}

	//检查用户权限
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	if comment.UserID != currentUser.ID {
		// 可以添加管理员权限检查
		return ErrUnauthorized
	}

	//  开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 从MySQL删除评论
		if err := s.commentSQL.DeleteComment(ctx, id); err != nil {
			return fmt.Errorf("删除评论失败: %w", err)
		}

		//  更新帖子评论数
		post, err := s.postSQL.GetPostByID(ctx, comment.PostID)
		if err != nil {
			// 帖子不存在，可能是已经被删除
			return nil
		}

		// 确保评论数不小于0
		newCount := uint(0)
		if post.CommentNumbers > 0 {
			newCount = post.CommentNumbers - 1
		}

		updates := map[string]interface{}{
			"comment_numbers": newCount,
			"updated_at":      time.Now(),
		}

		if err := s.postSQL.UpdatePost(ctx, comment.PostID, updates); err != nil {
			return fmt.Errorf("更新帖子评论数失败: %w", err)
		}

		//  更新Redis缓存
		if err := s.commentCache.DecrCommentCount(ctx, comment.PostID); err != nil {
			fmt.Printf("Redis评论数缓存失败: %v\n", err)
		}

		// 删除评论的点赞缓存
		if err := s.commentCache.DeleteCommentLikeCache(ctx, id); err != nil {
			fmt.Printf("Redis评论点赞缓存删除失败: %v\n", err)
		}

		return nil
	})

	return err
}
