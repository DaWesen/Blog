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

// 错误定义
var (
	ErrPostNotFound        = errors.New("文章不存在")
	ErrPostSlugExists      = errors.New("文章别名已存在")
	ErrInvalidPostTitle    = errors.New("文章标题不能为空")
	ErrUnauthorized        = errors.New("用户未认证")
	ErrPostAlreadyLiked    = errors.New("已经点赞过此帖子")
	ErrPostNotLiked        = errors.New("还没有点赞此帖子")
	ErrPostAlreadyStarred  = errors.New("已经收藏过此帖子")
	ErrPostNotStarred      = errors.New("还没有收藏此帖子")
	ErrRateLimited         = errors.New("操作过于频繁，请稍后再试")
	ErrOperationInProgress = errors.New("操作正在进行中，请稍后再试")
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

	// 分布式锁管理器
	lockManager *utils.LockManager

	// 限流器
	rateLimiter *utils.RateLimiter

	// 缓存读取锁（本地锁，用于缓存读保护）
	readCacheLock sync.RWMutex
	// 热点数据缓存
	hotPostsCache map[uint]*model.Post
	hotPostsTTL   map[uint]time.Time
	hotPostLock   sync.RWMutex
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
	lockManager *utils.LockManager,
	rateLimiter *utils.RateLimiter,
) PostService {
	return &postService{
		postSQL:       postSQL,
		userSQL:       userSQL,
		categorySQL:   categorySQL,
		tagSQL:        tagSQL,
		likeSQL:       likeSQL,
		starSQL:       starSQL,
		commentSQL:    commentSQL,
		db:            db,
		viewCache:     viewCache,
		likeCache:     likeCache,
		starCache:     starCache,
		commentCache:  commentCache,
		lockManager:   lockManager,
		rateLimiter:   rateLimiter,
		hotPostsCache: make(map[uint]*model.Post),
		hotPostsTTL:   make(map[uint]time.Time),
	}
}

