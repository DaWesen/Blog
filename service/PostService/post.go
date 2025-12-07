// service/post_service.go - 完整版本
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

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrPostNotFound       = errors.New("文章不存在")
	ErrPostSlugExists     = errors.New("文章别名已存在")
	ErrInvalidPostTitle   = errors.New("文章标题不能为空")
	ErrUnauthorized       = errors.New("用户未认证")
	ErrPostAlreadyLiked   = errors.New("已经点赞过此帖子")
	ErrPostNotLiked       = errors.New("还没有点赞此帖子")
	ErrPostAlreadyStarred = errors.New("已经收藏过此帖子")
	ErrPostNotStarred     = errors.New("还没有收藏此帖子")
)

// PostService 接口 - 包含所有帖子功能
type PostService interface {
	// 帖子基本功能
	CreatePost(ctx context.Context, req *CreatePostRequest) (*model.Post, error)
	GetPost(ctx context.Context, id uint) (*model.Post, error)
	GetPostBySlug(ctx context.Context, slug string) (*model.Post, error)
	UpdatePost(ctx context.Context, id uint, req *UpdatePostRequest) (*model.Post, error)
	DeletePost(ctx context.Context, id uint) error
	ListPosts(ctx context.Context, page, size int) ([]*model.Post, int64, error)
	ListPostsByCategory(ctx context.Context, categoryID uint, page, size int) ([]*model.Post, int64, error)
	ListPostsByTag(ctx context.Context, tagID uint, page, size int) ([]*model.Post, int64, error)
	SearchPosts(ctx context.Context, keyword string, page, size int) ([]*model.Post, int64, error)

	// 统计功能
	LikePost(ctx context.Context, postID uint) error
	UnlikePost(ctx context.Context, postID uint) error
	GetPostLikes(ctx context.Context, postID uint) (uint, error)
	IsPostLiked(ctx context.Context, postID uint) (bool, error)

	StarPost(ctx context.Context, postID uint) error
	UnstarPost(ctx context.Context, postID uint) error
	GetPostStars(ctx context.Context, postID uint) (uint, error)
	IsPostStarred(ctx context.Context, postID uint) (bool, error)

	GetPostCommentsCount(ctx context.Context, postID uint) (uint, error)
	IncrementComments(ctx context.Context, postID uint) error
	DecrementComments(ctx context.Context, postID uint) error

	IncrementViews(ctx context.Context, postID uint) error
	GetPostViews(ctx context.Context, postID uint) (uint, error)

	GetPostStats(ctx context.Context, postID uint) (*PostStats, error)
}

// 统计数据结构
type PostStats struct {
	PostID    uint `json:"post_id"`
	Likes     uint `json:"likes"`
	Stars     uint `json:"stars"`
	Comments  uint `json:"comments"`
	Views     uint `json:"views"`
	IsLiked   bool `json:"is_liked"`   // 当前用户是否点赞
	IsStarred bool `json:"is_starred"` // 当前用户是否收藏
}

// 请求结构体
type CreatePostRequest struct {
	Title      string `json:"title" binding:"required,min=1,max=255"`
	Content    string `json:"content" binding:"required,min=1"`
	Summary    string `json:"summary,omitempty"`
	Slug       string `json:"slug,omitempty" binding:"omitempty,min=1,max=255"`
	CategoryID uint   `json:"category_id" binding:"required"`
	TagIDs     []uint `json:"tag_ids,omitempty"`
	Visibility string `json:"visibility,omitempty" binding:"omitempty,oneof=public private password friends"`
}

