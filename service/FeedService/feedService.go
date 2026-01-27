package service

import (
	mysql "blog/dao/mysql"
	"blog/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// FeedService 动态流服务接口（简化版）
type FeedService interface {
	// 获取用户动态流（关注的人的文章）
	GetUserFeed(ctx context.Context, userID uint, page, size int) ([]*model.Post, int64, error)

	// 获取热门文章（简化热度算法）
	GetHotPosts(ctx context.Context, page, size int) ([]*model.Post, int64, error)
}

// feedService 动态流服务实现
type feedService struct {
	db        *gorm.DB
	followSQL mysql.FollowSQL
	postSQL   mysql.PostSQL
}

// NewFeedService 创建动态流服务
func NewFeedService(db *gorm.DB, followSQL mysql.FollowSQL, postSQL mysql.PostSQL) FeedService {
	return &feedService{
		db:        db,
		followSQL: followSQL,
		postSQL:   postSQL,
	}
}

// GetUserFeed 获取用户动态流（关注的人的文章）
func (s *feedService) GetUserFeed(ctx context.Context, userID uint, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}
	offset := (page - 1) * size

	// 获取用户关注列表
	following, err := s.followSQL.FindFollows(ctx, "user_id = ?", userID)
	if err != nil {
		return nil, 0, fmt.Errorf("获取关注列表失败: %w", err)
	}

	if len(following) == 0 {
		return []*model.Post{}, 0, nil
	}

	// 提取关注用户ID
	var followingIDs []uint
	for _, f := range following {
		followingIDs = append(followingIDs, f.FollowingID)
	}

	// 获取关注用户的文章
	var posts []*model.Post
	var total int64

	// 获取总数
	err = s.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("user_id IN ? AND visibility = ?", followingIDs, model.VisibilityPublic).
		Count(&total).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取文章总数失败: %w", err)
	}

	// 获取文章列表
	err = s.db.WithContext(ctx).
		Where("user_id IN ? AND visibility = ?", followingIDs, model.VisibilityPublic).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Category").
		Find(&posts).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取文章失败: %w", err)
	}

	return posts, total, nil
}

// GetHotPosts 获取热门文章（简化版）
func (s *feedService) GetHotPosts(ctx context.Context, page, size int) ([]*model.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}
	offset := (page - 1) * size

	var posts []*model.Post
	var total int64

	// 获取总数
	err := s.db.WithContext(ctx).
		Model(&model.Post{}).
		Where("visibility = ?", model.VisibilityPublic).
		Count(&total).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取文章总数失败: %w", err)
	}
	err = s.db.WithContext(ctx).
		Where("visibility = ?", model.VisibilityPublic).
		Order("(liketimes + staredtimes + comment_numbers) DESC, created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&posts).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取热门文章失败: %w", err)
	}
	return posts, total, nil
}
