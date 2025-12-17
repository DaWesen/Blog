package service

import (
	dao "blog/dao/mysql"
	"blog/model"
	"blog/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrCategoryExists      = errors.New("分类名称已存在")
	ErrInvalidCategoryName = errors.New("分类名称不能为空")
	ErrCategoryNotFound    = errors.New("分类不存在")
	ErrRateLimited         = errors.New("操作过于频繁，请稍后再试")
)

type CategoryService interface {
	CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*model.Category, error)
	GetCategory(ctx context.Context, id uint) (*model.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*model.Category, error)
	UpdateCategory(ctx context.Context, id uint, req *UpdateCategoryRequest) (*model.Category, error)
	DeleteCategory(ctx context.Context, id uint) error
	ListCategories(ctx context.Context, page, size int) ([]*model.Category, int64, error)
	SearchCategories(ctx context.Context, keyword string) ([]*model.Category, error)
}

type categoryService struct {
	categorySQL dao.CategorySQL

	// 分布式锁管理器
	lockManager *utils.LockManager

	// 限流器
	rateLimiter *utils.RateLimiter

	// 缓存
	categoryCache     map[uint]*model.Category
	categoryCacheTTL  map[uint]time.Time
	categoryCacheLock sync.RWMutex
	slugToID          map[string]uint
	slugLock          sync.RWMutex
	readCacheLock     sync.RWMutex
}

func NewCategoryService(categorySQL dao.CategorySQL, lockManager *utils.LockManager, rateLimiter *utils.RateLimiter) CategoryService {
	return &categoryService{
		categorySQL:      categorySQL,
		lockManager:      lockManager,
		rateLimiter:      rateLimiter,
		categoryCache:    make(map[uint]*model.Category),
		categoryCacheTTL: make(map[uint]time.Time),
		slugToID:         make(map[string]uint),
	}
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Slug string `json:"slug,omitempty" binding:"omitempty,min=1,max=100"`
}