type UpdatePostRequest struct {
	Title      *string `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	Content    *string `json:"content,omitempty" binding:"omitempty,min=1"`
	Summary    *string `json:"summary,omitempty"`
	Slug       *string `json:"slug,omitempty" binding:"omitempty,min=1,max=255"`
	CategoryID *uint   `json:"category_id,omitempty"`
	TagIDs     *[]uint `json:"tag_ids,omitempty"`
	Visibility *string `json:"visibility,omitempty" binding:"omitempty,oneof=public private password friends"`
}

// Service实现结构体
type postService struct {
	postSQL     mysql.PostSQL
	userSQL     mysql.UserSQL
	categorySQL mysql.CategorySQL
	tagSQL      mysql.TagSQL
	likeSQL     mysql.LikeSQL
	starSQL     mysql.StarSQL
	commentSQL  mysql.CommentSQL
	db          *gorm.DB

	// Redis缓存接口
	viewCache    redis.ViewCache
	likeCache    redis.LikeCache
	starCache    redis.StarCache
	commentCache redis.CommentCache
}

// 创建Service实例
func NewPostService(
	postSQL mysql.PostSQL,
	userSQL mysql.UserSQL,
	categorySQL mysql.CategorySQL,
	tagSQL mysql.TagSQL,
	likeSQL mysql.LikeSQL,
	starSQL mysql.StarSQL,
	commentSQL mysql.CommentSQL,
	db *gorm.DB,
	viewCache redis.ViewCache,
	likeCache redis.LikeCache,
	starCache redis.StarCache,
	commentCache redis.CommentCache,
) PostService {
	return &postService{
		postSQL:      postSQL,
		userSQL:      userSQL,
		categorySQL:  categorySQL,
		tagSQL:       tagSQL,
		likeSQL:      likeSQL,
		starSQL:      starSQL,
		commentSQL:   commentSQL,
		db:           db,
		viewCache:    viewCache,
		likeCache:    likeCache,
		starCache:    starCache,
		commentCache: commentCache,
	}
}

// getCurrentUserIDFromContext 从上下文中获取当前用户ID
func (s *postService) getCurrentUserIDFromContext(ctx context.Context) (uint, error) {
	// 尝试从Gin上下文中获取（适用于HTTP请求）
	if ginCtx, ok := ctx.Value("ginContext").(*gin.Context); ok {
		userID, err := utils.GetUserIDFromGin(ginCtx)
		if err == nil {
			return userID, nil
		}
	}

	// 尝试从标准context中获取（适用于gRPC或其他场景）
	if userID, ok := ctx.Value("user_id").(uint); ok {
		return userID, nil
	}

	// 尝试从标准context中获取（字符串类型）
	if userIDStr, ok := ctx.Value("user_id").(string); ok {
		var userID uint
		fmt.Sscanf(userIDStr, "%d", &userID)
		return userID, nil
	}

	return 0, ErrUnauthorized
}

// getCurrentUser 从上下文中获取当前用户完整信息
func (s *postService) getCurrentUser(ctx context.Context) (*model.User, error) {
	userID, err := s.getCurrentUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 从数据库获取用户详细信息
	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// getPostWithAssociations 获取帖子及其关联数据
func (s *postService) getPostWithAssociations(ctx context.Context, postID uint) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			// 只选择需要的字段，避免返回敏感信息
			return db.Select("id, name, avatar_url, bio")
		}).
		Preload("Category").
		Preload("Tags").
		First(&post, postID).Error

	if err != nil {
		return nil, err
	}

	return &post, nil
}

// CreatePost 创建帖子（自动补充作者信息）
func (s *postService) CreatePost(ctx context.Context, req *CreatePostRequest) (*model.Post, error) {
	// 1. 参数验证
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrInvalidPostTitle
	}

	// 2. 获取当前用户信息（自动补充作者的个人特色昵称）
	user, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 检查分类是否存在
	if _, err := s.categorySQL.GetCategoryByID(ctx, req.CategoryID); err != nil {
		return nil, errors.New("分类不存在")
	}

	// 4. 检查标签是否存在（如果提供了标签）
	for _, tagID := range req.TagIDs {
		if _, err := s.tagSQL.GetTagByID(ctx, tagID); err != nil {
			return nil, fmt.Errorf("标签ID %d 不存在", tagID)
		}
	}

	// 5. 处理slug（如果没传则自动生成）
	slug := ""
	if req.Slug != "" {
		slug = utils.SanitizeSlug(req.Slug)
	} else {
		slug = utils.GenerateSlug(title)
	}

	// 6. 检查slug是否已存在
	existing, _ := s.postSQL.GetPostBySlug(ctx, slug)
	if existing != nil {
		// 如果slug已存在，添加时间戳后缀
		timestamp := time.Now().Format("20060102-150405")
		slug = fmt.Sprintf("%s-%s", slug, timestamp)

		// 再次检查
		existing, _ = s.postSQL.GetPostBySlug(ctx, slug)
		if existing != nil {
			return nil, ErrPostSlugExists
		}
	}

	// 7. 处理摘要（如果没传则从内容生成）
	summary := req.Summary
	if summary == "" && req.Content != "" {
		// 取内容前200个字符作为摘要，确保不截断中文字符
		contentRunes := []rune(req.Content)
		if len(contentRunes) > 200 {
			summary = string(contentRunes[:200]) + "..."
		} else {
			summary = req.Content
		}
	}

	// 8. 处理可见性（默认为公开）
	var visibility model.Visibility
	if req.Visibility != "" {
		visibility = model.Visibility(req.Visibility)
	} else {
		visibility = model.VisibilityPublic
	}

	// 9. 创建帖子对象（设置作者的个人特色昵称）
	post := &model.Post{
		Title:      title,
		Slug:       slug,
		Content:    req.Content,
		Summary:    summary,
		UserID:     user.ID,   // 用户ID用于关联
		AuthorName: user.Name, // 作者的个人特色昵称，存储在数据库中
		CategoryID: req.CategoryID,
		Visibility: visibility,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 10. 开启事务保存帖子和标签关联
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 保存帖子
		if err := s.postSQL.InsertPost(ctx, post); err != nil {
			return fmt.Errorf("保存帖子失败: %w", err)
		}

		// 如果有关联标签，创建关联
		if len(req.TagIDs) > 0 {
			for _, tagID := range req.TagIDs {
				postTag := &model.PostTag{
					PostID:    post.ID,
					TagID:     tagID,
					CreatedAt: time.Now(),
				}
				if err := tx.WithContext(ctx).Create(postTag).Error; err != nil {
					return fmt.Errorf("关联标签失败: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 11. 获取完整的帖子信息（包含关联数据）
	fullPost, err := s.getPostWithAssociations(ctx, post.ID)
	if err != nil {
		return nil, fmt.Errorf("获取帖子详情失败: %w", err)
	}

	return fullPost, nil
}

// GetPost 获取帖子详情
func (s *postService) GetPost(ctx context.Context, id uint) (*model.Post, error) {
	post, err := s.getPostWithAssociations(ctx, id)
	if err != nil {
		return nil, err
	}

	// 异步增加浏览量（不阻塞返回）
	go func() {
		_ = s.IncrementViews(ctx, id)
	}()

	return post, nil
}

// GetPostBySlug 通过slug获取帖子
func (s *postService) GetPostBySlug(ctx context.Context, slug string) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url, bio")
		}).
		Preload("Category").
		Preload("Tags").
		Where("slug = ?", slug).
		First(&post).Error

	if err != nil {
		return nil, ErrPostNotFound
	}

	// 异步增加浏览量
	go func() {
		_ = s.IncrementViews(ctx, post.ID)
	}()

	return &post, nil
}

// UpdatePost 更新帖子
func (s *postService) UpdatePost(ctx context.Context, id uint, req *UpdatePostRequest) (*model.Post, error) {
	// 1. 获取现有帖子
	post, err := s.postSQL.GetPostByID(ctx, id)
	if err != nil {
		return nil, ErrPostNotFound
	}

	// 2. 检查用户权限
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	if post.UserID != currentUser.ID {
		// 这里可以添加管理员权限检查
		return nil, errors.New("没有权限修改此帖子")
	}

	// 3. 构建更新数据
	updates := make(map[string]interface{})

	if req.Title != nil {
		newTitle := strings.TrimSpace(*req.Title)
		if newTitle != "" && newTitle != post.Title {
			updates["title"] = newTitle
		}
	}

	if req.Content != nil && *req.Content != post.Content {
		updates["content"] = *req.Content
	}

	if req.Summary != nil && *req.Summary != post.Summary {
		updates["summary"] = *req.Summary
	}

	if req.Slug != nil {
		newSlug := utils.SanitizeSlug(*req.Slug)
		if newSlug != "" && newSlug != post.Slug {
			// 检查新slug是否已被其他帖子使用
			existing, _ := s.postSQL.GetPostBySlug(ctx, newSlug)
			if existing != nil && existing.ID != id {
				return nil, ErrPostSlugExists
			}
			updates["slug"] = newSlug
		}
	}

	if req.CategoryID != nil && *req.CategoryID != post.CategoryID {
		// 检查分类是否存在
		if _, err := s.categorySQL.GetCategoryByID(ctx, *req.CategoryID); err != nil {
			return nil, errors.New("分类不存在")
		}
		updates["category_id"] = *req.CategoryID
	}

	if req.Visibility != nil && *req.Visibility != string(post.Visibility) {
		updates["visibility"] = *req.Visibility
	}

	// 注意：不更新 AuthorName，因为作者昵称不应该被修改

	// 如果没有更新内容，直接返回
	if len(updates) == 0 {
		return s.getPostWithAssociations(ctx, id)
	}

	updates["updated_at"] = time.Now()

	// 4. 更新帖子
	if err := s.postSQL.UpdatePost(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("更新帖子失败: %w", err)
	}

	// 5. 获取更新后的帖子
	return s.getPostWithAssociations(ctx, id)
}

// DeletePost 删除帖子
func (s *postService) DeletePost(ctx context.Context, id uint) error {
	// 先检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, id)
	if err != nil {
		return ErrPostNotFound
	}

	// 检查用户权限
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	if post.UserID != currentUser.ID {
		// 这里可以添加管理员权限检查
		return errors.New("没有权限删除此帖子")
	}

	return s.postSQL.DeletePost(ctx, id)
}

// ListPosts 分页列出帖子
func (s *postService) ListPosts(ctx context.Context, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	var posts []*model.Post
	var total int64

	// 统计总数
	s.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("visibility = ?", model.VisibilityPublic).
		Count(&total)

	// 查询帖子并预加载关联数据
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Category").
		Preload("Tags").
		Where("visibility = ?", model.VisibilityPublic).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&posts).Error

	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// ListPostsByCategory 按分类列出帖子
func (s *postService) ListPostsByCategory(ctx context.Context, categoryID uint, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	var posts []*model.Post
	var total int64

	// 统计总数
	s.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("category_id = ? AND visibility = ?", categoryID, model.VisibilityPublic).
		Count(&total)

	// 查询帖子并预加载关联数据
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Category").
		Preload("Tags").
		Where("category_id = ? AND visibility = ?", categoryID, model.VisibilityPublic).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&posts).Error

	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// ListPostsByTag 按标签列出帖子
func (s *postService) ListPostsByTag(ctx context.Context, tagID uint, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	var posts []*model.Post
	var total int64

	// 统计总数（通过多表查询）
	s.db.WithContext(ctx).
		Model(&model.Post{}).
		Joins("JOIN post_tags ON posts.id = post_tags.post_id").
		Where("post_tags.tag_id = ? AND posts.visibility = ?", tagID, model.VisibilityPublic).
		Count(&total)

	// 查询帖子并预加载关联数据
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Category").
		Preload("Tags").
		Joins("JOIN post_tags ON posts.id = post_tags.post_id").
		Where("post_tags.tag_id = ? AND posts.visibility = ?", tagID, model.VisibilityPublic).
		Order("posts.created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&posts).Error

	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// SearchPosts 搜索帖子
func (s *postService) SearchPosts(ctx context.Context, keyword string, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return s.ListPosts(ctx, page, size)
	}

	var posts []*model.Post
	var total int64

	searchPattern := "%" + keyword + "%"

	// 统计总数
	s.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("(title LIKE ? OR content LIKE ? OR summary LIKE ? OR author_name LIKE ?) AND visibility = ?",
			searchPattern, searchPattern, searchPattern, searchPattern, model.VisibilityPublic).
		Count(&total)

	// 查询帖子并预加载关联数据
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Category").
		Preload("Tags").
		Where("(title LIKE ? OR content LIKE ? OR summary LIKE ? OR author_name LIKE ?) AND visibility = ?",
			searchPattern, searchPattern, searchPattern, searchPattern, model.VisibilityPublic).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&posts).Error

	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// LikePost 点赞帖子
func (s *postService) LikePost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 3. 检查是否已经点赞过
	isLiked, err := s.likeCache.IsLiked(ctx, currentUser.ID, postID)
	if err != nil {
		// Redis查询失败，从MySQL检查
		likes, err := s.likeSQL.FindLikes(ctx, "user_id = ? AND post_id = ?", currentUser.ID, postID)
		if err == nil && len(likes) > 0 {
			return ErrPostAlreadyLiked
		}
	} else if isLiked {
		return ErrPostAlreadyLiked
	}

	// 4. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 4.1 保存到MySQL点赞表
		if err := s.likeSQL.InsertLike(ctx, currentUser.ID, postID); err != nil {
			return fmt.Errorf("保存点赞记录失败: %w", err)
		}

		// 4.2 更新帖子点赞数
		updates := map[string]interface{}{
			"liketimes":  post.Liketimes + 1,
			"updated_at": time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
			return fmt.Errorf("更新帖子点赞数失败: %w", err)
		}

		// 4.3 保存到Redis缓存
		if err := s.likeCache.Like(ctx, currentUser.ID, postID); err != nil {
			// Redis操作失败不影响主流程，记录日志即可
			fmt.Printf("Redis点赞缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// UnlikePost 取消点赞
func (s *postService) UnlikePost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 3. 检查是否已经点赞过
	isLiked, err := s.likeCache.IsLiked(ctx, currentUser.ID, postID)
	if err != nil {
		// Redis查询失败，从MySQL检查
		likes, err := s.likeSQL.FindLikes(ctx, "user_id = ? AND post_id = ?", currentUser.ID, postID)
		if err != nil || len(likes) == 0 {
			return ErrPostNotLiked
		}
	} else if !isLiked {
		return ErrPostNotLiked
	}

	// 4. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 4.1 从MySQL删除点赞记录
		if err := s.likeSQL.DeleteLike(ctx, currentUser.ID, postID); err != nil {
			return fmt.Errorf("删除点赞记录失败: %w", err)
		}

		// 4.2 更新帖子点赞数
		if post.Liketimes > 0 {
			updates := map[string]interface{}{
				"liketimes":  post.Liketimes - 1,
				"updated_at": time.Now(),
			}
			if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
				return fmt.Errorf("更新帖子点赞数失败: %w", err)
			}
		}

		// 4.3 从Redis缓存删除
		if err := s.likeCache.Unlike(ctx, currentUser.ID, postID); err != nil {
			fmt.Printf("Redis取消点赞缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// GetPostLikes 获取帖子点赞数
func (s *postService) GetPostLikes(ctx context.Context, postID uint) (uint, error) {
	// 1. 尝试从Redis获取
	count, err := s.likeCache.CountLikes(ctx, postID)
	if err == nil && count > 0 {
		return uint(count), nil
	}

	// 2. 从MySQL获取
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return 0, ErrPostNotFound
	}

	return post.Liketimes, nil
}

// IsPostLiked 检查当前用户是否点赞过帖子
func (s *postService) IsPostLiked(ctx context.Context, postID uint) (bool, error) {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return false, err
	}

	// 2. 尝试从Redis获取
	isLiked, err := s.likeCache.IsLiked(ctx, currentUser.ID, postID)
	if err == nil {
		return isLiked, nil
	}

	// 3. 从MySQL获取
	likes, err := s.likeSQL.FindLikes(ctx, "user_id = ? AND post_id = ?", currentUser.ID, postID)
	if err != nil {
		return false, err
	}

	return len(likes) > 0, nil
}

// StarPost 收藏帖子
func (s *postService) StarPost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 3. 检查是否已经收藏过
	isStarred, err := s.starCache.IsStarred(ctx, currentUser.ID, postID)
	if err != nil {
		// Redis查询失败，从MySQL检查
		stars, err := s.starSQL.FindStars(ctx, "user_id = ? AND post_id = ?", currentUser.ID, postID)
		if err == nil && len(stars) > 0 {
			return ErrPostAlreadyStarred
		}
	} else if isStarred {
		return ErrPostAlreadyStarred
	}

	// 4. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 4.1 保存到MySQL收藏表
		if err := s.starSQL.InsertStar(ctx, currentUser.ID, postID); err != nil {
			return fmt.Errorf("保存收藏记录失败: %w", err)
		}

		// 4.2 更新帖子收藏数
		updates := map[string]interface{}{
			"staredtimes": post.Staredtimes + 1,
			"updated_at":  time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
			return fmt.Errorf("更新帖子收藏数失败: %w", err)
		}

		// 4.3 保存到Redis缓存
		if err := s.starCache.Star(ctx, currentUser.ID, postID); err != nil {
			fmt.Printf("Redis收藏缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// UnstarPost 取消收藏
func (s *postService) UnstarPost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 3. 检查是否已经收藏过
	isStarred, err := s.starCache.IsStarred(ctx, currentUser.ID, postID)
	if err != nil {
		// Redis查询失败，从MySQL检查
		stars, err := s.starSQL.FindStars(ctx, "user_id = ? AND post_id = ?", currentUser.ID, postID)
		if err != nil || len(stars) == 0 {
			return ErrPostNotStarred
		}
	} else if !isStarred {
		return ErrPostNotStarred
	}

	// 4. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 4.1 从MySQL删除收藏记录
		if err := s.starSQL.DeleteStar(ctx, currentUser.ID, postID); err != nil {
			return fmt.Errorf("删除收藏记录失败: %w", err)
		}

		// 4.2 更新帖子收藏数
		if post.Staredtimes > 0 {
			updates := map[string]interface{}{
				"staredtimes": post.Staredtimes - 1,
				"updated_at":  time.Now(),
			}
			if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
				return fmt.Errorf("更新帖子收藏数失败: %w", err)
			}
		}

		// 4.3 从Redis缓存删除
		if err := s.starCache.Unstar(ctx, currentUser.ID, postID); err != nil {
			fmt.Printf("Redis取消收藏缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// GetPostStars 获取帖子收藏数
func (s *postService) GetPostStars(ctx context.Context, postID uint) (uint, error) {
	// 1. 尝试从Redis获取
	count, err := s.starCache.CountStars(ctx, postID)
	if err == nil && count > 0 {
		return uint(count), nil
	}

	// 2. 从MySQL获取
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return 0, ErrPostNotFound
	}

	return post.Staredtimes, nil
}

// IsPostStarred 检查当前用户是否收藏过帖子
func (s *postService) IsPostStarred(ctx context.Context, postID uint) (bool, error) {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return false, err
	}

	// 2. 尝试从Redis获取
	isStarred, err := s.starCache.IsStarred(ctx, currentUser.ID, postID)
	if err == nil {
		return isStarred, nil
	}

	// 3. 从MySQL获取
	stars, err := s.starSQL.FindStars(ctx, "user_id = ? AND post_id = ?", currentUser.ID, postID)
	if err != nil {
		return false, err
	}

	return len(stars) > 0, nil
}

// GetPostCommentsCount 获取帖子评论数
func (s *postService) GetPostCommentsCount(ctx context.Context, postID uint) (uint, error) {
	// 1. 尝试从Redis获取
	count, err := s.commentCache.GetCommentCount(ctx, postID)
	if err == nil && count > 0 {
		return uint(count), nil
	}

	// 2. 从MySQL获取
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return 0, ErrPostNotFound
	}

	return post.CommentNumbers, nil
}

// IncrementComments 增加评论数（由评论服务调用）
func (s *postService) IncrementComments(ctx context.Context, postID uint) error {
	// 1. 获取帖子
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 2. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 更新帖子评论数
		updates := map[string]interface{}{
			"comment_numbers": post.CommentNumbers + 1,
			"updated_at":      time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
			return fmt.Errorf("更新帖子评论数失败: %w", err)
		}

		// 更新Redis缓存
		if err := s.commentCache.IncrCommentCount(ctx, postID); err != nil {
			fmt.Printf("Redis评论数缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// DecrementComments 减少评论数（由评论服务调用）
func (s *postService) DecrementComments(ctx context.Context, postID uint) error {
	// 1. 获取帖子
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 2. 确保评论数不小于0
	newCount := uint(0)
	if post.CommentNumbers > 0 {
		newCount = post.CommentNumbers - 1
	}

	// 3. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 更新帖子评论数
		updates := map[string]interface{}{
			"comment_numbers": newCount,
			"updated_at":      time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
			return fmt.Errorf("更新帖子评论数失败: %w", err)
		}

		// 更新Redis缓存
		if err := s.commentCache.DecrCommentCount(ctx, postID); err != nil {
			fmt.Printf("Redis评论数缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// IncrementViews 增加浏览量
func (s *postService) IncrementViews(ctx context.Context, postID uint) error {
	// 1. 获取帖子
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return ErrPostNotFound
	}

	// 2. 开启事务
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 更新帖子浏览量
		updates := map[string]interface{}{
			"clicktimes": post.Clicktimes + 1,
			"updated_at": time.Now(),
		}
		if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
			return fmt.Errorf("更新帖子浏览量失败: %w", err)
		}

		// 更新Redis缓存
		if err := s.viewCache.IncrViewCount(ctx, postID); err != nil {
			fmt.Printf("Redis浏览量缓存失败: %v\n", err)
		}

		return nil
	})

	return err
}

// GetPostViews 获取帖子浏览量
func (s *postService) GetPostViews(ctx context.Context, postID uint) (uint, error) {
	// 1. 尝试从Redis获取
	count, err := s.viewCache.GetViewCount(ctx, postID)
	if err == nil && count > 0 {
		return uint(count), nil
	}

	// 2. 从MySQL获取
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return 0, ErrPostNotFound
	}

	return post.Clicktimes, nil
}

// GetPostStats 获取帖子综合统计数据
func (s *postService) GetPostStats(ctx context.Context, postID uint) (*PostStats, error) {
	// 1. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return nil, ErrPostNotFound
	}

	// 2. 获取当前用户（用于判断是否点赞/收藏）
	currentUser, _ := s.getCurrentUser(ctx) // 忽略错误，游客也可以查看统计

	stats := &PostStats{
		PostID:   postID,
		Likes:    post.Liketimes,
		Stars:    post.Staredtimes,
		Comments: post.CommentNumbers,
		Views:    post.Clicktimes,
	}

	// 3. 如果用户已登录，获取点赞和收藏状态
	if currentUser != nil {
		// 并行获取点赞和收藏状态
		likeChan := make(chan bool, 1)
		starChan := make(chan bool, 1)

		go func() {
			isLiked, _ := s.IsPostLiked(ctx, postID)
			likeChan <- isLiked
		}()

		go func() {
			isStarred, _ := s.IsPostStarred(ctx, postID)
			starChan <- isStarred
		}()

		stats.IsLiked = <-likeChan
		stats.IsStarred = <-starChan
	}

	return stats, nil
}
