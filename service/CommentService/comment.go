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
	"sync"
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
	ErrRateLimited               = errors.New("操作过于频繁，请稍后再试")
	ErrOperationInProgress       = errors.New("操作正在进行中，请稍后再试")
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

	// 分布式锁管理器
	lockManager *utils.LockManager

	// 限流器
	rateLimiter *utils.RateLimiter

	// 缓存
	hotCommentsCache map[uint]*model.Comment
	hotCommentsTTL   map[uint]time.Time
	hotCommentLock   sync.RWMutex
	readCacheLock    sync.RWMutex
}

func NewCommentService(
	commentSQL mysql.CommentSQL,
	postSQL mysql.PostSQL,
	userSQL mysql.UserSQL,
	commentLikeSQL mysql.CommentLikeSQL,
	commentCache redis.CommentCache,
	db *gorm.DB,
	lockManager *utils.LockManager,
	rateLimiter *utils.RateLimiter,
) CommentService {
	return &commentService{
		commentSQL:       commentSQL,
		postSQL:          postSQL,
		userSQL:          userSQL,
		commentLikeSQL:   commentLikeSQL,
		commentCache:     commentCache,
		db:               db,
		lockManager:      lockManager,
		rateLimiter:      rateLimiter,
		hotCommentsCache: make(map[uint]*model.Comment),
		hotCommentsTTL:   make(map[uint]time.Time),
	}
}

