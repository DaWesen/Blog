package service

import (
	mysql "blog/dao/mysql"
	"blog/model"
	"blog/utils"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// 错误定义
var (
	ErrFollowYourself   = errors.New("不能关注自己")
	ErrAlreadyFollowing = errors.New("已经关注该用户")
	ErrNotFollowing     = errors.New("未关注该用户")
	ErrUserNotFound     = errors.New("用户不存在")
	ErrRateLimited      = errors.New("点击频率过快")
)

// FollowService 关注服务接口
type FollowService interface {
	// 关注功能
	FollowUser(ctx context.Context, followingID uint) error
	UnfollowUser(ctx context.Context, followingID uint) error
	IsFollowing(ctx context.Context, followingID uint) (bool, error)

	// 列表功能
	GetFollowingList(ctx context.Context, userID uint, page, size int) ([]*model.User, int64, error)
	GetFollowerList(ctx context.Context, userID uint, page, size int) ([]*model.User, int64, error)

	// 统计功能
	GetFollowingCount(ctx context.Context, userID uint) (uint, error)
	GetFollowerCount(ctx context.Context, userID uint) (uint, error)
	GetFollowStats(ctx context.Context, userID uint) (*FollowStats, error)
}

// 统计数据结构
type FollowStats struct {
	FollowingCount uint `json:"following_count"`
	FollowerCount  uint `json:"follower_count"`
	IsFollowing    bool `json:"is_following"`
}

// followService 关注服务实现
type followService struct {
	followSQL mysql.FollowSQL
	userSQL   mysql.UserSQL
	db        *gorm.DB
}

// NewFollowService 创建关注服务
func NewFollowService(
	followSQL mysql.FollowSQL,
	userSQL mysql.UserSQL,
	db *gorm.DB,
) FollowService {
	return &followService{
		followSQL: followSQL,
		userSQL:   userSQL,
		db:        db,
	}
}

// getCurrentUser 获取当前用户
func (s *followService) getCurrentUser(ctx context.Context) (*model.User, error) {
	userID, err := utils.GetCurrentUserIDFromContext(ctx)
	if err != nil {
		return nil, errors.New("用户未认证")
	}

	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// FollowUser 关注用户
func (s *followService) FollowUser(ctx context.Context, followingID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 检查是否关注自己
	if currentUser.ID == followingID {
		return ErrFollowYourself
	}

	// 3. 检查被关注用户是否存在
	_, err = s.userSQL.GetUserByID(ctx, followingID)
	if err != nil {
		return ErrUserNotFound
	}

	// 4. 检查是否已经关注
	follows, err := s.followSQL.FindFollows(ctx, "user_id = ? AND following_id = ?", currentUser.ID, followingID)
	if err != nil {
		return fmt.Errorf("检查关注关系失败: %w", err)
	}

	if len(follows) > 0 {
		return ErrAlreadyFollowing
	}

	// 5. 创建关注关系
	if err := s.followSQL.InsertFollow(ctx, currentUser.ID, followingID); err != nil {
		return fmt.Errorf("创建关注关系失败: %w", err)
	}

	return nil
}

// UnfollowUser 取消关注
func (s *followService) UnfollowUser(ctx context.Context, followingID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 检查是否关注自己
	if currentUser.ID == followingID {
		return ErrFollowYourself
	}

	// 3. 检查被关注用户是否存在
	_, err = s.userSQL.GetUserByID(ctx, followingID)
	if err != nil {
		return ErrUserNotFound
	}

	// 4. 检查是否已经关注
	follows, err := s.followSQL.FindFollows(ctx, "user_id = ? AND following_id = ?", currentUser.ID, followingID)
	if err != nil {
		return fmt.Errorf("检查关注关系失败: %w", err)
	}

	if len(follows) == 0 {
		return ErrNotFollowing
	}

	// 5. 删除关注关系
	if err := s.followSQL.DeleteFollow(ctx, currentUser.ID, followingID); err != nil {
		return fmt.Errorf("删除关注关系失败: %w", err)
	}

	return nil
}

// IsFollowing 检查是否关注
func (s *followService) IsFollowing(ctx context.Context, followingID uint) (bool, error) {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return false, err
	}

	// 2. 检查是否关注自己
	if currentUser.ID == followingID {
		return false, nil
	}

	// 3. 从数据库检查
	follows, err := s.followSQL.FindFollows(ctx, "user_id = ? AND following_id = ?", currentUser.ID, followingID)
	if err != nil {
		return false, fmt.Errorf("检查关注关系失败: %w", err)
	}

	return len(follows) > 0, nil
}

// GetFollowingList 获取关注列表
func (s *followService) GetFollowingList(ctx context.Context, userID uint, page, size int) ([]*model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	// 从数据库获取
	var followRelations []*model.UserFollower
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&followRelations).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取关注列表失败: %w", err)
	}

	// 获取总数
	var total int64
	err = s.db.WithContext(ctx).
		Model(&model.UserFollower{}).
		Where("user_id = ?", userID).
		Count(&total).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取关注总数失败: %w", err)
	}

	if len(followRelations) == 0 {
		return []*model.User{}, total, nil
	}

	// 获取用户信息
	var followingIDs []uint
	for _, rel := range followRelations {
		followingIDs = append(followingIDs, rel.FollowingID)
	}

	var users []*model.User
	err = s.db.WithContext(ctx).
		Select("id, name, avatar_url, bio").
		Where("id IN ?", followingIDs).
		Find(&users).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取用户信息失败: %w", err)
	}

	return users, total, nil
}

