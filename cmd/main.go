package main

import (
	"BlockPoker/config"
	"BlockPoker/internal/auth"
	"BlockPoker/internal/game/manager"
	"BlockPoker/internal/matchmaker"
	"BlockPoker/internal/middleware"
	"BlockPoker/internal/storage"
	"BlockPoker/internal/utils"
	"BlockPoker/internal/websocket"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()

	//-------------------------------------------------------
	// 1. 初始化 Redis
	//-------------------------------------------------------
	if err := storage.InitRedis(
		config.C.Redis.Addr,
		config.C.Redis.Password,
		config.C.Redis.DB,
	); err != nil {
		utils.Error.Fatalf("Redis init failed: %v", err)
	}

	//-------------------------------------------------------
	// 2. 初始化 Gin + CORS
	//-------------------------------------------------------
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	//-------------------------------------------------------
	// 3. 初始化 Hub（必须最先启动）
	//-------------------------------------------------------
	hub := websocket.NewHub()
	go hub.Run()

	//-------------------------------------------------------
	// 4. 初始化 GameManager（用来启动 Engine）
	//-------------------------------------------------------
	gameMgr := manager.NewGameManager(hub)

	//-------------------------------------------------------
	// 5. 初始化匹配系统 Matchmaker
	//-------------------------------------------------------
	repo := matchmaker.NewRedisRepo(storage.Rdb)
	svc := matchmaker.NewService(repo, 300, hub)

	// 💡 成桌回调：RoomReady
	svc.OnRoomReady = func(room *matchmaker.Room) {
		utils.Info.Printf("Room ready: %s Players=%v", room.ID, room.Players)

		// 让 GameManager 接手并启动 Engine
		if err := gameMgr.StartRoom(room); err != nil {
			utils.Error.Printf("StartRoom error: %v", err)
		}
	}

	authGroup := r.Group("/auth")
	{
		auth := auth.NewHandler()
		authGroup.GET("/nonce", auth.GetNonce)
		authGroup.POST("/nonce", auth.PostNonce)
		authGroup.POST("/login", auth.Login)
	}

	//-------------------------------------------------------
	// 6. WebSocket 入口
	//-------------------------------------------------------
	// 若将来需要 JWT，在这里恢复 middleware
	//r.GET("/ws", websocket.ServeWS(hub))

	secret := ([]byte)(config.C.JWT.Secret)
	auth := r.Group("/", middleware.JwtAuthMiddleware(secret))
	{
		auth.GET("/ws", websocket.ServeWS(hub))

		mh := matchmaker.NewHandler(svc)
		//api := r.Group("/match")
		auth.POST("/match/join", mh.Join)
		auth.POST("/match/cancel", mh.Cancel)
	}

	//-------------------------------------------------------
	// 7. 匹配路由
	//-------------------------------------------------------
	// mh := matchmaker.NewHandler(svc)
	// api := r.Group("/match")
	// api.POST("/join", mh.Join)
	// api.POST("/cancel", mh.Cancel)

	//-------------------------------------------------------
	// 8. 启动服务器
	//-------------------------------------------------------
	utils.Info.Printf("Server running on %s", config.C.Server.Port)
	r.Run(config.C.Server.Port)
}