// getCommentWithUser 获取评论及其用户信息（带缓存）
func (s *commentService) getCommentWithUser(ctx context.Context, commentID uint) (*model.Comment, error) {
	// 检查热点缓存
	s.hotCommentLock.RLock()
	if comment, ok := s.hotCommentsCache[commentID]; ok {
		if s.hotCommentsTTL[commentID].After(time.Now()) {
			s.hotCommentLock.RUnlock()
			return comment, nil
		}
	}
	s.hotCommentLock.RUnlock()

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("comment_query:%d", commentID)
	lock := s.lockManager.GetLock(lockKey, 3*time.Second)

	s.readCacheLock.RLock()
	acquired, err := lock.Acquire(ctx)
	if err != nil || !acquired {
		s.readCacheLock.RUnlock()
		return s.queryCommentWithUser(ctx, commentID)
	}
	defer lock.Release(ctx)
	s.readCacheLock.RUnlock()

	// 再次检查缓存
	s.hotCommentLock.RLock()
	if comment, ok := s.hotCommentsCache[commentID]; ok {
		if s.hotCommentsTTL[commentID].After(time.Now()) {
			s.hotCommentLock.RUnlock()
			return comment, nil
		}
	}
	s.hotCommentLock.RUnlock()

	// 查询数据库
	comment, err := s.queryCommentWithUser(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// 更新缓存（只缓存热点数据）
	if comment.LikeCount > 50 { // 认为是热点数据
		s.hotCommentLock.Lock()
		s.hotCommentsCache[commentID] = comment
		s.hotCommentsTTL[commentID] = time.Now().Add(3 * time.Minute) // 缓存3分钟
		s.hotCommentLock.Unlock()
	}

	return comment, nil
}

func (s *commentService) queryCommentWithUser(ctx context.Context, commentID uint) (*model.Comment, error) {
	var comment model.Comment
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
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

	// 使用分布式锁保护用户信息获取
	lockKey := fmt.Sprintf("user_info:%d", userID)
	lock := s.lockManager.GetLock(lockKey, 5*time.Second)

	acquired, err := lock.Acquire(ctx)
	if err != nil || !acquired {
		user, err := s.userSQL.GetUserByID(ctx, userID)
		if err != nil {
			return nil, ErrUnauthorized
		}
		return user, nil
	}
	defer lock.Release(ctx)

	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

// CreateComment 创建评论（带限流和锁保护）
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

	// 3. 用户级限流
	userRateLimitKey := fmt.Sprintf("create_comment:user:%d", currentUser.ID)
	userRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 30, // 每分钟最多30条评论
	}

	if err := s.rateLimiter.Allow(ctx, userRateLimitKey, userRateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 4. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, req.PostID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostIsDeleted
		}
		return nil, fmt.Errorf("获取帖子失败: %w", err)
	}

	// 5. 创建评论对象
	comment := &model.Comment{
		Content:   content,
		PostID:    req.PostID,
		UserID:    currentUser.ID,
		Status:    "published",
		Level:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 6. 使用分布式锁保护评论创建
	lockKey := fmt.Sprintf("post_comment:%d", req.PostID)
	var createdComment *model.Comment

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 保存评论到数据库
		if err := s.commentSQL.InsertComment(ctx, comment); err != nil {
			return fmt.Errorf("保存评论失败: %w", err)
		}

		// 更新帖子评论数
		updates := map[string]interface{}{
			"comment_numbers": post.CommentNumbers + 1,
			"updated_at":      time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, req.PostID, updates); err != nil {
			return fmt.Errorf("更新帖子评论数失败: %w", err)
		}

		// 更新Redis缓存
		if err := s.commentCache.IncrCommentCount(ctx, req.PostID); err != nil {
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

// GetComment 获取评论（带限流）
func (s *commentService) GetComment(ctx context.Context, id uint) (*model.Comment, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_comment:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 500,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 获取评论
	comment, err := s.getCommentWithUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return comment, nil
}

// DeleteComment 删除评论（带分布式锁）
func (s *commentService) DeleteComment(ctx context.Context, id uint) error {
	// 获取现有评论
	comment, err := s.commentSQL.GetCommentByID(ctx, id)
	if err != nil {
		return ErrCommentNotFound
	}

	// 检查用户权限
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	if comment.UserID != currentUser.ID {
		return ErrUnauthorized
	}

	// 使用分布式锁保护删除操作
	lockKey := fmt.Sprintf("comment_delete:%d", id)

	err = s.lockManager.GetLock(lockKey, 15*time.Second).Mutex(ctx, func() error {
		// 从MySQL删除评论
		if err := s.commentSQL.DeleteComment(ctx, id); err != nil {
			return fmt.Errorf("删除评论失败: %w", err)
		}

		// 更新帖子评论数
		post, err := s.postSQL.GetPostByID(ctx, comment.PostID)
		if err != nil {
			return nil
		}

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

		// 更新Redis缓存
		if err := s.commentCache.DecrCommentCount(ctx, comment.PostID); err != nil {
			fmt.Printf("Redis评论数缓存失败: %v\n", err)
		}

		// 删除评论的点赞缓存
		if err := s.commentCache.DeleteCommentLikeCache(ctx, id); err != nil {
			fmt.Printf("Redis评论点赞缓存删除失败: %v\n", err)
		}

		// 清除缓存
		s.hotCommentLock.Lock()
		delete(s.hotCommentsCache, id)
		delete(s.hotCommentsTTL, id)
		s.hotCommentLock.Unlock()

		return nil
	})

	return err
}

// ListCommentsByPost 获取帖子评论列表（带缓存和限流）
func (s *commentService) ListCommentsByPost(ctx context.Context, postID uint, page, size int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("list_comments:post:%d:ip:%s", postID, ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 300,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, 0, ErrRateLimited
	}

	offset := (page - 1) * size

	// 帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return nil, 0, ErrPostIsDeleted
	}

	condition := "post_id = ? AND parent_id IS NULL AND status = 'published'"
	args := []interface{}{post.ID}

	var total int64
	err = s.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where(condition, args...).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取评论总数失败: %w", err)
	}

	var comments []*model.Comment
	err = s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id,name,avatar_url")
		}).
		Where(condition, args...).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&comments).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取评论列表失败: %w", err)
	}

	// 并行获取回复和点赞数
	var wg sync.WaitGroup
	for _, comment := range comments {
		wg.Add(1)
		go func(c *model.Comment) {
			defer wg.Done()

			// 获取该评论的直接回复
			var replies []*model.Comment
			err = s.db.WithContext(ctx).
				Preload("User", func(db *gorm.DB) *gorm.DB {
					return db.Select("id, name, avatar_url")
				}).
				Where("post_id = ? AND parent_id = ? AND status = 'published'", postID, c.ID).
				Order("created_at ASC").
				Limit(3).
				Find(&replies).Error

			if err == nil && len(replies) > 0 {
				c.Replies = replies
			}

			// 获取评论点赞数
			likeCount, err := s.commentCache.CountCommentLikes(ctx, c.ID)
			if err == nil {
				c.LikeCount = uint(likeCount)
			} else {
				dbLikeCount, err := s.commentLikeSQL.CommentFindLikes(ctx, "comment_id = ?", c.ID)
				if err == nil {
					c.LikeCount = uint(len(dbLikeCount))
				}
			}
		}(comment)
	}
	wg.Wait()

	return comments, total, nil
}

// ListCommentsByUser 获取用户评论列表（带缓存和限流）
func (s *commentService) ListCommentsByUser(ctx context.Context, userID uint, page, size int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("list_user_comments:user:%d:ip:%s", userID, ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 300,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, 0, ErrRateLimited
	}

	offset := (page - 1) * size

	// 获取用户评论总数
	var total int64
	err := s.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("user_id = ? AND status = 'published'", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取用户评论总数失败: %w", err)
	}

	// 获取用户评论列表
	var comments []*model.Comment
	err = s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Post", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title, slug")
		}).
		Where("user_id = ? AND status = 'published'", userID).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&comments).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取用户评论列表失败: %w", err)
	}

	return comments, total, nil
}

