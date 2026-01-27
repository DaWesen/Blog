package service

import (
	mysql "blog/dao/mysql"
	"blog/model"
	"blog/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 错误定义
var (
	ErrAnswerNotFound        = errors.New("回答不存在")
	ErrInvalidAnswerContent  = errors.New("回答内容不能为空")
	ErrQuestionNotFound      = errors.New("问题不存在")
	ErrCannotAcceptOwnAnswer = errors.New("不能采纳自己的回答")
	ErrAlreadyAccepted       = errors.New("问题已有采纳的回答")
)

// AnswerService 回答服务接口
type AnswerService interface {
	// 回答基础功能
	CreateAnswer(ctx context.Context, req *CreateAnswerRequest) (*model.Answer, error)
	GetAnswer(ctx context.Context, id uint) (*model.Answer, error)
	UpdateAnswer(ctx context.Context, id uint, req *UpdateAnswerRequest) (*model.Answer, error)
	DeleteAnswer(ctx context.Context, id uint) error
	ListAnswersByQuestion(ctx context.Context, questionID uint, page, size int, order string) ([]*model.Answer, int64, error)
	ListAnswersByUser(ctx context.Context, userID uint, page, size int) ([]*model.Answer, int64, error)

	// 采纳功能
	AcceptAnswer(ctx context.Context, answerID uint) error
}

// 请求结构体
type CreateAnswerRequest struct {
	QuestionID uint   `json:"question_id" binding:"required"`
	Content    string `json:"content" binding:"required,min=1"`
}

type UpdateAnswerRequest struct {
	Content *string `json:"content,omitempty" binding:"omitempty,min=1"`
}

// answerService 回答服务实现
type answerService struct {
	answerSQL mysql.AnswerSQL
	postSQL   mysql.PostSQL
	userSQL   mysql.UserSQL
	db        *gorm.DB

	// 分布式锁管理器
	lockManager *utils.LockManager

	// 限流器
	rateLimiter *utils.RateLimiter
}

// NewAnswerService 创建回答服务
func NewAnswerService(
	answerSQL mysql.AnswerSQL,
	postSQL mysql.PostSQL,
	userSQL mysql.UserSQL,
	db *gorm.DB,
	lockManager *utils.LockManager,
	rateLimiter *utils.RateLimiter,
) AnswerService {
	return &answerService{
		answerSQL:   answerSQL,
		postSQL:     postSQL,
		userSQL:     userSQL,
		db:          db,
		lockManager: lockManager,
		rateLimiter: rateLimiter,
	}
}

// getCurrentUser 获取当前用户
func (s *answerService) getCurrentUser(ctx context.Context) (*model.User, error) {
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

// CreateAnswer 创建回答
func (s *answerService) CreateAnswer(ctx context.Context, req *CreateAnswerRequest) (*model.Answer, error) {
	// 1. 验证回答内容
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrInvalidAnswerContent
	}

	// 2. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 检查问题是否存在
	question, err := s.postSQL.GetPostByID(ctx, req.QuestionID)
	if err != nil {
		return nil, ErrQuestionNotFound
	}

	// 4. 创建回答对象
	answer := &model.Answer{
		Content:    content,
		QuestionID: req.QuestionID,
		UserID:     currentUser.ID,
		Status:     model.AnswerStatusPublished,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 5. 保存回答到数据库
	if err := s.answerSQL.InsertAnswer(ctx, answer); err != nil {
		return nil, fmt.Errorf("保存回答失败: %w", err)
	}

	// 6. 更新问题回答数
	updates := map[string]interface{}{
		"answer_count": question.AnswerCount + 1,
		"updated_at":   time.Now(),
	}
	if err := s.postSQL.UpdatePost(ctx, req.QuestionID, updates); err != nil {
		// 记录错误但不回滚
		fmt.Printf("更新问题回答数失败: %v\n", err)
	}

	// 7. 获取完整的回答信息
	fullAnswer, err := s.answerSQL.GetAnswerWithUser(ctx, answer.ID)
	if err != nil {
		return nil, fmt.Errorf("获取回答详情失败: %w", err)
	}

	return fullAnswer, nil
}

// GetAnswer 获取回答详情
func (s *answerService) GetAnswer(ctx context.Context, id uint) (*model.Answer, error) {
	answer, err := s.answerSQL.GetAnswerWithUser(ctx, id)
	if err != nil {
		return nil, ErrAnswerNotFound
	}

	return answer, nil
}

// UpdateAnswer 更新回答
func (s *answerService) UpdateAnswer(ctx context.Context, id uint, req *UpdateAnswerRequest) (*model.Answer, error) {
	// 1. 获取现有回答
	answer, err := s.answerSQL.GetAnswerByID(ctx, id)
	if err != nil {
		return nil, ErrAnswerNotFound
	}

	// 2. 检查用户权限
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	if answer.UserID != currentUser.ID {
		return nil, errors.New("没有权限修改此回答")
	}

	// 3. 构建更新数据
	updates := make(map[string]interface{})

	if req.Content != nil {
		newContent := strings.TrimSpace(*req.Content)
		if newContent != "" && newContent != answer.Content {
			updates["content"] = newContent
		}
	}

	// 如果没有更新内容，直接返回当前回答
	if len(updates) == 0 {
		return s.answerSQL.GetAnswerWithUser(ctx, id)
	}

	updates["updated_at"] = time.Now()

	// 4. 更新数据库
	if err := s.answerSQL.UpdateAnswer(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("更新回答失败: %w", err)
	}

	// 5. 获取更新后的回答
	return s.answerSQL.GetAnswerWithUser(ctx, id)
}

// DeleteAnswer 删除回答
func (s *answerService) DeleteAnswer(ctx context.Context, id uint) error {
	// 1. 获取现有回答
	answer, err := s.answerSQL.GetAnswerByID(ctx, id)
	if err != nil {
		return ErrAnswerNotFound
	}

	// 2. 检查用户权限
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	if answer.UserID != currentUser.ID {
		return errors.New("没有权限删除此回答")
	}

	// 3. 获取问题
	question, err := s.postSQL.GetPostByID(ctx, answer.QuestionID)
	if err != nil {
		return ErrQuestionNotFound
	}

	// 4. 删除回答
	if err := s.answerSQL.DeleteAnswer(ctx, id); err != nil {
		return fmt.Errorf("删除回答失败: %w", err)
	}

	// 5. 更新问题回答数
	newCount := uint(0)
	if question.AnswerCount > 0 {
		newCount = question.AnswerCount - 1
	}

	updates := map[string]interface{}{
		"answer_count": newCount,
		"updated_at":   time.Now(),
	}

	// 如果删除的是采纳的回答，清除采纳状态
	if question.BestAnswerID != nil && *question.BestAnswerID == id {
		updates["best_answer_id"] = nil
		updates["is_solved"] = false
	}

	if err := s.postSQL.UpdatePost(ctx, answer.QuestionID, updates); err != nil {
		return fmt.Errorf("更新问题回答数失败: %w", err)
	}

	return nil
}

// ListAnswersByQuestion 按问题列出回答
func (s *answerService) ListAnswersByQuestion(ctx context.Context, questionID uint, page, size int, order string) ([]*model.Answer, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 默认排序：已采纳的优先，然后按点赞数降序，再按时间
	if order == "" {
		order = "is_accepted DESC, upvote_count DESC, created_at DESC"
	}

	// 检查问题是否存在
	_, err := s.postSQL.GetPostByID(ctx, questionID)
	if err != nil {
		return nil, 0, ErrQuestionNotFound
	}

	return s.answerSQL.ListAnswersByQuestion(ctx, questionID, page, size, order)
}

// ListAnswersByUser 按用户列出回答
func (s *answerService) ListAnswersByUser(ctx context.Context, userID uint, page, size int) ([]*model.Answer, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	return s.answerSQL.ListAnswersByUser(ctx, userID, page, size)
}

// AcceptAnswer 采纳回答
func (s *answerService) AcceptAnswer(ctx context.Context, answerID uint) error {
	// 1. 获取当前用户
	currentUser, err := s.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	// 2. 获取回答
	answer, err := s.answerSQL.GetAnswerByID(ctx, answerID)
	if err != nil {
		return ErrAnswerNotFound
	}

	// 3. 获取问题
	question, err := s.postSQL.GetPostByID(ctx, answer.QuestionID)
	if err != nil {
		return ErrQuestionNotFound
	}

	// 4. 检查权限：只有问题创建者可以采纳回答
	if question.UserID != currentUser.ID {
		return errors.New("只有问题创建者可以采纳回答")
	}

	// 5. 检查是否已经是采纳的回答
	if question.BestAnswerID != nil && *question.BestAnswerID == answerID {
		return ErrAlreadyAccepted
	}

	// 6. 检查是否采纳自己的回答
	if answer.UserID == currentUser.ID {
		return ErrCannotAcceptOwnAnswer
	}

	// 7. 更新问题采纳状态
	bestAnswerID := answerID
	questionUpdates := map[string]interface{}{
		"best_answer_id": &bestAnswerID,
		"is_solved":      true,
		"updated_at":     time.Now(),
	}

	if err := s.postSQL.UpdatePost(ctx, answer.QuestionID, questionUpdates); err != nil {
		return fmt.Errorf("更新问题采纳状态失败: %w", err)
	}

	// 8. 更新回答采纳状态
	answerUpdates := map[string]interface{}{
		"is_accepted": true,
		"updated_at":  time.Now(),
	}

	if err := s.answerSQL.UpdateAnswer(ctx, answerID, answerUpdates); err != nil {
		return fmt.Errorf("更新回答采纳状态失败: %w", err)
	}

	// 9. 如果之前有采纳的回答，取消其采纳状态
	if question.BestAnswerID != nil && *question.BestAnswerID != answerID {
		oldAnswerUpdates := map[string]interface{}{
			"is_accepted": false,
			"updated_at":  time.Now(),
		}

		if err := s.answerSQL.UpdateAnswer(ctx, *question.BestAnswerID, oldAnswerUpdates); err != nil {
			// 记录错误但不中断流程
			fmt.Printf("取消旧采纳回答失败: %v\n", err)
		}
	}

	return nil
}
