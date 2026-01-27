package main

import (
	"blog/config"
	mysqldao "blog/dao/mysql"
	redisdao "blog/dao/redis"
	"blog/handler"
	mysqlpkg "blog/pkg/mysql"
	redispkg "blog/pkg/redis"
	answerservice "blog/service/AnswerService"
	CategoryService "blog/service/CategoryService"
	CommentService "blog/service/CommentService"
	feedservice "blog/service/FeedService"
	followservice "blog/service/FollowService"
	PostService "blog/service/PostService"
	UserService "blog/service/UserService"
	"blog/utils"
	"fmt"
	"log"
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
	answerSQL := mysqldao.NewAnswerSQL(db.DB)
	commentLikeSQL := mysqldao.NewCommentLikeSQL(db.DB)
	followSQL := mysqldao.NewFollowSQL(db.DB)

	// 6. 初始化Redis Cache
	redisCache := redisdao.NewRedisCache(redisClient.Client)

	// 7. 初始化Service
	userService := UserService.NewUserService(userSQL, lockManager, rateLimiter)
	categoryService := CategoryService.NewCategoryService(categorySQL, lockManager, rateLimiter)

	// 创建AnswerService
	answerService := answerservice.NewAnswerService(
		answerSQL,
		postSQL,
		userSQL,
		db.DB,
		lockManager,
		rateLimiter,
	)

	// 创建FollowService
	followService := followservice.NewFollowService(
		followSQL,
		userSQL,
		db.DB,
	)

	// 创建FeedService
	feedService := feedservice.NewFeedService(
		db.DB,
		followSQL,
		postSQL,
	)

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
		answerService,
		followService,
		feedService,
		lockManager,
		rateLimiter,
	)

	// 添加静态文件服务
	router.Static("/uploads", "./uploads")
	router.Static("/frontend", "./frontend")

	// 10. 启动服务器
	log.Printf("服务器启动在端口 %d", cfg.Server.Port)
	log.Printf("前端地址: http://localhost:%d/frontend", cfg.Server.Port)
	if err := router.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