// CreateCategory 创建分类（带分布式锁和限流）
func (s *categoryService) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*model.Category, error) {
	// 1. IP级别限流
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("create_category:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Hour,
		MaxRequests: 50, // 每小时最多创建50个分类
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 2. 参数验证
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrInvalidCategoryName
	}

	// 3. 处理slug
	slug := ""
	if req.Slug != "" {
		slug = utils.SanitizeSlug(req.Slug)
	} else {
		slug = utils.GenerateSlug(name)
	}

	// 4. 清除可能存在的缓存残留
	s.categoryCacheLock.Lock()
	delete(s.categoryCache, 0) // 清除可能存在的无效条目
	s.categoryCacheLock.Unlock()

	s.slugLock.Lock()
	delete(s.slugToID, slug) // 清除该slug的缓存映射
	s.slugLock.Unlock()

	// 5. 使用分布式锁检查分类是否已存在
	slugLockKey := fmt.Sprintf("category_slug:%s", slug)
	nameLockKey := fmt.Sprintf("category_name:%s", name)

	var category *model.Category

	// 同时获取两个锁
	err := s.lockManager.GetLock(slugLockKey, 5*time.Second).Mutex(ctx, func() error {
		return s.lockManager.GetLock(nameLockKey, 5*time.Second).Mutex(ctx, func() error {
			// 重新从数据库检查，忽略缓存
			existingBySlug, err := s.categorySQL.GetCategoryBySlug(ctx, slug)
			if err != nil {
				// 如果是"record not found"错误，说明slug不存在，这是正常情况
				if err.Error() == "record not found" || strings.Contains(err.Error(), "not found") {
					// 继续检查name
				} else {
					return fmt.Errorf("检查分类slug失败: %w", err)
				}
			} else if existingBySlug != nil {
				return ErrCategoryExists
			}

			// 检查name是否已存在
			existingByName, err := s.categorySQL.FindCategories(ctx, "name = ?", name)
			if err != nil {
				return fmt.Errorf("检查分类name失败: %w", err)
			}
			if len(existingByName) > 0 {
				return ErrCategoryExists
			}

			// 5. 创建分类对象
			category = &model.Category{
				Name:      name,
				Slug:      slug,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// 6. 保存到数据库
			if err := s.categorySQL.InsertCategory(ctx, category); err != nil {
				if strings.Contains(err.Error(), "Duplicate entry") {
					// 如果是唯一约束冲突，从数据库重新检查
					existingBySlug, _ := s.categorySQL.GetCategoryBySlug(ctx, slug)
					if existingBySlug != nil {
						return ErrCategoryExists
					}
					existingByName, _ := s.categorySQL.FindCategories(ctx, "name = ?", name)
					if len(existingByName) > 0 {
						return ErrCategoryExists
					}
					return fmt.Errorf("未知的唯一约束冲突: %w", err)
				}
				return fmt.Errorf("创建分类失败: %w", err)
			}

			// 7. 更新缓存
			s.cacheCategory(category)
			s.slugLock.Lock()
			s.slugToID[slug] = category.ID
			s.slugLock.Unlock()

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return category, nil
}

// GetCategory 获取分类详情（带缓存和限流）
func (s *categoryService) GetCategory(ctx context.Context, id uint) (*model.Category, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_category:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 500,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 首先尝试从缓存获取
	if category, ok := s.getCachedCategory(ctx, id); ok {
		return category, nil
	}

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("category_query:%d", id)
	var category *model.Category

	err := s.lockManager.GetLock(lockKey, 3*time.Second).Mutex(ctx, func() error {
		// 再次检查缓存
		if cachedCategory, ok := s.getCachedCategory(ctx, id); ok {
			category = cachedCategory
			return nil
		}

		// 从数据库获取
		var err error
		category, err = s.categorySQL.GetCategoryByID(ctx, id)
		if err != nil {
			return ErrCategoryNotFound
		}

		// 更新缓存
		s.cacheCategory(category)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return category, nil
}

// GetCategoryBySlug 通过slug获取分类（带缓存和限流）
func (s *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*model.Category, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_category_slug:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 500,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 首先尝试从slug映射获取
	s.slugLock.RLock()
	if categoryID, ok := s.slugToID[slug]; ok {
		s.slugLock.RUnlock()
		if category, ok := s.getCachedCategory(ctx, categoryID); ok {
			return category, nil
		}
	} else {
		s.slugLock.RUnlock()
	}

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("category_by_slug:%s", slug)
	var category *model.Category

	err := s.lockManager.GetLock(lockKey, 3*time.Second).Mutex(ctx, func() error {
		// 从数据库获取
		var err error
		category, err = s.categorySQL.GetCategoryBySlug(ctx, slug)
		if err != nil {
			return ErrCategoryNotFound
		}

		// 更新缓存
		s.cacheCategory(category)
		s.slugLock.Lock()
		s.slugToID[slug] = category.ID
		s.slugLock.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return category, nil
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Slug *string `json:"slug,omitempty" binding:"omitempty,min=1,max=100"`
}

// UpdateCategory 更新分类（带分布式锁）
func (s *categoryService) UpdateCategory(ctx context.Context, id uint, req *UpdateCategoryRequest) (*model.Category, error) {
	// 1. 获取现有分类
	category, err := s.GetCategory(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	// 2. 构建更新数据
	updates := make(map[string]interface{})

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName != "" && newName != category.Name {
			updates["name"] = newName
		}
	}

	// 处理slug更新（需要分布式锁）
	if req.Slug != nil {
		newSlug := utils.SanitizeSlug(*req.Slug)
		if newSlug != "" && newSlug != category.Slug {
			// 使用分布式锁检查新slug是否已被其他分类使用
			slugLockKey := fmt.Sprintf("category_slug:%s", newSlug)
			err = s.lockManager.GetLock(slugLockKey, 5*time.Second).Mutex(ctx, func() error {
				existing, _ := s.categorySQL.GetCategoryBySlug(ctx, newSlug)
				if existing != nil && existing.ID != id {
					return ErrCategoryExists
				}
				updates["slug"] = newSlug
				return nil
			})

			if err != nil {
				return nil, err
			}
		}
	}

	// 如果没有更新内容，直接返回
	if len(updates) == 0 {
		return category, nil
	}

	updates["updated_at"] = time.Now()

	// 3. 使用分布式锁更新分类
	updateLockKey := fmt.Sprintf("category_update:%d", id)
	err = s.lockManager.GetLock(updateLockKey, 10*time.Second).Mutex(ctx, func() error {
		// 更新数据库
		if err := s.categorySQL.UpdateCategory(ctx, id, updates); err != nil {
			return fmt.Errorf("更新分类失败: %w", err)
		}

		// 清除缓存
		s.categoryCacheLock.Lock()
		delete(s.categoryCache, id)
		delete(s.categoryCacheTTL, id)
		s.categoryCacheLock.Unlock()

		s.slugLock.Lock()
		delete(s.slugToID, category.Slug)
		s.slugLock.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 4. 获取更新后的分类
	return s.GetCategory(ctx, id)
}

// DeleteCategory 删除分类（带分布式锁）
func (s *categoryService) DeleteCategory(ctx context.Context, id uint) error {
	// 先检查是否存在
	category, err := s.GetCategory(ctx, id)
	if err != nil {
		return ErrCategoryNotFound
	}

	// 使用分布式锁保护删除操作
	lockKey := fmt.Sprintf("category_delete:%d", id)

	return s.lockManager.GetLock(lockKey, 15*time.Second).Mutex(ctx, func() error {
		// 清除缓存
		s.categoryCacheLock.Lock()
		delete(s.categoryCache, id)
		delete(s.categoryCacheTTL, id)
		s.categoryCacheLock.Unlock()

		s.slugLock.Lock()
		delete(s.slugToID, category.Slug)
		s.slugLock.Unlock()

		// 删除分类
		return s.categorySQL.DeleteCategory(ctx, id)
	})
}

// ListCategories 分页列出分类（带缓存和限流）
func (s *categoryService) ListCategories(ctx context.Context, page, size int) ([]*model.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("list_categories:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 300,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, 0, ErrRateLimited
	}

	offset := (page - 1) * size

	// 使用读锁保护缓存读取
	s.readCacheLock.RLock()
	defer s.readCacheLock.RUnlock()

	// 获取总数
	total, err := s.categorySQL.CountCategories(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 查询分类
	categories, err := s.categorySQL.FindCategories(ctx, "1 = 1 ORDER BY created_at DESC LIMIT ? OFFSET ?", size, offset)
	if err != nil {
		return nil, 0, err
	}

	// 更新缓存
	for _, category := range categories {
		s.cacheCategory(category)
		s.slugLock.Lock()
		s.slugToID[category.Slug] = category.ID
		s.slugLock.Unlock()
	}

	return categories, total, nil
}

// SearchCategories 搜索分类（带限流）
func (s *categoryService) SearchCategories(ctx context.Context, keyword string) ([]*model.Category, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("search_categories:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 200,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return s.categorySQL.FindCategories(ctx, "1 = 1 ORDER BY created_at DESC")
	}

	searchPattern := "%" + keyword + "%"
	categories, err := s.categorySQL.FindCategories(ctx, "name LIKE ? OR slug LIKE ? ORDER BY created_at DESC", searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	for _, category := range categories {
		s.cacheCategory(category)
		s.slugLock.Lock()
		s.slugToID[category.Slug] = category.ID
		s.slugLock.Unlock()
	}

	return categories, nil
}

// 辅助方法
func (s *categoryService) getCachedCategory(ctx context.Context, id uint) (*model.Category, bool) {
	s.categoryCacheLock.RLock()
	defer s.categoryCacheLock.RUnlock()

	if category, ok := s.categoryCache[id]; ok {
		if s.categoryCacheTTL[id].After(time.Now()) {
			return category, true
		}
	}
	return nil, false
}

func (s *categoryService) cacheCategory(category *model.Category) {
	s.categoryCacheLock.Lock()
	defer s.categoryCacheLock.Unlock()

	s.categoryCache[category.ID] = category
	s.categoryCacheTTL[category.ID] = time.Now().Add(15 * time.Minute) // 缓存15分钟
}
