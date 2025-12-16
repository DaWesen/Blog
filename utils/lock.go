package utils

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ErrLockNotAcquired = errors.New("lock not acquired")
	ErrLockNotOwned    = errors.New("lock not owned by this client")
	ErrLockExpired     = errors.New("lock has expired")
)

// DistributedLock 分布式锁
type DistributedLock struct {
	client     redis.UniversalClient
	key        string
	token      string
	expiration time.Duration
	autoRenew  bool
	stopRenew  chan struct{}
	renewMutex sync.RWMutex
	isLocked   bool
}

type LockOption func(*DistributedLock)

// WithAutoRenew 设置自动续期
func WithAutoRenew(interval time.Duration) LockOption {
	return func(dl *DistributedLock) {
		dl.autoRenew = true
		// 默认续期间隔为过期时间的1/3
		if interval == 0 {
			interval = dl.expiration / 3
		}
		go dl.autoRenewLock(interval)
	}
}

// WithCustomToken 设置自定义token
func WithCustomToken(token string) LockOption {
	return func(dl *DistributedLock) {
		if token != "" {
			dl.token = token
		}
	}
}

// NewDistributedLock 创建分布式锁实例
func NewDistributedLock(client redis.UniversalClient, key string, expiration time.Duration, opts ...LockOption) *DistributedLock {
	token, _ := generateToken()

	dl := &DistributedLock{
		client:     client,
		key:        fmt.Sprintf("lock:%s", key),
		token:      token,
		expiration: expiration,
		stopRenew:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(dl)
	}

	return dl
}

// generateToken 生成随机token
func generateToken() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		// 如果随机数生成失败，使用纳秒时间戳
		return fmt.Sprintf("%d", time.Now().UnixNano()), nil
	}
	return fmt.Sprintf("%d", n), nil
}

// Acquire 获取锁
func (dl *DistributedLock) Acquire(ctx context.Context) (bool, error) {
	dl.renewMutex.Lock()
	defer dl.renewMutex.Unlock()

	if dl.isLocked {
		return true, nil
	}

	result, err := dl.client.SetNX(ctx, dl.key, dl.token, dl.expiration).Result()
	if err != nil {
		return false, fmt.Errorf("acquire lock failed: %w", err)
	}

	if result {
		dl.isLocked = true
		return true, nil
	}

	// 检查锁是否已经过期但未被清理
	ttl, err := dl.client.TTL(ctx, dl.key).Result()
	if err != nil {
		return false, fmt.Errorf("check lock ttl failed: %w", err)
	}

	// 如果锁已过期或不存在，尝试重新获取
	if ttl == -1 || ttl == -2 {
		result, err := dl.client.SetNX(ctx, dl.key, dl.token, dl.expiration).Result()
		if err != nil {
			return false, fmt.Errorf("retry acquire lock failed: %w", err)
		}
		if result {
			dl.isLocked = true
		}
		return result, nil
	}

	return false, nil
}

// AcquireWithRetry 带重试的获取锁
func (dl *DistributedLock) AcquireWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		acquired, err := dl.Acquire(ctx)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}

		if i < maxRetries-1 {
			select {
			case <-time.After(retryDelay):
				continue
			case <-ctx.Done():
				return false, ctx.Err()
			}
		}
	}

	return false, ErrLockNotAcquired
}

// Release 释放锁
func (dl *DistributedLock) Release(ctx context.Context) error {
	dl.renewMutex.Lock()
	defer dl.renewMutex.Unlock()

	if !dl.isLocked {
		return nil
	}

	// 使用GET和DEL确保只有锁的持有者才能释放
	currentToken, err := dl.client.Get(ctx, dl.key).Result()
	if err != nil {
		if err == redis.Nil {
			// 锁已经不存在
			dl.isLocked = false
			return nil
		}
		return fmt.Errorf("get lock token failed: %w", err)
	}

	if currentToken != dl.token {
		return ErrLockNotOwned
	}

	_, err = dl.client.Del(ctx, dl.key).Result()
	if err != nil {
		return fmt.Errorf("release lock failed: %w", err)
	}

	// 停止自动续期
	if dl.autoRenew {
		close(dl.stopRenew)
		dl.autoRenew = false
	}

	dl.isLocked = false
	return nil
}

