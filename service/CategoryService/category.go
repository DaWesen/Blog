// service/category_service.go
package service

import (
	dao "blog/dao/mysql"
	"blog/model"
	"blog/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrCategoryExists      = errors.New("分类名称已存在")
	ErrInvalidCategoryName = errors.New("分类名称不能为空")
	ErrCategoryNotFound    = errors.New("分类不存在")
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
}

func NewCategoryService(categorySQL dao.CategorySQL) CategoryService {
	return &categoryService{
		categorySQL: categorySQL,
	}
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Slug string `json:"slug,omitempty" binding:"omitempty,min=1,max=100"`
}

// CreateCategory 创建分类（使用DAO接口）
func (s *categoryService) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*model.Category, error) {
	// 1. 参数验证
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrInvalidCategoryName
	}

	// 2. 处理slug（如果没传则自动生成）
	slug := ""
	if req.Slug != "" {
		slug = utils.SanitizeSlug(req.Slug)
	} else {
		slug = utils.GenerateSlug(name)
	}

	// 3. 检查分类是否已存在
	// 方法A：通过名称检查
	existing, _ := s.categorySQL.GetCategoryBySlug(ctx, slug)
	if existing != nil {
		return nil, ErrCategoryExists
	}

	// 4. 创建分类对象
	category := &model.Category{
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 5. 通过DAO保存到数据库
	if err := s.categorySQL.InsertCategory(ctx, category); err != nil {
		// 处理可能的唯一约束错误（虽然前面检查了，但并发情况下仍可能发生）
		if strings.Contains(err.Error(), "Duplicate entry") {
			// 快速重新检查
			existing, _ := s.categorySQL.GetCategoryBySlug(ctx, slug)
			if existing != nil {
				return nil, ErrCategoryExists
			}
		}
		return nil, fmt.Errorf("创建分类失败: %w", err)
	}

	return category, nil
}

// GetCategory 获取分类详情
func (s *categoryService) GetCategory(ctx context.Context, id uint) (*model.Category, error) {
	category, err := s.categorySQL.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

// GetCategoryBySlug 通过slug获取分类
func (s *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*model.Category, error) {
	category, err := s.categorySQL.GetCategoryBySlug(ctx, slug)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Slug *string `json:"slug,omitempty" binding:"omitempty,min=1,max=100"`
}

// UpdateCategory 更新分类
func (s *categoryService) UpdateCategory(ctx context.Context, id uint, req *UpdateCategoryRequest) (*model.Category, error) {
	// 1. 获取现有分类
	category, err := s.categorySQL.GetCategoryByID(ctx, id)
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

	if req.Slug != nil {
		newSlug := utils.SanitizeSlug(*req.Slug)
		if newSlug != "" && newSlug != category.Slug {
			// 检查新slug是否已被其他分类使用
			existing, _ := s.categorySQL.GetCategoryBySlug(ctx, newSlug)
			if existing != nil && existing.ID != id {
				return nil, ErrCategoryExists
			}
			updates["slug"] = newSlug
		}
	}

	// 如果没有更新内容，直接返回
	if len(updates) == 0 {
		return category, nil
	}

	updates["updated_at"] = time.Now()

	// 3. 通过DAO更新
	if err := s.categorySQL.UpdateCategory(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("更新分类失败: %w", err)
	}

	// 4. 获取更新后的分类
	return s.categorySQL.GetCategoryByID(ctx, id)
}

// DeleteCategory 删除分类
func (s *categoryService) DeleteCategory(ctx context.Context, id uint) error {
	// 先检查是否存在
	_, err := s.categorySQL.GetCategoryByID(ctx, id)
	if err != nil {
		return ErrCategoryNotFound
	}

	return s.categorySQL.DeleteCategory(ctx, id)
}

// ListCategories 分页列出分类
func (s *categoryService) ListCategories(ctx context.Context, page, size int) ([]*model.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	// 使用DAO的通用查询接口
	categories, err := s.categorySQL.FindCategories(ctx, "1 = 1 ORDER BY created_at DESC LIMIT ? OFFSET ?", size, offset)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	total, err := s.categorySQL.CountCategories(ctx)
	return categories, total, nil
}

// SearchCategories 搜索分类
func (s *categoryService) SearchCategories(ctx context.Context, keyword string) ([]*model.Category, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return s.categorySQL.FindCategories(ctx, "1 = 1 ORDER BY created_at DESC")
	}

	searchPattern := "%" + keyword + "%"
	return s.categorySQL.FindCategories(ctx, "name LIKE ? OR slug LIKE ? ORDER BY created_at DESC", searchPattern, searchPattern)
}