// LikeComment 点赞评论（完整分布式锁实现）
func (s *commentService) LikeComment(ctx context.Context, commentID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 用户级限流
	userRateLimitKey := fmt.Sprintf("like_comment:user:%d", currentUser.ID)
	userRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 100, // 每分钟最多100次点赞
	}

	if err := s.rateLimiter.Allow(ctx, userRateLimitKey, userRateLimitConfig); err != nil {
		return ErrRateLimited
	}

	// 3. 使用分布式锁保护点赞操作
	lockKey := fmt.Sprintf("comment_like:%d:user:%d", commentID, currentUser.ID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 检查评论是否存在
		comment, err := s.getCommentWithUser(ctx, commentID)
		if err != nil {
			return ErrCommentNotFound
		}

		// 检查是否已经点赞过
		isLiked, err := s.commentCache.IsCommentLiked(ctx, currentUser.ID, commentID)
		if err != nil {
			// Redis查询失败，从MySQL检查
			likes, err := s.commentLikeSQL.CommentFindLikes(ctx, "user_id = ? AND comment_id = ?", currentUser.ID, commentID)
			if err == nil && len(likes) > 0 {
				return ErrCommentAlreadyLiked
			}
		} else if isLiked {
			return ErrCommentAlreadyLiked
		}

		// 开启事务
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 保存到MySQL点赞表
			if err := s.commentLikeSQL.CommentInsertLike(ctx, currentUser.ID, commentID); err != nil {
				return fmt.Errorf("保存评论点赞记录失败: %w", err)
			}

			// 更新评论点赞数
			updates := map[string]interface{}{
				"like_count": comment.LikeCount + 1,
				"updated_at": time.Now(),
			}
			if err := s.commentSQL.UpdateComment(ctx, commentID, updates); err != nil {
				return fmt.Errorf("更新评论点赞数失败: %w", err)
			}

			// 保存到Redis缓存
			if err := s.commentCache.LikeComment(ctx, currentUser.ID, commentID); err != nil {
				fmt.Printf("Redis评论点赞缓存失败: %v\n", err)
			}

			// 清除缓存
			s.hotCommentLock.Lock()
			delete(s.hotCommentsCache, commentID)
			delete(s.hotCommentsTTL, commentID)
			s.hotCommentLock.Unlock()

			return nil
		})

		return err
	})

	return err
}

// UnlikeComment 取消点赞评论（完整分布式锁实现）
func (s *commentService) UnlikeComment(ctx context.Context, commentID uint) error {
	// 获取用户
	currentuser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 使用分布式锁保护取消点赞操作
	lockKey := fmt.Sprintf("comment_like:%d:user:%d", commentID, currentuser.ID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		comment, err := s.getCommentWithUser(ctx, commentID)
		if err != nil {
			return ErrCommentNotFound
		}

		// 检查是否被点赞
		isliked, err := s.commentCache.IsCommentLiked(ctx, currentuser.ID, commentID)
		if err != nil {
			likes, err := s.commentLikeSQL.CommentFindLikes(ctx, "user_id = ? AND comment_id = ?", currentuser.ID, commentID)
			if err == nil || len(likes) > 0 {
				return ErrCommentNotLiked
			}
		} else if !isliked {
			return ErrCommentNotLiked
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 从MySQL删除点赞记录
			if err := s.commentLikeSQL.CommentDeleteLike(ctx, currentuser.ID, commentID); err != nil {
				return fmt.Errorf("删除评论点赞记录失败: %w", err)
			}

			// 更新评论点赞数
			if comment.LikeCount > 0 {
				updates := map[string]interface{}{
					"like_count": comment.LikeCount - 1,
					"updated_at": time.Now(),
				}
				if err := s.commentSQL.UpdateComment(ctx, commentID, updates); err != nil {
					return fmt.Errorf("更新评论点赞数失败: %w", err)
				}
			}

			// 从Redis缓存删除
			if err := s.commentCache.UnlikeComment(ctx, currentuser.ID, commentID); err != nil {
				fmt.Printf("Redis取消评论点赞缓存失败: %v\n", err)
			}

			// 清除缓存
			s.hotCommentLock.Lock()
			delete(s.hotCommentsCache, commentID)
			delete(s.hotCommentsTTL, commentID)
			s.hotCommentLock.Unlock()

			return nil
		})

		return err
	})

	return err
}

