package service

import (
	dao "blog/dao/mysql"
	"blog/model"
	"blog/utils"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
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
	ErrRateLimited        = errors.New("操作过于频繁，请稍后再试")
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

	// 分布式锁管理器
	lockManager *utils.LockManager

	// 限流器
	rateLimiter *utils.RateLimiter

	// 用户信息缓存
	userCache     map[uint]*model.User
	userCacheTTL  map[uint]time.Time
	userCacheLock sync.RWMutex
	readCacheLock sync.RWMutex
	// 用户名->用户ID映射（用于快速查找）
	usernameToID map[string]uint
	usernameLock sync.RWMutex
}

func NewUserService(userSQL dao.UserSQL, lockManager *utils.LockManager, rateLimiter *utils.RateLimiter) UserService {
	return &userService{
		userSQL:      userSQL,
		lockManager:  lockManager,
		rateLimiter:  rateLimiter,
		userCache:    make(map[uint]*model.User),
		userCacheTTL: make(map[uint]time.Time),
		usernameToID: make(map[string]uint),
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
	return strings.TrimSpace(username)
}

// sanitizeUsername 清理用户名
func sanitizeUsername(username string) string {
	trimmed := strings.TrimSpace(username)
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

// getCachedUser 获取缓存的用户信息
func (s *userService) getCachedUser(ctx context.Context, userID uint) (*model.User, bool) {
	s.userCacheLock.RLock()
	defer s.userCacheLock.RUnlock()

	if user, ok := s.userCache[userID]; ok {
		if s.userCacheTTL[userID].After(time.Now()) {
			return user, true
		}
	}
	return nil, false
}

// cacheUser 缓存用户信息
func (s *userService) cacheUser(user *model.User) {
	s.userCacheLock.Lock()
	defer s.userCacheLock.Unlock()

	s.userCache[user.ID] = user
	s.userCacheTTL[user.ID] = time.Now().Add(10 * time.Minute) // 缓存10分钟

	// 更新用户名映射
	s.usernameLock.Lock()
	s.usernameToID[user.Name] = user.ID
	s.usernameLock.Unlock()
}

// Register 用户注册（带分布式锁和限流）
func (s *userService) Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error) {
	// 1. IP级别限流
	ip := utils.GetIPFromContext(ctx)
	ipRateLimitKey := fmt.Sprintf("register:ip:%s", ip)
	ipRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Hour,
		MaxRequests: 10, // 每小时最多10次注册
	}

	if err := s.rateLimiter.Allow(ctx, ipRateLimitKey, ipRateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 2. 验证邮箱
	if err := validateEmailFormat(req.Email); err != nil {
		return nil, err
	}

	// 3. 验证用户名
	if err := validateUsernameFormat(req.Username); err != nil {
		return nil, err
	}

	normalizedEmail := normalizeEmail(req.Email)
	sanitizedUsername := sanitizeUsername(req.Username)

	// 4. 使用分布式锁检查邮箱和用户名
	emailLockKey := fmt.Sprintf("register_email:%s", normalizedEmail)
	usernameLockKey := fmt.Sprintf("register_username:%s", sanitizedUsername)

	// 同时获取两个锁，避免死锁
	err := s.lockManager.GetLock(emailLockKey, 5*time.Second).Mutex(ctx, func() error {
		return s.lockManager.GetLock(usernameLockKey, 5*time.Second).Mutex(ctx, func() error {
			// 检查邮箱是否存在
			existingUser, _ := s.userSQL.GetUserByEmail(ctx, normalizedEmail)
			if existingUser != nil {
				return ErrEmailExists
			}

			// 检查用户名是否存在
			existingUser, _ = s.userSQL.GetUserByName(ctx, sanitizedUsername)
			if existingUser != nil {
				return ErrUsernameExists
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
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
	}

	// 7. 保存到数据库（使用分布式锁保护）
	registerLockKey := fmt.Sprintf("user_register:%s", sanitizedUsername)
	err = s.lockManager.GetLock(registerLockKey, 10*time.Second).Mutex(ctx, func() error {
		if err := s.userSQL.InsertUser(ctx, user); err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint") {
				// 再次检查是哪一项重复了
				existingUser, _ := s.userSQL.GetUserByEmail(ctx, normalizedEmail)
				if existingUser != nil {
					return ErrEmailExists
				}

				existingUser, _ = s.userSQL.GetUserByName(ctx, sanitizedUsername)
				if existingUser != nil {
					return ErrUsernameExists
				}
			}
			return fmt.Errorf("创建用户失败: %w", err)
		}

		// 缓存新用户
		s.cacheUser(user)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return userToResponse(user), nil
}

// Login 用户登录（带分布式锁和限流）
func (s *userService) Login(ctx context.Context, req *LoginRequest) (*UserResponse, error) {
	// 1. IP级别限流（防止暴力破解）
	ip := utils.GetIPFromContext(ctx)
	ipRateLimitKey := fmt.Sprintf("login:ip:%s", ip)
	ipRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 30, // 每分钟最多30次登录尝试
	}

	if err := s.rateLimiter.Allow(ctx, ipRateLimitKey, ipRateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	var user *model.User
	var err error

	// 2. 根据用户名或邮箱查找用户
	if strings.Contains(req.UsernameOrEmail, "@") {
		// 尝试按邮箱查找
		normalizedEmail := normalizeEmail(req.UsernameOrEmail)

		// 使用分布式锁保护登录过程
		emailLockKey := fmt.Sprintf("login_email:%s", normalizedEmail)
		err = s.lockManager.GetLock(emailLockKey, 5*time.Second).Mutex(ctx, func() error {
			user, err = s.userSQL.GetUserByEmail(ctx, normalizedEmail)
			return err
		})
	} else {
		// 尝试按用户名查找
		sanitizedUsername := sanitizeUsername(req.UsernameOrEmail)

		// 先尝试从缓存获取
		s.usernameLock.RLock()
		if userID, ok := s.usernameToID[sanitizedUsername]; ok {
			s.usernameLock.RUnlock()
			if cachedUser, ok := s.getCachedUser(ctx, userID); ok {
				user = cachedUser
			} else {
				// 使用分布式锁保护登录过程
				usernameLockKey := fmt.Sprintf("login_username:%s", sanitizedUsername)
				err = s.lockManager.GetLock(usernameLockKey, 5*time.Second).Mutex(ctx, func() error {
					user, err = s.userSQL.GetUserByName(ctx, sanitizedUsername)
					return err
				})
			}
		} else {
			s.usernameLock.RUnlock()
			// 使用分布式锁保护登录过程
			usernameLockKey := fmt.Sprintf("login_username:%s", sanitizedUsername)
			err = s.lockManager.GetLock(usernameLockKey, 5*time.Second).Mutex(ctx, func() error {
				user, err = s.userSQL.GetUserByName(ctx, sanitizedUsername)
				return err
			})
		}
	}

	// 3. 处理用户不存在的情况
	if err != nil || user == nil {
		// 使用虚拟哈希防止时序攻击
		dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, ErrInvalidCredentials
	}

	// 4. 用户级登录限流
	userRateLimitKey := fmt.Sprintf("login_user:%d", user.ID)
	userRateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 10, // 每分钟最多10次登录尝试
	}

	if err := s.rateLimiter.Allow(ctx, userRateLimitKey, userRateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 5. 检查用户状态
	if user.Status == model.UserStatusBanned {
		return nil, errors.New("账号已被封禁")
	}

	if user.Status == model.UserStatusInactive {
		return nil, errors.New("账号未激活")
	}

	// 6. 验证密码（使用分布式锁保护）
	passwordLockKey := fmt.Sprintf("password_check:%d", user.ID)
	passwordErr := s.lockManager.GetLock(passwordLockKey, 3*time.Second).Mutex(ctx, func() error {
		return checkPassword(user.Password, req.Password)
	})

	if passwordErr != nil {
		return nil, ErrInvalidCredentials
	}

	// 7. 更新登录信息（使用分布式锁保护）
	updateLockKey := fmt.Sprintf("user_update:%d", user.ID)
	_ = s.lockManager.GetLock(updateLockKey, 5*time.Second).Mutex(ctx, func() error {
		updates := map[string]interface{}{
			"login_at": time.Now(),
			"login_ip": utils.GetIPFromContext(ctx),
		}

		// 更新用户信息
		if err := s.userSQL.UpdateUser(ctx, user.ID, updates); err != nil {
			return err
		}

		// 更新缓存中的用户信息
		user.LoginAt = time.Now()
		user.LoginIP = utils.GetIPFromContext(ctx)
		s.cacheUser(user)

		return nil
	})

	return userToResponse(user), nil
}

// GetUserProfile 获取当前用户资料（带缓存）
func (s *userService) GetUserProfile(ctx context.Context, userID uint) (*UserResponse, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_profile:user:%d:ip:%s", userID, ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 300,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 首先尝试从缓存获取
	s.readCacheLock.RLock()
	if cachedUser, ok := s.getCachedUser(ctx, userID); ok {
		s.readCacheLock.RUnlock()
		return userToResponse(cachedUser), nil
	}
	s.readCacheLock.RUnlock()

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("user_profile:%d", userID)
	var user *model.User

	err := s.lockManager.GetLock(lockKey, 5*time.Second).Mutex(ctx, func() error {
		// 再次检查缓存
		if cachedUser, ok := s.getCachedUser(ctx, userID); ok {
			user = cachedUser
			return nil
		}

		// 从数据库获取
		var err error
		user, err = s.userSQL.GetUserByID(ctx, userID)
		if err != nil {
			return ErrUserNotFound
		}

		// 缓存用户信息
		s.cacheUser(user)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return userToResponse(user), nil
}

// GetUserPublicProfile 获取公开用户资料（带缓存）
func (s *userService) GetUserPublicProfile(ctx context.Context, username string) (*UserResponse, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("get_public_profile:username:%s:ip:%s", username, ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 500,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return nil, ErrRateLimited
	}

	// 清理用户名
	sanitizedUsername := sanitizeUsername(username)

	// 首先尝试从用户名映射获取用户ID
	s.usernameLock.RLock()
	userID, ok := s.usernameToID[sanitizedUsername]
	s.usernameLock.RUnlock()

	if ok {
		// 从缓存获取用户信息
		s.userCacheLock.RLock()
		if user, ok := s.userCache[userID]; ok {
			if s.userCacheTTL[userID].After(time.Now()) {
				s.userCacheLock.RUnlock()
				return userToResponse(user), nil
			}
		}
		s.userCacheLock.RUnlock()
	}

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("user_public_profile:%s", sanitizedUsername)
	var user *model.User

	err := s.lockManager.GetLock(lockKey, 5*time.Second).Mutex(ctx, func() error {
		// 从数据库获取
		var err error
		user, err = s.userSQL.GetUserByName(ctx, sanitizedUsername)
		if err != nil {
			return ErrUserNotFound
		}

		// 缓存用户信息
		s.cacheUser(user)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return userToResponse(user), nil
}

// UpdateProfile 更新个人资料（带分布式锁）
func (s *userService) UpdateProfile(ctx context.Context, userID uint, req *UpdateProfileRequest) (*UserResponse, error) {
	// 1. 获取当前用户
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 2. 构建更新数据
	updates := make(map[string]interface{})

	// 用户名更新需要特殊处理
	if req.Name != nil && *req.Name != user.Name {
		// 验证新用户名
		if err := validateUsernameFormat(*req.Name); err != nil {
			return nil, err
		}

		sanitizedNewName := sanitizeUsername(*req.Name)

		// 使用分布式锁检查新用户名是否已被其他人使用
		usernameLockKey := fmt.Sprintf("username_check:%s", sanitizedNewName)
		err = s.lockManager.GetLock(usernameLockKey, 5*time.Second).Mutex(ctx, func() error {
			existingUser, _ := s.userSQL.GetUserByName(ctx, sanitizedNewName)
			if existingUser != nil && existingUser.ID != userID {
				return ErrUsernameExists
			}

			updates["name"] = sanitizedNewName
			return nil
		})

		if err != nil {
			return nil, err
		}
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

	// 3. 使用分布式锁更新用户信息
	updateLockKey := fmt.Sprintf("user_update_profile:%d", userID)
	err = s.lockManager.GetLock(updateLockKey, 10*time.Second).Mutex(ctx, func() error {
		updates["updated_at"] = time.Now()

		if err := s.userSQL.UpdateUser(ctx, userID, updates); err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "name") {
				return ErrUsernameExists
			}
			return fmt.Errorf("更新资料失败: %w", err)
		}

		// 清除缓存
		s.userCacheLock.Lock()
		delete(s.userCache, userID)
		delete(s.userCacheTTL, userID)
		s.userCacheLock.Unlock()

		s.usernameLock.Lock()
		delete(s.usernameToID, user.Name)
		s.usernameLock.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 4. 获取更新后的用户信息
	updatedUser, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return userToResponse(updatedUser), nil
}

// CheckUsernameExists 检查用户名是否存在（带缓存和限流）
func (s *userService) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("check_username:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 200,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return false, ErrRateLimited
	}

	sanitizedUsername := sanitizeUsername(username)

	// 首先检查缓存
	s.usernameLock.RLock()
	if _, ok := s.usernameToID[sanitizedUsername]; ok {
		s.usernameLock.RUnlock()
		return true, nil
	}
	s.usernameLock.RUnlock()

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("check_username_db:%s", sanitizedUsername)
	var exists bool

	err := s.lockManager.GetLock(lockKey, 3*time.Second).Mutex(ctx, func() error {
		user, err := s.userSQL.GetUserByName(ctx, sanitizedUsername)
		if err != nil {
			// 如果是"record not found"错误，说明用户名不存在
			if err.Error() == "record not found" || strings.Contains(err.Error(), "not found") {
				exists = false
				return nil
			}
			return err
		}

		exists = user != nil

		// 如果存在，更新缓存
		if exists {
			s.usernameLock.Lock()
			s.usernameToID[sanitizedUsername] = user.ID
			s.usernameLock.Unlock()
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	return exists, nil
}

// CheckEmailExists 检查邮箱是否存在（带缓存和限流）
func (s *userService) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	// 限流检查
	ip := utils.GetIPFromContext(ctx)
	rateLimitKey := fmt.Sprintf("check_email:ip:%s", ip)
	rateLimitConfig := utils.LimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 200,
	}

	if err := s.rateLimiter.Allow(ctx, rateLimitKey, rateLimitConfig); err != nil {
		return false, ErrRateLimited
	}

	normalizedEmail := normalizeEmail(email)

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("check_email_db:%s", normalizedEmail)
	var exists bool

	err := s.lockManager.GetLock(lockKey, 3*time.Second).Mutex(ctx, func() error {
		user, err := s.userSQL.GetUserByEmail(ctx, normalizedEmail)
		if err != nil {
			// 如果是"record not found"错误，说明邮箱不存在
			if err.Error() == "record not found" || strings.Contains(err.Error(), "not found") {
				exists = false
				return nil
			}
			return err
		}

		exists = user != nil
		return nil
	})

	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetUserByID 通过ID获取用户（带缓存）
func (s *userService) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	// 首先尝试从缓存获取
	s.readCacheLock.RLock()
	if cachedUser, ok := s.getCachedUser(ctx, userID); ok {
		s.readCacheLock.RUnlock()
		return cachedUser, nil
	}
	s.readCacheLock.RUnlock()

	// 使用分布式锁保护数据库查询
	lockKey := fmt.Sprintf("user_by_id:%d", userID)
	var user *model.User

	err := s.lockManager.GetLock(lockKey, 5*time.Second).Mutex(ctx, func() error {
		// 再次检查缓存
		if cachedUser, ok := s.getCachedUser(ctx, userID); ok {
			user = cachedUser
			return nil
		}

		// 从数据库获取
		var err error
		user, err = s.userSQL.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}

		// 缓存用户信息
		s.cacheUser(user)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}
