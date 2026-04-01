package handler

import (
	"auction/internal/database"
	"auction/internal/middleware"
	"auction/internal/models"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type itemCreate struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"` // draft or pending_review
}

func (d Deps) CreateItem(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var body itemCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	st := body.Status
	if st == "" {
		st = "pending_review"
	}
	if st != "draft" && st != "pending_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be draft or pending_review"})
		return
	}
	it := models.Item{
		SellerID:    uid,
		Title:       body.Title,
		Description: body.Description,
		Status:      st,
	}
	if err := d.DB.Create(&it).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, it)
}

func (d Deps) ListMyItems(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var items []models.Item
	d.DB.Where("seller_id = ?", uid).Order("id desc").Find(&items)
	c.JSON(http.StatusOK, items)
}

func (d Deps) GetItem(c *gin.Context) {
	var it models.Item
	if err := d.DB.First(&it, c.Param("id")).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	paths, _ := imagePaths(it.ImagePaths)
	c.JSON(http.StatusOK, gin.H{
		"item":       it,
		"image_urls": publicURLs(d.Cfg.StaticURLPath, paths),
	})
}

func (d Deps) SubmitItemReview(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var it models.Item
	if err := d.DB.First(&it, c.Param("id")).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if it.SellerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "not owner"})
		return
	}
	if it.Status != "draft" && it.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot submit from this status"})
		return
	}
	it.Status = "pending_review"
	it.RejectReason = ""
	if err := d.DB.Save(&it).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, it)
}

// UploadItemImages accepts multipart files; field name "images".
func (d Deps) UploadItemImages(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var it models.Item
	if err := d.DB.First(&it, c.Param("id")).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if it.SellerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "not owner"})
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart expected"})
		return
	}
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no images"})
		return
	}
	_ = os.MkdirAll(d.Cfg.UploadDir, 0o755)
	paths, _ := imagePaths(it.ImagePaths)
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
			continue
		}
		name := uuid.NewString() + ext
		dest := filepath.Join(d.Cfg.UploadDir, name)
		if err := c.SaveUploadedFile(fh, dest); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rel := name
		paths = append(paths, rel)
	}
	b, err := json.Marshal(paths)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	it.ImagePaths = string(b)
	if err := d.DB.Save(&it).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"item":       it,
		"image_urls": publicURLs(d.Cfg.StaticURLPath, paths),
	})
}

func imagePaths(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var p []string
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	return p, nil
}

func publicURLs(base string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		u := strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(p, "/")
		out = append(out, u)
	}
	return out
}