// GetCommentLikes 获取评论点赞数（带缓存）
func (s *commentService) GetCommentLikes(ctx context.Context, commentID uint) (uint, error) {
	// 尝试从Redis获取
	count, err := s.commentCache.CountCommentLikes(ctx, commentID)
	if err == nil && count > 0 {
		return uint(count), nil
	}

	// 从MySQL获取
	comment, err := s.commentSQL.GetCommentByID(ctx, commentID)
	if err != nil {
		return 0, ErrCommentNotFound
	}

	return comment.LikeCount, nil
}

// IsCommentLiked 检查是否点赞评论
func (s *commentService) IsCommentLiked(ctx context.Context, commentID uint) (bool, error) {
	// 获取用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return false, err
	}

	// 从redis获取
	isLiked, err := s.commentCache.IsCommentLiked(ctx, currentUser.ID, commentID)
	if err != nil {
		return isLiked, nil
	}

	// 从sql获取
	likes, err := s.commentLikeSQL.CommentFindLikes(ctx, "user_id = ? AND comment_id = ?", currentUser.ID, commentID)
	if err != nil {
		return false, err
	}
	return len(likes) > 0, nil
}

// CreateReply 创建回复（带限流和锁保护）
func (s *commentService) CreateReply(ctx context.Context, req *CreateReplyRequest) (*model.Comment, error) {
	// 回复不能为空
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrCommentInvalidContent
	}

	// 获取用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// 用户级限流
	userRateLimitKey := fmt.Sprintf("create_reply:user:%d", currentUser.ID)
	userRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 50,
	}

	if err := s.rateLimiter.Allow(ctx, userRateLimitKey, userRateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, req.PostID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostIsDeleted
		}
		return nil, fmt.Errorf("获取帖子失败：%w", err)
	}

	// 获取上一级评论
	parentComment, err := s.commentSQL.GetCommentByID(ctx, req.ParentID)
	if err != nil {
		return nil, ErrReplyToNonexistentComment
	}

	// 创建回复
	reply := &model.Comment{
		Content:   content,
		PostID:    req.PostID,
		ParentID:  &req.ParentID,
		UserID:    currentUser.ID,
		Level:     parentComment.Level + 1,
		Status:    "published",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var createdReply *model.Comment

	// 使用分布式锁保护回复创建
	lockKey := fmt.Sprintf("comment_reply:%d", req.ParentID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		if err := s.commentSQL.InsertComment(ctx, reply); err != nil {
			return fmt.Errorf("保存回复失败:%w", err)
		}

		updates := map[string]interface{}{
			"comment_numbers": post.CommentNumbers + 1,
			"updated_at":      time.Now(),
		}

		if err := s.postSQL.UpdatePost(ctx, req.PostID, updates); err != nil {
			return fmt.Errorf("更新帖子评论数失败:%w", err)
		}

		if err := s.commentCache.IncrCommentCount(ctx, req.PostID); err != nil {
			return fmt.Errorf("评论数缓存失败:%w", err)
		}

		createdReply, err = s.getCommentWithUser(ctx, reply.ID)
		if err != nil {
			return fmt.Errorf("获取回复详情失败%w", err)
		}

		// 清除父评论缓存
		s.hotCommentLock.Lock()
		delete(s.hotCommentsCache, req.ParentID)
		delete(s.hotCommentsTTL, req.ParentID)
		s.hotCommentLock.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdReply, nil
}

// ListReplies 获取评论回复列表（带缓存和限流）
func (s *commentService) ListReplies(ctx context.Context, commentID uint, page, size int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("list_replies:comment:%d:ip:%s", commentID, ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 200,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, 0, ErrRateLimited
	}

	offset := (page - 1) * size

	// 检查上级评论
	_, err := s.commentSQL.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, 0, ErrCommentNotFound
	}

	// 获取回复总数
	var total int64
	err = s.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("parent_id = ? AND status = 'published'", commentID).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取回复总数失败:%w", err)
	}

	// 获取回复列表
	var replies []*model.Comment
	err = s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id,name,avatar_url")
		}).
		Where("parent_id = ? AND status = 'published'", commentID).
		Order("created_at ASC").
		Limit(size).
		Offset(offset).
		Find(&replies).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取回复列表失败：%w", err)
	}

	// 并行获取点赞数
	var wg sync.WaitGroup
	for _, reply := range replies {
		wg.Add(1)
		go func(r *model.Comment) {
			defer wg.Done()
			likeCount, err := s.commentCache.CountCommentLikes(ctx, r.ID)
			if err == nil {
				r.LikeCount = uint(likeCount)
			} else {
				dbLikeCount, err := s.commentLikeSQL.CommentFindLikes(ctx, "comment_id = ?", r.ID)
				if err == nil {
					r.LikeCount = uint(len(dbLikeCount))
				}
			}
		}(reply)
	}
	wg.Wait()

	return replies, total, nil
}
