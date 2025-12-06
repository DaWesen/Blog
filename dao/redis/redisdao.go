package dao

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// 接口
type ViewCache interface {
	IncrViewCount(ctx context.Context, postID uint) error
	GetViewCount(ctx context.Context, postID uint) (int64, error)
}

type LikeCache interface {
	IsLiked(ctx context.Context, userID, postID uint) (bool, error)
	Like(ctx context.Context, userID, postID uint) error
	Unlike(ctx context.Context, userID, postID uint) error
	CountLikes(ctx context.Context, postID uint) (int64, error)
}

type StarCache interface {
	IsStarred(ctx context.Context, userID, postID uint) (bool, error)
	Star(ctx context.Context, userID, postID uint) error
	Unstar(ctx context.Context, userID, postID uint) error
	CountStars(ctx context.Context, postID uint) (int64, error)
}

type CommentCache interface {
	// 评论计数
	IncrCommentCount(ctx context.Context, postID uint) error
	DecrCommentCount(ctx context.Context, postID uint) error
	GetCommentCount(ctx context.Context, postID uint) (int64, error)

	// 评论点赞
	IsCommentLiked(ctx context.Context, userID, commentID uint) (bool, error)
	LikeComment(ctx context.Context, userID, commentID uint) error
	UnlikeComment(ctx context.Context, userID, commentID uint) error
	CountCommentLikes(ctx context.Context, commentID uint) (int64, error)
	DeleteCommentLikeCache(ctx context.Context, commentID uint) error
}

type redisCache struct{ rdb redis.UniversalClient }

var (
	_ ViewCache    = (*redisCache)(nil)
	_ LikeCache    = (*redisCache)(nil)
	_ StarCache    = (*redisCache)(nil)
	_ CommentCache = (*redisCache)(nil)
)

func NewRedisCache(rdb redis.UniversalClient) *redisCache {
	return &redisCache{rdb: rdb}
}

// 阅读
func (c *redisCache) IncrViewCount(ctx context.Context, postID uint) error {
	return c.rdb.Incr(ctx, fmt.Sprintf("post:%d:views", postID)).Err()
}

func (c *redisCache) GetViewCount(ctx context.Context, postID uint) (int64, error) {
	return c.rdb.Get(ctx, fmt.Sprintf("post:%d:views", postID)).Int64()
}

// 帖子点赞
func (c *redisCache) IsLiked(ctx context.Context, userID, postID uint) (bool, error) {
	return c.rdb.SIsMember(ctx, fmt.Sprintf("post:%d:likes", postID), userID).Result()
}

func (c *redisCache) Like(ctx context.Context, userID, postID uint) error {
	return c.rdb.SAdd(ctx, fmt.Sprintf("post:%d:likes", postID), userID).Err()
}

func (c *redisCache) Unlike(ctx context.Context, userID, postID uint) error {
	return c.rdb.SRem(ctx, fmt.Sprintf("post:%d:likes", postID), userID).Err()
}

func (c *redisCache) CountLikes(ctx context.Context, postID uint) (int64, error) {
	return c.rdb.SCard(ctx, fmt.Sprintf("post:%d:likes", postID)).Result()
}

// 帖子收藏
func (c *redisCache) IsStarred(ctx context.Context, userID, postID uint) (bool, error) {
	return c.rdb.SIsMember(ctx, fmt.Sprintf("post:%d:stars", postID), userID).Result()
}

func (c *redisCache) Star(ctx context.Context, userID, postID uint) error {
	return c.rdb.SAdd(ctx, fmt.Sprintf("post:%d:stars", postID), userID).Err()
}

func (c *redisCache) Unstar(ctx context.Context, userID, postID uint) error {
	return c.rdb.SRem(ctx, fmt.Sprintf("post:%d:stars", postID), userID).Err()
}

func (c *redisCache) CountStars(ctx context.Context, postID uint) (int64, error) {
	return c.rdb.SCard(ctx, fmt.Sprintf("post:%d:stars", postID)).Result()
}

// 评论计数
func (c *redisCache) IncrCommentCount(ctx context.Context, postID uint) error {
	return c.rdb.Incr(ctx, fmt.Sprintf("post:%d:commentCount", postID)).Err()
}

func (c *redisCache) DecrCommentCount(ctx context.Context, postID uint) error {
	return c.rdb.Decr(ctx, fmt.Sprintf("post:%d:commentCount", postID)).Err()
}

func (c *redisCache) GetCommentCount(ctx context.Context, postID uint) (int64, error) {
	return c.rdb.Get(ctx, fmt.Sprintf("post:%d:commentCount", postID)).Int64()
}

// 评论点赞
func (c *redisCache) IsCommentLiked(ctx context.Context, userID, commentID uint) (bool, error) {
	return c.rdb.SIsMember(ctx, fmt.Sprintf("comment:%d:likes", commentID), userID).Result()
}

func (c *redisCache) LikeComment(ctx context.Context, userID, commentID uint) error {
	return c.rdb.SAdd(ctx, fmt.Sprintf("comment:%d:likes", commentID), userID).Err()
}

func (c *redisCache) UnlikeComment(ctx context.Context, userID, commentID uint) error {
	return c.rdb.SRem(ctx, fmt.Sprintf("comment:%d:likes", commentID), userID).Err()
}

func (c *redisCache) CountCommentLikes(ctx context.Context, commentID uint) (int64, error) {
	return c.rdb.SCard(ctx, fmt.Sprintf("comment:%d:likes", commentID)).Result()
}
func (c *redisCache) DeleteCommentLikeCache(ctx context.Context, commentID uint) error {
	return c.rdb.Del(ctx, fmt.Sprintf("comment:%d:likes", commentID)).Err()
}
