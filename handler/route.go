package handler

import (
	"net/http"
	"time"

	answerservice "blog/service/AnswerService"
	categoryservice "blog/service/CategoryService"
	commentservice "blog/service/CommentService"
	feedservice "blog/service/FeedService"
	followservice "blog/service/FollowService"
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
	answerService answerservice.AnswerService,
	followService followservice.FollowService,
	feedService feedservice.FeedService,
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
				"关注功能": "POST /api/follow/user/:user_id",
				"动态流":  "GET /api/feed/user/me",
				"热门文章": "GET /api/feed/hot",
			},
		})
	})

	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
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
	answerHandler := NewAnswerHandler(answerService)
	followHandler := NewFollowHandler(followService)
	feedHandler := NewFeedHandler(feedService)
	searchHandler := NewSearchHandler(postService, userService)

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

			// 头像获取接口
			userGroup.GET("/users/:username/avatar", userHandler.GetAvatar)
		}

		// 文章相关路由
		postGroup := public.Group("/posts")
		{
			postGroup.GET("", postHandler.ListPosts)
			postGroup.GET("/slug/:slug", postHandler.GetPostBySlug)
			postGroup.GET("/search", postHandler.SearchPosts)
			postGroup.GET("/category/:category_id", postHandler.ListPostsByCategory)
			postGroup.GET("/tag/:tag_id", postHandler.ListPostsByTag)

			// 文章详情路由组
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
				categories, _, err := categoryService.ListCategories(c.Request.Context(), 1, 1000)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分类失败", "details": err.Error()})
					return
				}

				c.JSON(http.StatusOK, categories)
			})
		}

		// 评论相关路由
		commentGroup := public.Group("/comments")
		{
			commentGroup.GET("/:id", commentHandler.GetComment)

			// 评论详情路由组
			commentDetailGroup := commentGroup.Group("/:id")
			{
				commentDetailGroup.GET("/likes", commentHandler.GetCommentLikes)
				commentDetailGroup.GET("/replies", commentHandler.ListReplies)
			}
		}

		// 回答相关路由
		answerGroup := public.Group("/answers")
		{
			answerGroup.GET("/:id", answerHandler.GetAnswer)
			answerGroup.GET("/question/:question_id", answerHandler.ListAnswersByQuestion)
			answerGroup.GET("/user/:user_id", answerHandler.ListAnswersByUser)
		}

		// 关注相关路由
		followGroup := public.Group("/follow")
		{
			followGroup.GET("/user/:user_id/following", followHandler.GetFollowingList)
			followGroup.GET("/user/:user_id/followers", followHandler.GetFollowerList)
			followGroup.GET("/user/:user_id/stats", followHandler.GetFollowStats)
		}

		// 高级搜索路由
		searchGroup := public.Group("/search")
		{
			searchGroup.GET("/posts", searchHandler.AdvancedSearch)
			searchGroup.GET("/users", searchHandler.SearchUsers)
		}

		public.GET("/feed/hot", feedHandler.GetHotFeed)
	}
	auth := router.Group("/api")
	auth.Use(utils.JWTAuthMiddleware())
	{
		// 用户相关
		userAuthGroup := auth.Group("/user")
		{
			userAuthGroup.GET("/profile", userHandler.GetProfile)
			userAuthGroup.PUT("/profile", userHandler.UpdateProfile)
			userAuthGroup.POST("/avatar", userHandler.UploadAvatar)
			userAuthGroup.DELETE("/avatar", userHandler.DeleteAvatar)
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

		// 回答相关
		answerAuthGroup := auth.Group("/answers")
		{
			answerAuthGroup.POST("", answerHandler.CreateAnswer)

			answerDetailAuthGroup := answerAuthGroup.Group("/:id")
			{
				answerDetailAuthGroup.PUT("", answerHandler.UpdateAnswer)
				answerDetailAuthGroup.DELETE("", answerHandler.DeleteAnswer)
				answerDetailAuthGroup.POST("/accept", answerHandler.AcceptAnswer)
			}
		}

		// 关注相关
		followAuthGroup := auth.Group("/follow")
		{
			followAuthGroup.POST("/user/:user_id", followHandler.FollowUser)
			followAuthGroup.DELETE("/user/:user_id", followHandler.UnfollowUser)
			followAuthGroup.GET("/user/:user_id/is-following", followHandler.IsFollowing)
		}
		auth.GET("/feed/user/:user_id", feedHandler.GetUserFeed)
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
