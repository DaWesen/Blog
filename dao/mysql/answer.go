package dao

import (
	"blog/model"
	"context"

	"gorm.io/gorm"
)

// 回答DAO接口
type AnswerSQL interface {
	InsertAnswer(ctx context.Context, a *model.Answer) error
	GetAnswerByID(ctx context.Context, id uint) (*model.Answer, error)
	GetAnswerWithUser(ctx context.Context, id uint) (*model.Answer, error)
	UpdateAnswer(ctx context.Context, id uint, updates map[string]any) error
	DeleteAnswer(ctx context.Context, id uint) error
	FindAnswers(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Answer, error)
	CountAnswers(ctx context.Context, condition interface{}, args ...interface{}) (int64, error)
	ListAnswersByQuestion(ctx context.Context, questionID uint, page, size int, order string) ([]*model.Answer, int64, error)
	ListAnswersByUser(ctx context.Context, userID uint, page, size int) ([]*model.Answer, int64, error)
}

// 回答DAO实现
type answerSQL struct{ db *gorm.DB }

func NewAnswerSQL(db *gorm.DB) AnswerSQL { return &answerSQL{db: db} }

func (d *answerSQL) InsertAnswer(ctx context.Context, a *model.Answer) error {
	return d.db.WithContext(ctx).Create(a).Error
}

func (d *answerSQL) GetAnswerByID(ctx context.Context, id uint) (*model.Answer, error) {
	var a model.Answer
	err := d.db.WithContext(ctx).First(&a, id).Error
	return &a, err
}

func (d *answerSQL) GetAnswerWithUser(ctx context.Context, id uint) (*model.Answer, error) {
	var a model.Answer
	err := d.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url, bio")
		}).
		Preload("Question", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title, slug")
		}).
		First(&a, id).Error
	return &a, err
}

func (d *answerSQL) UpdateAnswer(ctx context.Context, id uint, updates map[string]any) error {
	return d.db.WithContext(ctx).Model(&model.Answer{}).Where("id = ?", id).Updates(updates).Error
}

func (d *answerSQL) DeleteAnswer(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.Answer{}, id).Error
}

func (d *answerSQL) FindAnswers(ctx context.Context, condition interface{}, args ...interface{}) ([]*model.Answer, error) {
	var answers []*model.Answer
	err := d.db.WithContext(ctx).Where(condition, args...).Find(&answers).Error
	return answers, err
}

func (d *answerSQL) CountAnswers(ctx context.Context, condition interface{}, args ...interface{}) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Answer{}).Where(condition, args...).Count(&count).Error
	return count, err
}

func (d *answerSQL) ListAnswersByQuestion(ctx context.Context, questionID uint, page, size int, order string) ([]*model.Answer, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	if order == "" {
		order = "created_at DESC"
	}

	offset := (page - 1) * size

	var answers []*model.Answer
	var total int64

	// 获取总数
	err := d.db.WithContext(ctx).
		Model(&model.Answer{}).
		Where("question_id = ? AND status = ?", questionID, model.AnswerStatusPublished).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取回答列表
	err = d.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Where("question_id = ? AND status = ?", questionID, model.AnswerStatusPublished).
		Order(order).
		Limit(size).
		Offset(offset).
		Find(&answers).Error
	if err != nil {
		return nil, 0, err
	}

	return answers, total, nil
}

func (d *answerSQL) ListAnswersByUser(ctx context.Context, userID uint, page, size int) ([]*model.Answer, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	var answers []*model.Answer
	var total int64

	// 获取总数
	err := d.db.WithContext(ctx).
		Model(&model.Answer{}).
		Where("user_id = ? AND status = ?", userID, model.AnswerStatusPublished).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取回答列表
	err = d.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, avatar_url")
		}).
		Preload("Question", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title, slug")
		}).
		Where("user_id = ? AND status = ?", userID, model.AnswerStatusPublished).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&answers).Error
	if err != nil {
		return nil, 0, err
	}

	return answers, total, nil
}
