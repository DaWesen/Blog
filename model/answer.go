package model

import (
	"time"
)

// 回答模型
type Answer struct {
	ID         uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Content    string `json:"content" gorm:"type:longtext;not null"`
	QuestionID uint   `json:"question_id" gorm:"index;not null"`
	UserID     uint   `json:"user_id" gorm:"index;not null"`

	// 统计字段
	UpvoteCount  uint `json:"upvote_count" gorm:"default:0"`
	CommentCount uint `json:"comment_count" gorm:"default:0"`

	// 状态
	Status     AnswerStatus `json:"status" gorm:"type:varchar(20);default:'published';index"`
	IsAccepted bool         `json:"is_accepted" gorm:"default:false;index"`

	// 时间
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联关系
	User     *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Question *Post `json:"question,omitempty" gorm:"foreignKey:QuestionID"`
}

// 回答状态
type AnswerStatus string

const (
	AnswerStatusPublished = "published"
	AnswerStatusDraft     = "draft"
	AnswerStatusHidden    = "hidden"
)