// getCurrentUser 从上下文中获取当前用户完整信息
func (s *postService) getCurrentUser(ctx context.Context) (*model.User, error) {
	userID, err := utils.GetCurrentUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 使用分布式锁保护用户信息获取
	lockKey := fmt.Sprintf("user_info:%d", userID)
	lock := s.lockManager.GetLock(lockKey, 5*time.Second)

	// 快速获取锁，避免阻塞
	acquired, err := lock.Acquire(ctx)
	if err != nil || !acquired {
		// 如果获取锁失败，尝试直接查询（可能会有并发问题，但用户信息通常变化不大）
		user, err := s.userSQL.GetUserByID(ctx, userID)
		if err != nil {
			return nil, errors.New("用户不存在")
		}
		return user, nil
	}
	defer lock.Release(ctx)

	// 从数据库获取用户详细信息
	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// getPostWithAssociations 获取帖子及其关联数据（带缓存）
func (s *postService) getPostWithAssociations(ctx context.Context, postID uint) (*model.Post, error) {
	// 首先检查热点缓存
	s.hotPostLock.RLock()
	if post, ok := s.hotPostsCache[postID]; ok {
		if s.hotPostsTTL[postID].After(time.Now()) {
			s.hotPostLock.RUnlock()
			return post, nil
		}
	}
	s.hotPostLock.RUnlock()

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("post_query:%d", postID)
	lock := s.lockManager.GetLock(lockKey, 3*time.Second)

	// 使用读锁保护，允许多个读取
	s.readCacheLock.RLock()
	acquired, err := lock.Acquire(ctx)
	if err != nil || !acquired {
		// 如果获取锁失败，尝试直接查询
		s.readCacheLock.RUnlock()
		return s.queryPostWithAssociations(ctx, postID)
	}
	defer lock.Release(ctx)
	s.readCacheLock.RUnlock()

	// 再次检查缓存（防止重复查询）
	s.hotPostLock.RLock()
	if post, ok := s.hotPostsCache[postID]; ok {
		if s.hotPostsTTL[postID].After(time.Now()) {
			s.hotPostLock.RUnlock()
			return post, nil
		}
	}
	s.hotPostLock.RUnlock()

	// 查询数据库
	post, err := s.queryPostWithAssociations(ctx, postID)
	if err != nil {
		return nil, err
	}

	// 更新缓存（只缓存热点数据）
	if post.Clicktimes > 1000 { // 认为是热点数据
		s.hotPostLock.Lock()
		s.hotPostsCache[postID] = post
		s.hotPostsTTL[postID] = time.Now().Add(5 * time.Minute) // 缓存5分钟
		s.hotPostLock.Unlock()
	}

	return post, nil
}

func (s *postService) queryPostWithAssociations(ctx context.Context, postID uint) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB {
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

// CreatePost 创建帖子（带限流和锁保护）
func (s *postService) CreatePost(ctx context.Context, req *CreatePostRequest) (*model.Post, error) {
	// 1. 限流检查：防止用户创建帖子过于频繁
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	rateLimitKey := fmt.Sprintf("create_post:user:%d", currentUser.ID)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Hour,
		MaxRequests: 50, // 每小时最多创建50个帖子
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 2. 参数验证
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrInvalidPostTitle
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

	// 6. 使用分布式锁检查slug是否已存在
	slugLockKey := fmt.Sprintf("post_slug:%s", slug)
	slugLock := s.lockManager.GetLock(slugLockKey, 5*time.Second)

	acquired, err := slugLock.AcquireWithRetry(ctx, 3, 100*time.Millisecond)
	if err != nil || !acquired {
		return nil, ErrOperationInProgress
	}
	defer slugLock.Release(ctx)

	// 检查slug是否已存在
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

	// 9. 创建帖子对象
	post := &model.Post{
		Title:      title,
		Slug:       slug,
		Content:    req.Content,
		Summary:    summary,
		UserID:     currentUser.ID,
		AuthorName: currentUser.Name,
		CategoryID: req.CategoryID,
		Visibility: visibility,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 10. 使用分布式事务锁
	txLockKey := fmt.Sprintf("post_create:user:%d", currentUser.ID)
	err = s.lockManager.GetLock(txLockKey, 30*time.Second).Mutex(ctx, func() error {
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
				if err := s.db.WithContext(ctx).Create(postTag).Error; err != nil {
					return fmt.Errorf("关联标签失败: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 11. 获取完整的帖子信息
	fullPost, err := s.getPostWithAssociations(ctx, post.ID)
	if err != nil {
		return nil, fmt.Errorf("获取帖子详情失败: %w", err)
	}

	return fullPost, nil
}

// GetPost 获取帖子详情（带缓存和限流）
func (s *postService) GetPost(ctx context.Context, id uint) (*model.Post, error) {
	// 限流检查：按IP限制获取频率
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_post:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 300, // 每分钟最多300次请求
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	post, err := s.getPostWithAssociations(ctx, id)
	if err != nil {
		return nil, err
	}

	// 异步增加浏览量（不阻塞返回）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.IncrementViews(ctx, id)
	}()

	return post, nil
}

// GetPostBySlug 通过slug获取帖子
func (s *postService) GetPostBySlug(ctx context.Context, slug string) (*model.Post, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_post_slug:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 300,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.IncrementViews(ctx, post.ID)
	}()

	return &post, nil
}

// UpdatePost 更新帖子（带分布式锁）
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

	// 使用分布式锁保护slug更新
	if req.Slug != nil {
		newSlug := utils.SanitizeSlug(*req.Slug)
		if newSlug != "" && newSlug != post.Slug {
			slugLockKey := fmt.Sprintf("post_slug:%s", newSlug)
			slugLock := s.lockManager.GetLock(slugLockKey, 5*time.Second)

			acquired, err := slugLock.AcquireWithRetry(ctx, 3, 100*time.Millisecond)
			if err != nil || !acquired {
				return nil, ErrOperationInProgress
			}
			defer slugLock.Release(ctx)

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

	// 如果没有更新内容，直接返回
	if len(updates) == 0 {
		return s.getPostWithAssociations(ctx, id)
	}

	updates["updated_at"] = time.Now()

	// 4. 使用分布式锁更新帖子
	lockKey := fmt.Sprintf("post_update:%d", id)
	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 更新帖子
		if err := s.postSQL.UpdatePost(ctx, id, updates); err != nil {
			return fmt.Errorf("更新帖子失败: %w", err)
		}

		// 清除缓存
		s.hotPostLock.Lock()
		delete(s.hotPostsCache, id)
		delete(s.hotPostsTTL, id)
		s.hotPostLock.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 5. 获取更新后的帖子
	return s.getPostWithAssociations(ctx, id)
}

// DeletePost 删除帖子（带分布式锁）
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

	// 使用分布式锁保护删除操作
	lockKey := fmt.Sprintf("post_delete:%d", id)
	return s.lockManager.GetLock(lockKey, 30*time.Second).Mutex(ctx, func() error {
		// 清除缓存
		s.hotPostLock.Lock()
		delete(s.hotPostsCache, id)
		delete(s.hotPostsTTL, id)
		s.hotPostLock.Unlock()

		// 删除帖子
		return s.postSQL.DeletePost(ctx, id)
	})
}

// ListPosts 分页列出帖子（带缓存）
func (s *postService) ListPosts(ctx context.Context, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("list_posts:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 500,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, 0, ErrRateLimited
	}

	offset := (page - 1) * size

	var posts []*model.Post
	var total int64

	// 使用读锁保护缓存读取
	s.readCacheLock.RLock()
	defer s.readCacheLock.RUnlock()

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

	// 限流检查：搜索操作比较消耗资源
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("search_posts:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 100,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, 0, ErrRateLimited
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

// LikePost 点赞帖子（完整分布式锁实现）
func (s *postService) LikePost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 用户级限流：防止用户频繁点赞
	userRateLimitKey := fmt.Sprintf("like_post:user:%d", currentUser.ID)
	userRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 60, // 每分钟最多点赞60次
	}

	if err := s.rateLimiter.Allow(ctx, userRateLimitKey, userRateLimitConfig); err != nil {
		return ErrRateLimited
	}

	// 3. 使用用户+帖子级别的分布式锁，防止重复点赞
	lockKey := fmt.Sprintf("post_like:%d:user:%d", postID, currentUser.ID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 4. 检查帖子是否存在
		post, err := s.postSQL.GetPostByID(ctx, postID)
		if err != nil {
			return ErrPostNotFound
		}

		// 5. 检查是否已经点赞过
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

		// 6. 开启事务
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 6.1 保存到MySQL点赞表
			if err := s.likeSQL.InsertLike(ctx, currentUser.ID, postID); err != nil {
				return fmt.Errorf("保存点赞记录失败: %w", err)
			}

			// 6.2 更新帖子点赞数
			updates := map[string]interface{}{
				"liketimes":  post.Liketimes + 1,
				"updated_at": time.Now(),
			}
			if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
				return fmt.Errorf("更新帖子点赞数失败: %w", err)
			}

			// 6.3 保存到Redis缓存
			if err := s.likeCache.Like(ctx, currentUser.ID, postID); err != nil {
				fmt.Printf("Redis点赞缓存失败: %v\n", err)
			}

			// 6.4 清除缓存
			s.hotPostLock.Lock()
			delete(s.hotPostsCache, postID)
			delete(s.hotPostsTTL, postID)
			s.hotPostLock.Unlock()

			return nil
		})

		return err
	})

	return err
}

// UnlikePost 取消点赞（完整分布式锁实现）
func (s *postService) UnlikePost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 使用用户+帖子级别的分布式锁
	lockKey := fmt.Sprintf("post_like:%d:user:%d", postID, currentUser.ID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 3. 检查帖子是否存在
		post, err := s.postSQL.GetPostByID(ctx, postID)
		if err != nil {
			return ErrPostNotFound
		}

		// 4. 检查是否已经点赞过
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

		// 5. 开启事务
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 5.1 从MySQL删除点赞记录
			if err := s.likeSQL.DeleteLike(ctx, currentUser.ID, postID); err != nil {
				return fmt.Errorf("删除点赞记录失败: %w", err)
			}

			// 5.2 更新帖子点赞数
			if post.Liketimes > 0 {
				updates := map[string]interface{}{
					"liketimes":  post.Liketimes - 1,
					"updated_at": time.Now(),
				}
				if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
					return fmt.Errorf("更新帖子点赞数失败: %w", err)
				}
			}

			// 5.3 从Redis缓存删除
			if err := s.likeCache.Unlike(ctx, currentUser.ID, postID); err != nil {
				fmt.Printf("Redis取消点赞缓存失败: %v\n", err)
			}

			// 5.4 清除缓存
			s.hotPostLock.Lock()
			delete(s.hotPostsCache, postID)
			delete(s.hotPostsTTL, postID)
			s.hotPostLock.Unlock()

			return nil
		})

		return err
	})

	return err
}

