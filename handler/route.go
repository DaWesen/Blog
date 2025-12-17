package handler

import (
	"net/http"
	"time"

	categoryservice "blog/service/CategoryService"
	commentservice "blog/service/CommentService"
	postservice "blog/service/PostService"
	userservice "blog/service/UserService"
	"blog/utils"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(
	userService userservice.UserService,
	postService postservice.PostService,
	categoryService categoryservice.CategoryService,
	commentService commentservice.CommentService,
	lockManager *utils.LockManager,
	rateLimiter *utils.RateLimiter,
) *gin.Engine {
	router := gin.Default()

	// 中间件
	router.Use(CORSMiddleware())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 添加根路径和favicon处理
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "欢迎使用博客API服务",
			"version": "1.0.0",
			"api": gin.H{
				"文档":   "查看 /api/health 和 /api/version 获取服务信息",
				"注册":   "POST /api/register",
				"登录":   "POST /api/login",
				"文章列表": "GET /api/posts",
			},
		})
	})

	router.GET("/favicon.ico", func(c *gin.Context) {
		// 可以返回一个空响应或者自定义图标
		c.Status(204) // No Content
	})

	// 健康检查接口
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API版本信息
	router.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "1.0.0",
			"name":    "博客系统API",
		})
	})

	// 初始化Handler
	userHandler := NewUserHandler(userService)
	postHandler := NewPostHandler(postService)
	categoryHandler := NewCategoryHandler(categoryService)
	commentHandler := NewCommentHandler(commentService)

	// 公共路由（无需认证）
	public := router.Group("/api")
	{
		// 用户相关路由
		userGroup := public.Group("/")
		{
			userGroup.POST("/register", userHandler.Register)
			userGroup.POST("/login", userHandler.Login)
			userGroup.GET("/check-username", userHandler.CheckUsernameExists)
			userGroup.GET("/check-email", userHandler.CheckEmailExists)
			userGroup.GET("/users/:username", userHandler.GetUserPublicProfile)

			// 添加统计接口
			userGroup.GET("/stats/users/count", func(c *gin.Context) {
				// 这里可以调用统计服务，暂时返回一个固定值
				c.JSON(200, gin.H{
					"count":   0,
					"message": "用户统计功能待实现",
				})
			})
		}

		// 文章相关路由
		postGroup := public.Group("/posts")
		{
			postGroup.GET("", postHandler.ListPosts)
			postGroup.GET("/slug/:slug", postHandler.GetPostBySlug)
			postGroup.GET("/search", postHandler.SearchPosts)
			postGroup.GET("/category/:category_id", postHandler.ListPostsByCategory)
			postGroup.GET("/tag/:tag_id", postHandler.ListPostsByTag)

			// 文章详情路由组 - 使用子路由
			postDetailGroup := postGroup.Group("/:id")
			{
				postDetailGroup.GET("", postHandler.GetPost)
				postDetailGroup.GET("/stats", postHandler.GetPostStats)
				postDetailGroup.GET("/comments", commentHandler.ListCommentsByPost)
			}
		}

		// 分类相关路由
		categoryGroup := public.Group("/categories")
		{
			categoryGroup.GET("", categoryHandler.ListCategories)
			categoryGroup.GET("/slug/:slug", categoryHandler.GetCategoryBySlug)
			categoryGroup.GET("/search", categoryHandler.SearchCategories)
			categoryGroup.GET("/:id", categoryHandler.GetCategory)

			// 添加纯数组格式的接口
			categoryGroup.GET("/all", func(c *gin.Context) {
				// 直接调用service获取所有分类
				categories, _, err := categoryService.ListCategories(c.Request.Context(), 1, 1000)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分类失败", "details": err.Error()})
					return
				}

				// 直接返回分类数组，不包含分页信息
				c.JSON(http.StatusOK, categories)
			})
		}

		// 评论相关路由
		commentGroup := public.Group("/comments")
		{
			commentGroup.GET("/:id", commentHandler.GetComment)

			// 评论详情路由组 - 使用子路由
			commentDetailGroup := commentGroup.Group("/:id")
			{
				commentDetailGroup.GET("/likes", commentHandler.GetCommentLikes)
				commentDetailGroup.GET("/replies", commentHandler.ListReplies)
			}
		}
	}

	// 需要认证的路由
	auth := router.Group("/api")
	auth.Use(utils.JWTAuthMiddleware())
	{
		// 用户相关
		userAuthGroup := auth.Group("/user")
		{
			userAuthGroup.GET("/profile", userHandler.GetProfile)
			userAuthGroup.PUT("/profile", userHandler.UpdateProfile)
		}

		// 文章相关
		postAuthGroup := auth.Group("/posts")
		{
			postAuthGroup.POST("", postHandler.CreatePost)

			postDetailAuthGroup := postAuthGroup.Group("/:id")
			{
				postDetailAuthGroup.PUT("", postHandler.UpdatePost)
				postDetailAuthGroup.DELETE("", postHandler.DeletePost)
				postDetailAuthGroup.POST("/like", postHandler.LikePost)
				postDetailAuthGroup.DELETE("/unlike", postHandler.UnlikePost)
				postDetailAuthGroup.POST("/star", postHandler.StarPost)
				postDetailAuthGroup.DELETE("/unstar", postHandler.UnstarPost)
			}
		}

		// 分类相关
		categoryAuthGroup := auth.Group("/categories")
		{
			categoryAuthGroup.POST("", categoryHandler.CreateCategory)

			categoryDetailAuthGroup := categoryAuthGroup.Group("/:id")
			{
				categoryDetailAuthGroup.PUT("", categoryHandler.UpdateCategory)
				categoryDetailAuthGroup.DELETE("", categoryHandler.DeleteCategory)
			}
		}

		// 评论相关
		commentAuthGroup := auth.Group("/comments")
		{
			commentAuthGroup.POST("", commentHandler.CreateComment)
			commentAuthGroup.POST("/reply", commentHandler.CreateReply)
			commentAuthGroup.GET("/user/:user_id", commentHandler.ListCommentsByUser)

			commentDetailAuthGroup := commentAuthGroup.Group("/:id")
			{
				commentDetailAuthGroup.DELETE("", commentHandler.DeleteComment)
				commentDetailAuthGroup.POST("/like", commentHandler.LikeComment)
				commentDetailAuthGroup.DELETE("/unlike", commentHandler.UnlikeComment)
				commentDetailAuthGroup.GET("/is-liked", commentHandler.IsCommentLiked)
			}
		}
	}

	return router
}

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
