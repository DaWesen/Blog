package dao

import (
	"blog/model"
	"context"

	"gorm.io/gorm"
)

type UserSQL interface {
	InsertUser(ctx context.Context, u *model.User) error
	GetUserByID(ctx context.Context, id uint) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)
	UpdateUser(ctx context.Context, id uint, updates map[string]any) error
	DeleteUser(ctx context.Context, id uint) error
}

// 评论
type CommentSQL interface {
	InsertComment(ctx context.Context, c *model.Comment) error
	GetCommentByID(ctx context.Context, id uint) (*model.Comment, error)
	UpdateComment(ctx context.Context, id uint, updates map[string]any) error
	DeleteComment(ctx context.Context, id uint) error
	FindComments(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Comment, error)
}

// 帖子
type PostSQL interface {
	InsertPost(ctx context.Context, p *model.Post) error
	GetPostByID(ctx context.Context, id uint) (*model.Post, error)
	GetPostBySlug(ctx context.Context, slug string) (*model.Post, error)
	UpdatePost(ctx context.Context, id uint, updates map[string]any) error
	DeletePost(ctx context.Context, id uint) error
	FindPosts(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Post, error)
}

// 分类
type CategorySQL interface {
	InsertCategory(ctx context.Context, c *model.Category) error
	GetCategoryByID(ctx context.Context, id uint) (*model.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*model.Category, error)
	UpdateCategory(ctx context.Context, id uint, updates map[string]any) error
	DeleteCategory(ctx context.Context, id uint) error
	FindCategories(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Category, error)
}

// 标签
type TagSQL interface {
	InsertTag(ctx context.Context, t *model.Tag) error
	GetTagByID(ctx context.Context, id uint) (*model.Tag, error)
	GetTagBySlug(ctx context.Context, slug string) (*model.Tag, error)
	UpdateTag(ctx context.Context, id uint, updates map[string]any) error
	DeleteTag(ctx context.Context, id uint) error
	FindTags(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Tag, error)
}

// 关注
type FollowSQL interface {
	InsertFollow(ctx context.Context, userID, followingID uint) error
	DeleteFollow(ctx context.Context, userID, followingID uint) error
	FindFollows(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.UserFollower, error)
}

// 点赞 / 收藏
type LikeSQL interface {
	InsertLike(ctx context.Context, userID, postID uint) error
	DeleteLike(ctx context.Context, userID, postID uint) error
	FindLikes(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.UserLikePost, error)
}

type StarSQL interface {
	InsertStar(ctx context.Context, userID, postID uint) error
	DeleteStar(ctx context.Context, userID, postID uint) error
	FindStars(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.UserStarPost, error)
}

// 用户
type userSQL struct{ db *gorm.DB }

func NewUserSQL(db *gorm.DB) *userSQL { return &userSQL{db: db} }

func (d *userSQL) InsertUser(ctx context.Context, u *model.User) error {
	return d.db.WithContext(ctx).Create(u).Error
}

func (d *userSQL) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	err := d.db.WithContext(ctx).First(&u, id).Error
	return &u, err
}

func (d *userSQL) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	var u model.User
	err := d.db.WithContext(ctx).Where("name = ?", name).First(&u).Error
	return &u, err
}

func (d *userSQL) UpdateUser(ctx context.Context, id uint, updates map[string]any) error {
	return d.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (d *userSQL) DeleteUser(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// 评论
type commentSQL struct{ db *gorm.DB }

func NewCommentSQL(db *gorm.DB) CommentSQL { return &commentSQL{db: db} }

func (d *commentSQL) InsertComment(ctx context.Context, c *model.Comment) error {
	return d.db.WithContext(ctx).Create(c).Error
}

func (d *commentSQL) GetCommentByID(ctx context.Context, id uint) (*model.Comment, error) {
	var c model.Comment
	err := d.db.WithContext(ctx).First(&c, id).Error
	return &c, err
}

func (d *commentSQL) UpdateComment(ctx context.Context, id uint, updates map[string]any) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).Where("id = ?", id).Updates(updates).Error
}

func (d *commentSQL) DeleteComment(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.Comment{}, id).Error
}

func (d *commentSQL) FindComments(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&comments).Error
	return comments, err
}

// 帖子
type postSQL struct{ db *gorm.DB }

func NewPostSQL(db *gorm.DB) PostSQL { return &postSQL{db: db} }

func (d *postSQL) InsertPost(ctx context.Context, p *model.Post) error {
	return d.db.WithContext(ctx).Create(p).Error
}

func (d *postSQL) GetPostByID(ctx context.Context, id uint) (*model.Post, error) {
	var p model.Post
	err := d.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (d *postSQL) GetPostBySlug(ctx context.Context, slug string) (*model.Post, error) {
	var p model.Post
	err := d.db.WithContext(ctx).Where("slug = ?", slug).First(&p).Error
	return &p, err
}

func (d *postSQL) UpdatePost(ctx context.Context, id uint, updates map[string]any) error {
	return d.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Updates(updates).Error
}

func (d *postSQL) DeletePost(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.Post{}, id).Error
}

