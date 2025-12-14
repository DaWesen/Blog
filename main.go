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

	// 4. 初始化DAO
	userSQL := mysqldao.NewUserSQL(db.DB)
	commentSQL := mysqldao.NewCommentSQL(db.DB)
	postSQL := mysqldao.NewPostSQL(db.DB)
	categorySQL := mysqldao.NewCategorySQL(db.DB)
	tagSQL := mysqldao.NewTagSQL(db.DB)
	likeSQL := mysqldao.NewLikeSQL(db.DB)
	starSQL := mysqldao.NewStarSQL(db.DB)
	commentLikeSQL := mysqldao.NewCommentLikeSQL(db.DB)

	// 5. 初始化Redis Cache
	redisCache := redisdao.NewRedisCache(redisClient.Client)

	// 6. 初始化Service
	userService := UserService.NewUserService(userSQL)
	categoryService := CategoryService.NewCategoryService(categorySQL)
	commentService := CommentService.NewCommentService(
		commentSQL,
		postSQL,
		userSQL,
		commentLikeSQL,
		redisCache,
		db.DB,
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
		redisCache, // 实现了ViewCache接口
		redisCache, // 实现了LikeCache接口
		redisCache, // 实现了StarCache接口
		redisCache, // 实现了CommentCache接口
	)

	// 7. 设置路由
	router := handler.SetupRouter(
		userService,
		postService,
		categoryService,
		commentService,
		// 不再需要传入cfg参数
	)

	// 8. 启动服务器
	log.Printf("服务器启动在端口 %d", cfg.Server.Port)
	if err := router.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