// GetPostLikes 获取帖子点赞数（带缓存）
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

// StarPost 收藏帖子（完整分布式锁实现）
func (s *postService) StarPost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 用户级限流
	userRateLimitKey := fmt.Sprintf("star_post:user:%d", currentUser.ID)
	userRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 60,
	}

	if err := s.rateLimiter.Allow(ctx, userRateLimitKey, userRateLimitConfig); err != nil {
		return ErrRateLimited
	}

	// 3. 使用用户+帖子级别的分布式锁
	lockKey := fmt.Sprintf("post_star:%d:user:%d", postID, currentUser.ID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 4. 检查帖子是否存在
		post, err := s.postSQL.GetPostByID(ctx, postID)
		if err != nil {
			return ErrPostNotFound
		}

		// 5. 检查是否已经收藏过
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

		// 6. 开启事务
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 6.1 保存到MySQL收藏表
			if err := s.starSQL.InsertStar(ctx, currentUser.ID, postID); err != nil {
				return fmt.Errorf("保存收藏记录失败: %w", err)
			}

			// 6.2 更新帖子收藏数
			updates := map[string]interface{}{
				"staredtimes": post.Staredtimes + 1,
				"updated_at":  time.Now(),
			}
			if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
				return fmt.Errorf("更新帖子收藏数失败: %w", err)
			}

			// 6.3 保存到Redis缓存
			if err := s.starCache.Star(ctx, currentUser.ID, postID); err != nil {
				fmt.Printf("Redis收藏缓存失败: %v\n", err)
			}

			// 6.4 清除缓存
			s.hotPostLock.Lock()
			delete(s.hotPostsCache, postID)
			delete(s.hotPostsTTL, postID)
			s.hotPostLock.Unlock()

			return nil
		})

		return err
	})

	return err
}

