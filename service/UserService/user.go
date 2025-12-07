// service/user_service.go
package service

import (
	dao "blog/dao/mysql"
	"blog/model"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// 错误定义
var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrEmailExists        = errors.New("邮箱已被使用")
	ErrUsernameExists     = errors.New("用户名已被使用")
	ErrWeakPassword       = errors.New("密码至少需要6位")
	ErrInvalidEmail       = errors.New("邮箱格式不正确")
	ErrInvalidUsername    = errors.New("用户名长度2-50个字符，不能全是空格")
)

// 请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email,max=191"`
	Password string `json:"password" binding:"required,min=6,max=255"`
	Bio      string `json:"bio,omitempty" binding:"max=500"`
}

type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email" binding:"required"`
	Password        string `json:"password" binding:"required"`
}

type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,min=2,max=50"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,url,max=500"`
	Bio       *string `json:"bio,omitempty" binding:"omitempty,max=500"`
}

// 响应结构体
type UserResponse struct {
	ID        uint             `json:"id"`
	Name      string           `json:"name"`
	Email     string           `json:"email"`
	AvatarURL string           `json:"avatar_url"`
	Bio       string           `json:"bio"`
	Status    model.UserStatus `json:"status"`
	Relation  model.UserRole   `json:"relation"`
	CreatedAt time.Time        `json:"created_at"`
}

// Service接口
type UserService interface {
	// 基础功能
	Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*UserResponse, error)

	// 资料管理
	GetUserProfile(ctx context.Context, userID uint) (*UserResponse, error)
	GetUserPublicProfile(ctx context.Context, username string) (*UserResponse, error)
	UpdateProfile(ctx context.Context, userID uint, req *UpdateProfileRequest) (*UserResponse, error)

	// 实用功能
	CheckUsernameExists(ctx context.Context, username string) (bool, error)
	CheckEmailExists(ctx context.Context, email string) (bool, error)
	GetUserByID(ctx context.Context, userID uint) (*model.User, error)
}

// 实现
type userService struct {
	userSQL dao.UserSQL
}

func NewUserService(userSQL dao.UserSQL) UserService {
	return &userService{
		userSQL: userSQL,
	}
}

// validateEmailFormat 验证邮箱格式
func validateEmailFormat(email string) error {
	if len(email) == 0 {
		return ErrInvalidEmail
	}

	if len(email) > 191 {
		return ErrInvalidEmail
	}

	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	if !re.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}

// validateUsernameFormat 验证用户名格式
func validateUsernameFormat(username string) error {
	// 去除首尾空格
	trimmed := strings.TrimSpace(username)

	// 检查长度
	if utf8.RuneCountInString(trimmed) < 2 {
		return errors.New("用户名至少需要2个字符")
	}

	if utf8.RuneCountInString(trimmed) > 50 {
		return errors.New("用户名不能超过50个字符")
	}

	// 检查是否全是空格
	if trimmed == "" {
		return ErrInvalidUsername
	}
	// 允许：汉字、字母、数字、空格、常用标点符号
	// 禁止：控制字符、特殊空白字符等
	for _, r := range trimmed {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return errors.New("用户名包含不允许的控制字符")
		}
	}

	return nil
}

// validatePassword 验证密码
func validatePassword(password string) error {
	if len(password) < 6 {
		return ErrWeakPassword
	}

	if len(password) > 255 {
		return errors.New("密码太长")
	}

	return nil
}

// normalizeEmail 标准化邮箱
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// normalizeUsername 标准化用户名
func normalizeUsername(username string) string {
	// 去除首尾空格，但保留中间的空格
	return strings.TrimSpace(username)
}

// sanitizeUsername 清理用户名（去除多余空格，但保留中间空格）
func sanitizeUsername(username string) string {
	// 去除首尾空格
	trimmed := strings.TrimSpace(username)

	// 将多个连续空格替换为单个空格
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(trimmed, " ")
}

// hashPassword 哈希密码
func hashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("密码加密失败: %w", err)
	}

	return string(hashedBytes), nil
}

// checkPassword 验证密码
func checkPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// userToResponse 转换为响应格式
func userToResponse(user *model.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
		Status:    user.Status,
		Relation:  user.Relation,
		CreatedAt: user.CreatedAt,
	}
}

// Register 用户注册
func (s *userService) Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error) {
	// 1. 验证邮箱
	if err := validateEmailFormat(req.Email); err != nil {
		return nil, err
	}

	// 2. 验证用户名
	if err := validateUsernameFormat(req.Username); err != nil {
		return nil, err
	}

	normalizedEmail := normalizeEmail(req.Email)
	sanitizedUsername := sanitizeUsername(req.Username)

	// 3. 检查邮箱是否存在
	existingUser, _ := s.userSQL.GetUserByEmail(ctx, normalizedEmail)
	if existingUser != nil {
		return nil, ErrEmailExists
	}

	// 4. 检查用户名是否存在（不区分大小写）
	existingUser, _ = s.userSQL.GetUserByName(ctx, sanitizedUsername)
	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	// 5. 哈希密码
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 6. 创建用户
	user := &model.User{
		Name:     sanitizedUsername,
		Email:    normalizedEmail,
		Password: hashedPassword,
		Bio:      req.Bio,
		Status:   model.UserStatusActive,
		Relation: model.UserRoleUser,
		LoginAt:  time.Now(),
		LoginIP:  "", // 实际使用时从请求中获取
		RealIP:   "", // 实际使用时从请求中获取
	}

	// 7. 保存到数据库
	if err := s.userSQL.InsertUser(ctx, user); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint") {
			// 再次检查是哪一项重复了
			existingUser, _ = s.userSQL.GetUserByEmail(ctx, normalizedEmail)
			if existingUser != nil {
				return nil, ErrEmailExists
			}

			existingUser, _ = s.userSQL.GetUserByName(ctx, sanitizedUsername)
			if existingUser != nil {
				return nil, ErrUsernameExists
			}
		}
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return userToResponse(user), nil
}

// Login 用户登录
func (s *userService) Login(ctx context.Context, req *LoginRequest) (*UserResponse, error) {
	var user *model.User
	var err error

	// 1. 根据用户名或邮箱查找用户
	if strings.Contains(req.UsernameOrEmail, "@") {
		// 尝试按邮箱查找
		normalizedEmail := normalizeEmail(req.UsernameOrEmail)
		user, err = s.userSQL.GetUserByEmail(ctx, normalizedEmail)
	} else {
		// 尝试按用户名查找
		// 登录时用户名可能包含空格，需要清理
		sanitizedUsername := sanitizeUsername(req.UsernameOrEmail)
		user, err = s.userSQL.GetUserByName(ctx, sanitizedUsername)
	}

	// 2. 处理用户不存在的情况
	if err != nil || user == nil {
		dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, ErrInvalidCredentials
	}

	// 3. 检查用户状态
	if user.Status == model.UserStatusBanned {
		return nil, errors.New("账号已被封禁")
	}

	if user.Status == model.UserStatusInactive {
		return nil, errors.New("账号未激活")
	}

	// 4. 验证密码
	if err := checkPassword(user.Password, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 5. 更新登录信息（可选）
	updates := map[string]interface{}{
		"login_at": time.Now(),
		"login_ip": "", // 实际使用中从请求中获取
	}

	// 登录成功，即使更新失败也不影响
	_ = s.userSQL.UpdateUser(ctx, user.ID, updates)

	return userToResponse(user), nil
}

// GetUserProfile 获取当前用户资料
func (s *userService) GetUserProfile(ctx context.Context, userID uint) (*UserResponse, error) {
	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return userToResponse(user), nil
}

// GetUserPublicProfile 获取公开用户资料（通过用户名）
func (s *userService) GetUserPublicProfile(ctx context.Context, username string) (*UserResponse, error) {
	// 清理用户名，因为数据库中存储的是清理后的格式
	sanitizedUsername := sanitizeUsername(username)

	user, err := s.userSQL.GetUserByName(ctx, sanitizedUsername)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return userToResponse(user), nil
}

// UpdateProfile 更新个人资料
func (s *userService) UpdateProfile(ctx context.Context, userID uint, req *UpdateProfileRequest) (*UserResponse, error) {
	// 1. 获取当前用户
	user, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 2. 构建更新数据
	updates := make(map[string]interface{})

	// 更新用户名（如果需要）
	if req.Name != nil && *req.Name != user.Name {
		// 验证新用户名
		if err := validateUsernameFormat(*req.Name); err != nil {
			return nil, err
		}

		sanitizedNewName := sanitizeUsername(*req.Name)

		// 检查新用户名是否已被其他人使用
		existingUser, _ := s.userSQL.GetUserByName(ctx, sanitizedNewName)
		if existingUser != nil && existingUser.ID != userID {
			return nil, ErrUsernameExists
		}

		updates["name"] = sanitizedNewName
	}

	// 更新头像
	if req.AvatarURL != nil && *req.AvatarURL != user.AvatarURL {
		updates["avatar_url"] = *req.AvatarURL
	}

	// 更新简介
	if req.Bio != nil && *req.Bio != user.Bio {
		updates["bio"] = *req.Bio
	}

	// 如果没有更新内容，直接返回当前用户
	if len(updates) == 0 {
		return userToResponse(user), nil
	}

	// 3. 更新用户信息
	if err := s.userSQL.UpdateUser(ctx, userID, updates); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "name") {
			return nil, ErrUsernameExists
		}
		return nil, fmt.Errorf("更新资料失败: %w", err)
	}

	// 4. 获取更新后的用户信息
	updatedUser, err := s.userSQL.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return userToResponse(updatedUser), nil
}

// CheckUsernameExists 检查用户名是否存在
func (s *userService) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	sanitizedUsername := sanitizeUsername(username)

	user, err := s.userSQL.GetUserByName(ctx, sanitizedUsername)
	if err != nil {
		// 如果是"record not found"错误，说明用户名不存在
		if err.Error() == "record not found" || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}

	return user != nil, nil
}

// CheckEmailExists 检查邮箱是否存在
func (s *userService) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	normalizedEmail := normalizeEmail(email)

	user, err := s.userSQL.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		// 如果是"record not found"错误，说明邮箱不存在
		if err.Error() == "record not found" || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}

	return user != nil, nil
}

// GetUserByID 通过ID获取用户（内部使用）
func (s *userService) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	return s.userSQL.GetUserByID(ctx, userID)
}