// GetFollowerList 获取粉丝列表
func (s *followService) GetFollowerList(ctx context.Context, userID uint, page, size int) ([]*model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	// 从数据库获取
	var followerRelations []*model.UserFollower
	err := s.db.WithContext(ctx).
		Where("following_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&followerRelations).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取粉丝列表失败: %w", err)
	}

	// 获取总数
	var total int64
	err = s.db.WithContext(ctx).
		Model(&model.UserFollower{}).
		Where("following_id = ?", userID).
		Count(&total).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取粉丝总数失败: %w", err)
	}

	if len(followerRelations) == 0 {
		return []*model.User{}, total, nil
	}

	// 获取用户信息
	var followerIDs []uint
	for _, rel := range followerRelations {
		followerIDs = append(followerIDs, rel.UserID)
	}

	var users []*model.User
	err = s.db.WithContext(ctx).
		Select("id, name, avatar_url, bio").
		Where("id IN ?", followerIDs).
		Find(&users).Error

	if err != nil {
		return nil, 0, fmt.Errorf("获取用户信息失败: %w", err)
	}

	return users, total, nil
}

// GetFollowingCount 获取关注数
func (s *followService) GetFollowingCount(ctx context.Context, userID uint) (uint, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&model.UserFollower{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("获取关注数失败: %w", err)
	}

	return uint(count), nil
}

// GetFollowerCount 获取粉丝数
func (s *followService) GetFollowerCount(ctx context.Context, userID uint) (uint, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&model.UserFollower{}).
		Where("following_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("获取粉丝数失败: %w", err)
	}

	return uint(count), nil
}

// GetFollowStats 获取关注统计
func (s *followService) GetFollowStats(ctx context.Context, userID uint) (*FollowStats, error) {
	stats := &FollowStats{}

	// 获取关注数
	followingCount, err := s.GetFollowingCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.FollowingCount = followingCount

	// 获取粉丝数
	followerCount, err := s.GetFollowerCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.FollowerCount = followerCount

	// 获取当前用户是否关注此用户
	currentUser, err := s.getCurrentUser(ctx)
	if err == nil && currentUser.ID != userID {
		isFollowing, _ := s.IsFollowing(ctx, userID)
		stats.IsFollowing = isFollowing
	}

	return stats, nil
}