// UnstarPost 取消收藏（完整分布式锁实现）
func (s *postService) UnstarPost(ctx context.Context, postID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 使用用户+帖子级别的分布式锁
	lockKey := fmt.Sprintf("post_star:%d:user:%d", postID, currentUser.ID)

	err = s.lockManager.GetLock(lockKey, 10*time.Second).Mutex(ctx, func() error {
		// 3. 检查帖子是否存在
		post, err := s.postSQL.GetPostByID(ctx, postID)
		if err != nil {
			return ErrPostNotFound
		}

		// 4. 检查是否已经收藏过
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

		// 5. 开启事务
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 5.1 从MySQL删除收藏记录
			if err := s.starSQL.DeleteStar(ctx, currentUser.ID, postID); err != nil {
				return fmt.Errorf("删除收藏记录失败: %w", err)
			}

			// 5.2 更新帖子收藏数
			if post.Staredtimes > 0 {
				updates := map[string]interface{}{
					"staredtimes": post.Staredtimes - 1,
					"updated_at":  time.Now(),
				}
				if err := s.postSQL.UpdatePost(ctx, postID, updates); err != nil {
					return fmt.Errorf("更新帖子收藏数失败: %w", err)
				}
			}

			// 5.3 从Redis缓存删除
			if err := s.starCache.Unstar(ctx, currentUser.ID, postID); err != nil {
				fmt.Printf("Redis取消收藏缓存失败: %v\n", err)
			}

			// 5.4 清除缓存
			s.hotPostLock.Lock()
			delete(s.hotPostsCache, postID)
			delete(s.hotPostsTTL, postID)
			s.hotPostLock.Unlock()

			return nil
		})

		return err
	})

	return err
}

