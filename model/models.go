package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	// 个人基础模型
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Email     string `json:"email" gorm:"type:varchar(191);not null;uniqueIndex"`
	AvatarURL string `json:"avatar_url" gorm:"type:varchar(500)"`
	Bio       string `json:"bio" gorm:"type:text"`
	RealIP    string `json:"real_ip" gorm:"type:varchar(45)"`

	// 安全模型
	Password string    `json:"-" gorm:"type:varchar(255);not null"`
	LoginAt  time.Time `json:"login_at" gorm:"autoUpdateTime"`
	LoginIP  string    `json:"login_ip" gorm:"type:varchar(45)"`

	// 状态模型
	Status   UserStatus `json:"status" gorm:"type:varchar(20);default:'active';index"`
	Relation UserRole   `json:"relation" gorm:"type:varchar(20);default:'user';index"`

	// 时间动向模型
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	UserPost []Post    `json:"user_post,omitempty" gorm:"foreignKey:UserID"`
	Follower []*User   `json:"follower,omitempty" gorm:"many2many:user_followers;foreignKey:ID;joinForeignKey:FollowingID;joinReferences:UserID"`
	Fans     []*User   `json:"fans,omitempty" gorm:"many2many:user_followers;foreignKey:ID;joinForeignKey:UserID;joinReferences:FollowingID"`
	StarPost []*Post   `json:"star_post,omitempty" gorm:"many2many:user_star_posts;foreignKey:ID;joinForeignKey:UserID;joinReferences:PostID"`
	LikePost []*Post   `json:"like_post,omitempty" gorm:"many2many:user_like_posts;foreignKey:ID;joinForeignKey:UserID;joinReferences:PostID"`
	Posts    []Post    `json:"posts,omitempty" gorm:"foreignKey:UserID"`
	Comments []Comment `json:"comments,omitempty" gorm:"foreignKey:UserID"`
}

type Post struct {
	// 帖子基础模型
	ID      uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Title   string `json:"title" gorm:"type:varchar(255);not null;index"`
	Slug    string `json:"slug" gorm:"type:varchar(255);not null;uniqueIndex"`
	Summary string `json:"summary" gorm:"type:text"`

	// 内容
	Content  string `json:"content,omitempty" gorm:"type:longtext"`  // 原始内容
	Rendered string `json:"rendered,omitempty" gorm:"type:longtext"` // 渲染后的HTML

	// 作者
	UserID     uint   `json:"user_id" gorm:"index;not null"`
	AuthorName string `json:"author_name" gorm:"type:varchar(100)"`
	Author     *User  `json:"author,omitempty" gorm:"foreignKey:UserID"`

	// 分类
	CategoryID uint      `json:"category_id" gorm:"index"`
	Category   *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`

	// 标签
	Tags []Tag `json:"tags,omitempty" gorm:"many2many:post_tags;"`

	// 时间动向模型
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 作者与读者互动
	Clicktimes     uint `json:"clicktimes" gorm:"default:0"`
	Liketimes      uint `json:"liketimes" gorm:"default:0"`
	Staredtimes    uint `json:"staredtimes" gorm:"default:0"`
	CommentNumbers uint `json:"comment_numbers" gorm:"default:0"`

	// 可见性
	Visibility Visibility `json:"visibility" gorm:"type:varchar(20);default:'public';index"`

	// 关联关系
	StarredBy []*User   `json:"starred_by,omitempty" gorm:"many2many:user_star_posts;foreignKey:ID;joinForeignKey:PostID;joinReferences:UserID"`
	LikedBy   []*User   `json:"liked_by,omitempty" gorm:"many2many:user_like_posts;foreignKey:ID;joinForeignKey:PostID;joinReferences:UserID"`
	Comments  []Comment `json:"comments,omitempty" gorm:"foreignKey:PostID"`
}

type Comment struct {
	ID       uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Content  string `json:"content" gorm:"type:text;not null"`
	ParentID *uint  `json:"parent_id" gorm:"index"`
	Level    uint   `json:"level" gorm:"default:0;index"`
	Status   string `json:"status" gorm:"type:varchar(20);default:'published';index"`
	// 关联
	UserID uint `json:"user_id" gorm:"index;not null"`
	PostID uint `json:"post_id" gorm:"index;not null"`

	// 关联关系
	User    *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Post    *Post     `json:"post,omitempty" gorm:"foreignKey:PostID"`
	Parent  *Comment  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Replies []Comment `json:"replies,omitempty" gorm:"foreignKey:ParentID"`

	// 时间
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 点赞
	LikeCount uint `json:"like_count" gorm:"default:0"`
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBanned   UserStatus = "banned"
)

type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleEditor UserRole = "editor"
	UserRoleUser   UserRole = "user"
	UserRoleGuest  UserRole = "guest"
)

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityPassword Visibility = "password"
	VisibilityFriends  Visibility = "friends"
)

type Category struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Slug      string    `json:"slug" gorm:"type:varchar(100);not null;uniqueIndex"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	Posts []Post `json:"posts,omitempty" gorm:"foreignKey:CategoryID"`
}

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"type:varchar(50);not null;uniqueIndex"`
	Slug      string    `json:"slug" gorm:"type:varchar(50);not null;uniqueIndex"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	Posts []Post `json:"posts,omitempty" gorm:"many2many:post_tags;"`
}

// 简化中间表结构体
type UserFollower struct {
	UserID      uint      `json:"user_id" gorm:"primaryKey"`
	FollowingID uint      `json:"following_id" gorm:"primaryKey"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type UserStarPost struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	PostID    uint      `json:"post_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type UserLikePost struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	PostID    uint      `json:"post_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type CommentLike struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	CommentID uint      `json:"comment_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type PostTag struct {
	PostID    uint      `json:"post_id" gorm:"primaryKey"`
	TagID     uint      `json:"tag_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) error {
	// 设置数据库引擎和字符集（MySQL特定）
	db = db.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4")
	tables := []interface{}{
		// 基础表
		&User{},
		&Category{},
		&Tag{},
		// 主表
		&Post{},
		&Comment{},
		// 关联表
		&UserFollower{},
		&UserStarPost{},
		&UserLikePost{},
		&PostTag{},
		&CommentLike{},
	}
	// 批量创建表
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return err
		}
	}

	return nil
}
