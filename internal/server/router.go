package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhangdongyu29-blip/mchat/internal/ai"
	"github.com/zhangdongyu29-blip/mchat/internal/config"
	"github.com/zhangdongyu29-blip/mchat/internal/db"
	"github.com/zhangdongyu29-blip/mchat/internal/model"
)

//go:embed static/dist/**
var embeddedDist embed.FS

// NewRouter sets up routes and middleware.
func NewRouter(cfg config.Config) *gin.Engine {
	r := gin.Default()

	// Basic CORS for dev (Vite) and production SPA.
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		api.GET("/roles", listRoles)
		api.POST("/roles", createRole)
		api.POST("/chat", func(c *gin.Context) { chatHandler(c, cfg) })
		api.GET("/chat/:roleID", listMessages)
		api.GET("/memories/:roleID", listMemories)
	}

	// Static frontend
	sub, _ := fs.Sub(embeddedDist, "static/dist")
	r.NoRoute(serveEmbedded(sub))

	return r
}

func listRoles(c *gin.Context) {
	var roles []model.Role
	if err := db.DB.Order("id desc").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roles)
}

type createRoleReq struct {
	Name        string `json:"name" binding:"required"`
	Background  string `json:"background"`
	Style       string `json:"style"`
	PersonaHint string `json:"persona_hint"`
}

func createRole(c *gin.Context) {
	var req createRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := model.Role{
		Name:        req.Name,
		Background:  req.Background,
		Style:       req.Style,
		PersonaHint: req.PersonaHint,
	}
	if err := db.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, role)
}

type chatRequest struct {
	RoleID  uint   `json:"role_id" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

func chatHandler(c *gin.Context, cfg config.Config) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	systemPrompt := buildSystemPrompt(req.RoleID)
	var history []model.ChatMessage
	_ = db.DB.Where("role_id = ?", req.RoleID).Order("id asc").Find(&history).Error

	var messages []ai.Message
	if systemPrompt != "" {
		messages = append(messages, ai.Message{Role: "system", Content: systemPrompt})
	}
	for _, m := range history {
		role := "user"
		switch m.Sender {
		case "ai":
			role = "assistant"
		case "system":
			role = "system"
		}
		messages = append(messages, ai.Message{Role: role, Content: m.Content})
	}
	messages = append(messages, ai.Message{Role: "user", Content: req.Message})

	reply, err := ai.Chat(ctx, cfg, messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	db.DB.Create(&model.ChatMessage{RoleID: req.RoleID, Sender: "user", Content: req.Message})
	db.DB.Create(&model.ChatMessage{RoleID: req.RoleID, Sender: "ai", Content: reply})
	_ = maybeSummarize(req.RoleID)

	c.JSON(http.StatusOK, chatResponse{Reply: reply})
}

func listMessages(c *gin.Context) {
	roleID, _ := strconv.Atoi(c.Param("roleID"))
	var msgs []model.ChatMessage
	if err := db.DB.Where("role_id = ?", roleID).Order("id asc").Find(&msgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

func listMemories(c *gin.Context) {
	roleID, _ := strconv.Atoi(c.Param("roleID"))
	var memories []model.Memory
	if err := db.DB.Where("role_id = ?", roleID).Order("id desc").Find(&memories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, memories)
}

func buildSystemPrompt(roleID uint) string {
	var role model.Role
	if err := db.DB.First(&role, roleID).Error; err != nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("你是一位聊天助手，以 H5 小程序风格回应用户。\n")
	if role.Name != "" {
		sb.WriteString("角色名: " + role.Name + "\n")
	}
	if role.Background != "" {
		sb.WriteString("背景: " + role.Background + "\n")
	}
	if role.Style != "" {
		sb.WriteString("语气风格: " + role.Style + "\n")
	}
	if role.PersonaHint != "" {
		sb.WriteString("附加设定: " + role.PersonaHint + "\n")
	}

	// include latest memory
	var mem model.Memory
	if err := db.DB.Where("role_id = ?", roleID).Order("id desc").First(&mem).Error; err == nil {
		sb.WriteString("历史记忆: " + mem.Summary + "\n")
	}
	return sb.String()
}

// maybeSummarize stores a simple memory every 12 messages to keep context light.
func maybeSummarize(roleID uint) error {
	var count int64
	db.DB.Model(&model.ChatMessage{}).Where("role_id = ?", roleID).Count(&count)
	if count%12 != 0 {
		return nil
	}
	var msgs []model.ChatMessage
	db.DB.Where("role_id = ?", roleID).Order("id desc").Limit(12).Find(&msgs)
	var sb strings.Builder
	for i := len(msgs) - 1; i >= 0; i-- {
		sb.WriteString(msgs[i].Sender + ": " + msgs[i].Content + "\n")
	}
	mem := model.Memory{RoleID: roleID, Summary: sb.String()}
	return db.DB.Create(&mem).Error
}

// serveEmbedded serves the SPA with history mode support.
func serveEmbedded(sub fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			c.FileFromFS("index.html", http.FS(sub))
			return
		}
		if _, err := fs.Stat(sub, strings.TrimPrefix(path, "/")); err == nil {
			c.FileFromFS(strings.TrimPrefix(path, "/"), http.FS(sub))
			return
		}
		// fallback to index.html for SPA routes
		c.FileFromFS("index.html", http.FS(sub))
	}
}
