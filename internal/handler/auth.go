package handler

import (
	"auction/internal/auth"
	"auction/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type regReq struct {
	Username    string `json:"username" binding:"required,min=2,max=64"`
	Password    string `json:"password" binding:"required,min=6,max=128"`
	DisplayName string `json:"display_name"`
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (d Deps) Register(c *gin.Context) {
	var body regReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var n int64
	d.DB.Model(&models.User{}).Where("username = ?", body.Username).Count(&n)
	if n > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "username taken"})
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	u := models.User{
		Username:     body.Username,
		PasswordHash: hash,
		DisplayName:  body.DisplayName,
	}
	if u.DisplayName == "" {
		u.DisplayName = body.Username
	}
	if err := d.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "username": u.Username})
}

func (d Deps) Login(c *gin.Context) {
	var body loginReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var u models.User
	if err := d.DB.Where("username = ?", body.Username).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !auth.CheckPassword(u.PasswordHash, body.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	tok, err := auth.IssueToken(d.Cfg.JWTSecret, u.ID, u.Username, u.IsAdmin, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      tok,
		"user_id":    u.ID,
		"username":   u.Username,
		"is_admin":   u.IsAdmin,
		"expires_in": int((24 * time.Hour).Seconds()),
	})
}