// GetPostStars 获取帖子收藏数（带缓存）
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

// GetPostCommentsCount 获取帖子评论数（带缓存）
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

// IncrementComments 增加评论数（带分布式锁）
func (s *postService) IncrementComments(ctx context.Context, postID uint) error {
	// 使用分布式锁
	lockKey := fmt.Sprintf("post_comments:%d", postID)

	return s.lockManager.GetLock(lockKey, 5*time.Second).Mutex(ctx, func() error {
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

			// 清除缓存
			s.hotPostLock.Lock()
			delete(s.hotPostsCache, postID)
			delete(s.hotPostsTTL, postID)
			s.hotPostLock.Unlock()

			return nil
		})

		return err
	})
}

// DecrementComments 减少评论数（带分布式锁）
func (s *postService) DecrementComments(ctx context.Context, postID uint) error {
	// 使用分布式锁
	lockKey := fmt.Sprintf("post_comments:%d", postID)

	return s.lockManager.GetLock(lockKey, 5*time.Second).Mutex(ctx, func() error {
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

			// 清除缓存
			s.hotPostLock.Lock()
			delete(s.hotPostsCache, postID)
			delete(s.hotPostsTTL, postID)
			s.hotPostLock.Unlock()

			return nil
		})

		return err
	})
}

// IncrementViews 增加浏览量（带分布式锁）
func (s *postService) IncrementViews(ctx context.Context, postID uint) error {
	// 使用分布式锁
	lockKey := fmt.Sprintf("post_views:%d", postID)

	return s.lockManager.GetLock(lockKey, 3*time.Second).Mutex(ctx, func() error {
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
	})
}

// GetPostViews 获取帖子浏览量（带缓存）
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

// GetPostStats 获取帖子综合统计数据（带缓存和并行获取）
func (s *postService) GetPostStats(ctx context.Context, postID uint) (*PostStats, error) {
	// 1. 检查帖子是否存在
	post, err := s.postSQL.GetPostByID(ctx, postID)
	if err != nil {
		return nil, ErrPostNotFound
	}

	// 2. 获取当前用户（用于判断是否点赞/收藏）
	currentUser, _ := s.getCurrentUser(ctx) // 忽略错误，游客也可以查看统计

	// 3. 并行获取所有统计信息
	var wg sync.WaitGroup
	var mu sync.Mutex
	var statsErr error

	stats := &PostStats{
		PostID:   postID,
		Likes:    post.Liketimes,
		Stars:    post.Staredtimes,
		Comments: post.CommentNumbers,
		Views:    post.Clicktimes,
	}

	// 如果用户已登录，并行获取点赞和收藏状态
	if currentUser != nil {
		wg.Add(2)

		// 获取点赞状态
		go func() {
			defer wg.Done()
			isLiked, err := s.IsPostLiked(ctx, postID)
			mu.Lock()
			if err != nil {
				statsErr = fmt.Errorf("获取点赞状态失败: %w", err)
			} else {
				stats.IsLiked = isLiked
			}
			mu.Unlock()
		}()

		// 获取收藏状态
		go func() {
			defer wg.Done()
			isStarred, err := s.IsPostStarred(ctx, postID)
			mu.Lock()
			if err != nil {
				statsErr = fmt.Errorf("获取收藏状态失败: %w", err)
			} else {
				stats.IsStarred = isStarred
			}
			mu.Unlock()
		}()

		wg.Wait()
	}

	// 检查是否有错误
	mu.Lock()
	if statsErr != nil {
		return nil, statsErr
	}
	mu.Unlock()

	return stats, nil
}
