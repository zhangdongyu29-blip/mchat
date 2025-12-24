package server

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
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
	"gorm.io/gorm"
)

//go:embed static/dist/**
var embeddedDist embed.FS

// NewRouter sets up routes and middleware.
func NewRouter(cfg config.Config) *gin.Engine {
	r := gin.Default()

	// Basic CORS for dev (Vite) and production SPA.
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
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
		api.GET("/roles/:roleID", getRole)
		api.PUT("/roles/:roleID", updateRole)
		api.DELETE("/roles/:roleID", deleteRole)

		api.GET("/roles/:roleID/conversations", listConversations)
		api.POST("/roles/:roleID/conversations", createConversation)
		api.DELETE("/conversations/:id", deleteConversation)

		api.POST("/chat", func(c *gin.Context) { chatHandler(c, cfg) })
		api.GET("/conversations/:id/messages", listMessages)
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
	CallMe      string `json:"call_me"`
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
		CallMe:      req.CallMe,
	}
	if err := db.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, role)
}

type chatRequest struct {
	RoleID         uint   `json:"role_id" binding:"required"`
	ConversationID uint   `json:"conversation_id"`
	Message        string `json:"message" binding:"required"`
}

type conversationRequest struct {
	Title string `json:"title"`
}

type updateRoleReq struct {
	Name        string `json:"name" binding:"required"`
	Background  string `json:"background"`
	Style       string `json:"style"`
	PersonaHint string `json:"persona_hint"`
	CallMe      string `json:"call_me"`
}

func getRole(c *gin.Context) {
	id := c.Param("roleID")
	var role model.Role
	if err := db.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	c.JSON(http.StatusOK, role)
}

func updateRole(c *gin.Context) {
	id := c.Param("roleID")
	var role model.Role
	if err := db.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	var req updateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role.Name = req.Name
	role.Background = req.Background
	role.Style = req.Style
	role.PersonaHint = req.PersonaHint
	role.CallMe = req.CallMe
	if err := db.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, role)
}

func deleteRole(c *gin.Context) {
	id := c.Param("roleID")
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&model.ChatMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&model.Memory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&model.Conversation{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Role{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func listConversations(c *gin.Context) {
	roleID := c.Param("roleID")
	var items []model.Conversation
	if err := db.DB.Where("role_id = ?", roleID).Order("updated_at desc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func createConversation(c *gin.Context) {
	roleID, _ := strconv.Atoi(c.Param("roleID"))
	var req conversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	convo := model.Conversation{
		RoleID: uint(roleID),
		Title:  req.Title,
	}
	if err := db.DB.Create(&convo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, convo)
}

func deleteConversation(c *gin.Context) {
	id := c.Param("id")
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", id).Delete(&model.ChatMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Conversation{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
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
	// ensure conversation
	var convo model.Conversation
	if req.ConversationID != 0 {
		_ = db.DB.First(&convo, req.ConversationID).Error
	}
	if convo.ID == 0 {
		convo = model.Conversation{RoleID: req.RoleID, Title: ""}
		db.DB.Create(&convo)
	}

	var history []model.ChatMessage
	_ = db.DB.Where("conversation_id = ?", convo.ID).Order("id asc").Find(&history).Error

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

	db.DB.Create(&model.ChatMessage{RoleID: req.RoleID, ConversationID: convo.ID, Sender: "user", Content: req.Message})

	resp, err := ai.ChatStream(ctx, cfg, messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Conversation-ID", strconv.Itoa(int(convo.ID)))
	c.Status(http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	var reply strings.Builder
	flusher, _ := c.Writer.(http.Flusher)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk ai.StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		token := chunk.Choices[0].Delta.Content
		if token == "" {
			continue
		}
		reply.WriteString(token)
		_, _ = c.Writer.Write([]byte(token))
		if flusher != nil {
			flusher.Flush()
		}
	}

	finalReply := reply.String()
	db.DB.Create(&model.ChatMessage{RoleID: req.RoleID, ConversationID: convo.ID, Sender: "ai", Content: finalReply})
	if convo.Title == "" {
		title := generateTitle(cfg, systemPrompt, req.Message, finalReply)
		db.DB.Model(&convo).Update("title", title)
	}
	db.DB.Model(&convo).Update("updated_at", time.Now())
	_ = maybeSummarize(req.RoleID)
}

func listMessages(c *gin.Context) {
	convID, _ := strconv.Atoi(c.Param("id"))
	var msgs []model.ChatMessage
	if err := db.DB.Where("conversation_id = ?", convID).Order("id asc").Find(&msgs).Error; err != nil {
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

// generateTitle produces a short conversation title from first user/assistant turn.
func generateTitle(cfg config.Config, systemPrompt, userMsg, aiMsg string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var msgs []ai.Message
	if systemPrompt != "" {
		msgs = append(msgs, ai.Message{Role: "system", Content: systemPrompt})
	}
	msgs = append(msgs,
		ai.Message{Role: "user", Content: userMsg},
		ai.Message{Role: "assistant", Content: aiMsg},
		ai.Message{Role: "system", Content: "请为本次对话生成一个不超过12字的中文摘要标题，简洁且避免标点。仅输出标题文本。"},
	)
	title, err := ai.Chat(ctx, cfg, msgs)
	if err != nil {
		return "新的对话"
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "新的对话"
	}
	if len([]rune(title)) > 20 {
		return string([]rune(title)[:20])
	}
	return title
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
	if role.CallMe != "" {
		sb.WriteString("请直接称呼用户为: " + role.CallMe + "\n")
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