// Renew 续期锁
func (dl *DistributedLock) Renew(ctx context.Context, newExpiration time.Duration) error {
	dl.renewMutex.RLock()
	defer dl.renewMutex.RUnlock()

	if !dl.isLocked {
		return ErrLockNotAcquired
	}

	currentToken, err := dl.client.Get(ctx, dl.key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrLockExpired
		}
		return fmt.Errorf("get lock token failed: %w", err)
	}

	if currentToken != dl.token {
		return ErrLockNotOwned
	}

	_, err = dl.client.Expire(ctx, dl.key, newExpiration).Result()
	if err != nil {
		return fmt.Errorf("renew lock failed: %w", err)
	}

	if newExpiration > 0 {
		dl.expiration = newExpiration
	}

	return nil
}

// IsLocked 检查锁是否被当前客户端持有
func (dl *DistributedLock) IsLocked(ctx context.Context) (bool, error) {
	dl.renewMutex.RLock()
	defer dl.renewMutex.RUnlock()

	if !dl.isLocked {
		return false, nil
	}

	currentToken, err := dl.client.Get(ctx, dl.key).Result()
	if err != nil {
		if err == redis.Nil {
			dl.isLocked = false
			return false, nil
		}
		return false, fmt.Errorf("check lock failed: %w", err)
	}

	if currentToken != dl.token {
		dl.isLocked = false
		return false, nil
	}

	return true, nil
}

// GetTTL 获取锁剩余时间
func (dl *DistributedLock) GetTTL(ctx context.Context) (time.Duration, error) {
	dl.renewMutex.RLock()
	defer dl.renewMutex.RUnlock()

	if !dl.isLocked {
		return 0, ErrLockNotAcquired
	}

	ttl, err := dl.client.TTL(ctx, dl.key).Result()
	if err != nil {
		return 0, fmt.Errorf("get lock ttl failed: %w", err)
	}

	return ttl, nil
}

// autoRenewLock 自动续期锁
func (dl *DistributedLock) autoRenewLock(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := dl.Renew(ctx, dl.expiration)
			cancel()

			if err != nil {
				return
			}
		case <-dl.stopRenew:
			return
		}
	}
}

// Mutex 互斥执行函数
func (dl *DistributedLock) Mutex(ctx context.Context, fn func() error, opts ...LockOption) error {
	// 创建新锁实例避免状态污染
	lock := NewDistributedLock(dl.client, dl.key[len("lock:"):], dl.expiration, opts...)

	acquired, err := lock.AcquireWithRetry(ctx, 3, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("acquire lock failed: %w", err)
	}

	if !acquired {
		return ErrLockNotAcquired
	}

	defer func() {
		// 释放锁，忽略错误
		_ = lock.Release(ctx)
	}()

	return fn()
}

// LockManager 锁管理器，用于管理多个锁
type LockManager struct {
	client redis.UniversalClient
	locks  sync.Map
}

func NewLockManager(client redis.UniversalClient) *LockManager {
	return &LockManager{
		client: client,
	}
}

// GetLock 获取或创建锁
func (lm *LockManager) GetLock(key string, expiration time.Duration, opts ...LockOption) *DistributedLock {
	lockKey := fmt.Sprintf("lock:%s", key)

	if lock, ok := lm.locks.Load(lockKey); ok {
		return lock.(*DistributedLock)
	}

	lock := NewDistributedLock(lm.client, key, expiration, opts...)
	lm.locks.Store(lockKey, lock)

	// 设置过期删除
	go func() {
		time.Sleep(expiration + 10*time.Second)
		lm.locks.Delete(lockKey)
	}()

	return lock
}

// ReleaseAll 释放所有锁
func (lm *LockManager) ReleaseAll(ctx context.Context) error {
	var errors []error

	lm.locks.Range(func(key, value interface{}) bool {
		lock := value.(*DistributedLock)
		if err := lock.Release(ctx); err != nil {
			errors = append(errors, fmt.Errorf("release lock %s failed: %w", key, err))
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("release all locks failed: %v", errors)
	}

	return nil
}
