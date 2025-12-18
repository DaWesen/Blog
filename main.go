package main

import (
	"blog/config"
	mysqldao "blog/dao/mysql"
	redisdao "blog/dao/redis"
	"blog/handler"
	mysqlpkg "blog/pkg/mysql"
	redispkg "blog/pkg/redis"
	CategoryService "blog/service/CategoryService"
	CommentService "blog/service/CommentService"
	PostService "blog/service/PostService"
	UserService "blog/service/UserService"
	"blog/utils"
	"log"
	"os"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	// 2. 初始化数据库
	db, err := mysqlpkg.InitMysql_or_sqlite(&cfg.Database)
	if err != nil {
		log.Fatal("初始化数据库失败:", err)
	}

	// 3. 初始化Redis
	redisClient := redispkg.NewRedisClient(&cfg.Redis)

	// 4. 初始化锁管理器和限流器
	lockManager := utils.NewLockManager(redisClient.Client)
	rateLimiter := utils.NewRateLimiter(redisClient.Client, "blog:rate_limit:")

	// 5. 初始化DAO
	userSQL := mysqldao.NewUserSQL(db.DB)
	commentSQL := mysqldao.NewCommentSQL(db.DB)
	postSQL := mysqldao.NewPostSQL(db.DB)
	categorySQL := mysqldao.NewCategorySQL(db.DB)
	tagSQL := mysqldao.NewTagSQL(db.DB)
	likeSQL := mysqldao.NewLikeSQL(db.DB)
	starSQL := mysqldao.NewStarSQL(db.DB)
	commentLikeSQL := mysqldao.NewCommentLikeSQL(db.DB)

	// 6. 初始化Redis Cache
	redisCache := redisdao.NewRedisCache(redisClient.Client)

	// 7. 初始化Service
	userService := UserService.NewUserService(userSQL, lockManager, rateLimiter)
	categoryService := CategoryService.NewCategoryService(categorySQL, lockManager, rateLimiter)

	commentService := CommentService.NewCommentService(
		commentSQL,
		postSQL,
		userSQL,
		commentLikeSQL,
		redisCache,
		db.DB,
		lockManager,
		rateLimiter,
	)

	// 创建PostService
	postService := PostService.NewPostService(
		postSQL,
		userSQL,
		categorySQL,
		tagSQL,
		likeSQL,
		starSQL,
		commentSQL,
		db.DB,
		redisCache,
		redisCache,
		redisCache,
		redisCache,
		lockManager,
		rateLimiter,
	)

	// 8. 设置路由
	router := handler.SetupRouter(
		userService,
		postService,
		categoryService,
		commentService,
		lockManager,
		rateLimiter,
	)

	// 9. 添加静态文件服务
	// 如果存在frontend文件夹，则提供静态文件服务
	router.Static("/frontend", "./frontend")

	// 添加头像上传目录的静态文件服务
	router.Static("/uploads", "./uploads")

	// 创建上传目录（如果不存在）
	createUploadDirs()

	// 10. 启动服务器
	router.Run(":8080")
}

// 创建上传目录
func createUploadDirs() {
	// 创建头像上传目录
	dirs := []string{
		"./uploads",
		"./uploads/avatars",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
			log.Printf("创建目录失败: %s, error: %v", dir, err)
		}
	}

	// 创建默认头像文件（如果不存在）
	defaultAvatarPath := "./uploads/default-avatar.png"
	if _, err := os.Stat(defaultAvatarPath); os.IsNotExist(err) {
		createDefaultAvatar(defaultAvatarPath)
	}
}

// 创建默认头像
func createDefaultAvatar(path string) {
	// 这里可以生成一个简单的默认头像
	// 为了简单起见，我们创建一个空的PNG文件占位
	file, err := os.Create(path)
	if err != nil {
		log.Printf("创建默认头像失败: %v", err)
		return
	}
	defer file.Close()

	// 可以在这里添加生成默认头像的逻辑
	// 现在只是创建一个空文件
	log.Printf("默认头像已创建: %s", path)
}