func (d *postSQL) FindPosts(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Post, error) {
	var posts []*model.Post
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&posts).Error
	return posts, err
}

// 分类
type categorySQL struct{ db *gorm.DB }

func NewCategorySQL(db *gorm.DB) CategorySQL { return &categorySQL{db: db} }

func (d *categorySQL) InsertCategory(ctx context.Context, c *model.Category) error {
	return d.db.WithContext(ctx).Create(c).Error
}

func (d *categorySQL) GetCategoryByID(ctx context.Context, id uint) (*model.Category, error) {
	var c model.Category
	err := d.db.WithContext(ctx).First(&c, id).Error
	return &c, err
}

func (d *categorySQL) GetCategoryBySlug(ctx context.Context, slug string) (*model.Category, error) {
	var c model.Category
	err := d.db.WithContext(ctx).Where("slug = ?", slug).First(&c).Error
	return &c, err
}

func (d *categorySQL) UpdateCategory(ctx context.Context, id uint, updates map[string]any) error {
	return d.db.WithContext(ctx).Model(&model.Category{}).Where("id = ?", id).Updates(updates).Error
}

func (d *categorySQL) DeleteCategory(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.Category{}, id).Error
}

func (d *categorySQL) FindCategories(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Category, error) {
	var categories []*model.Category
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&categories).Error
	return categories, err
}

// 标签
type tagSQL struct{ db *gorm.DB }

func NewTagSQL(db *gorm.DB) TagSQL { return &tagSQL{db: db} }

func (d *tagSQL) InsertTag(ctx context.Context, t *model.Tag) error {
	return d.db.WithContext(ctx).Create(t).Error
}

func (d *tagSQL) GetTagByID(ctx context.Context, id uint) (*model.Tag, error) {
	var t model.Tag
	err := d.db.WithContext(ctx).First(&t, id).Error
	return &t, err
}

func (d *tagSQL) GetTagBySlug(ctx context.Context, slug string) (*model.Tag, error) {
	var t model.Tag
	err := d.db.WithContext(ctx).Where("slug = ?", slug).First(&t).Error
	return &t, err
}

func (d *tagSQL) UpdateTag(ctx context.Context, id uint, updates map[string]any) error {
	return d.db.WithContext(ctx).Model(&model.Tag{}).Where("id = ?", id).Updates(updates).Error
}

func (d *tagSQL) DeleteTag(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.Tag{}, id).Error
}

func (d *tagSQL) FindTags(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Tag, error) {
	var tags []*model.Tag
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&tags).Error
	return tags, err
}

// 关注
type followSQL struct{ db *gorm.DB }

func NewFollowSQL(db *gorm.DB) FollowSQL { return &followSQL{db: db} }

func (d *followSQL) InsertFollow(ctx context.Context, userID, followingID uint) error {
	return d.db.WithContext(ctx).Create(&model.UserFollower{
		UserID:      userID,
		FollowingID: followingID,
	}).Error
}

func (d *followSQL) DeleteFollow(ctx context.Context, userID, followingID uint) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND following_id = ?", userID, followingID).
		Delete(&model.UserFollower{}).Error
}

func (d *followSQL) FindFollows(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.UserFollower, error) {
	var follows []*model.UserFollower
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&follows).Error
	return follows, err
}

// 点赞
type likeSQL struct{ db *gorm.DB }

func NewLikeSQL(db *gorm.DB) LikeSQL { return &likeSQL{db: db} }

func (d *likeSQL) InsertLike(ctx context.Context, userID, postID uint) error {
	return d.db.WithContext(ctx).Create(&model.UserLikePost{
		UserID: userID,
		PostID: postID,
	}).Error
}

func (d *likeSQL) DeleteLike(ctx context.Context, userID, postID uint) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.UserLikePost{}).Error
}

func (d *likeSQL) FindLikes(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.UserLikePost, error) {
	var likes []*model.UserLikePost
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&likes).Error
	return likes, err
}

// 收藏
type starSQL struct{ db *gorm.DB }

func NewStarSQL(db *gorm.DB) StarSQL { return &starSQL{db: db} }

func (d *starSQL) InsertStar(ctx context.Context, userID, postID uint) error {
	return d.db.WithContext(ctx).Create(&model.UserStarPost{
		UserID: userID,
		PostID: postID,
	}).Error
}

func (d *starSQL) DeleteStar(ctx context.Context, userID, postID uint) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.UserStarPost{}).Error
}

func (d *starSQL) FindStars(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.UserStarPost, error) {
	var stars []*model.UserStarPost
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&stars).Error
	return stars, err
}
