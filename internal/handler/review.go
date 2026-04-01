package handler

import (
	"auction/internal/database"
	"auction/internal/middleware"
	"auction/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type reviewBody struct {
	AuctionID uint   `json:"auction_id" binding:"required"`
	ToUserID  uint   `json:"to_user_id" binding:"required"`
	Rating    int    `json:"rating" binding:"required,min=1,max=5"`
	Comment   string `json:"comment"`
}

func (d Deps) CreateReview(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var body reviewBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var a models.Auction
	if err := d.DB.Preload("Item").First(&a, body.AuctionID).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if a.Status != "settled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction must be settled"})
		return
	}
	if body.ToUserID == uid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot review yourself"})
		return
	}
	participant := false
	if a.WinnerUserID != nil && *a.WinnerUserID == uid {
		participant = true
	}
	if a.Item.SellerID == uid {
		participant = true
	}
	if !participant {
		c.JSON(http.StatusForbidden, gin.H{"error": "only seller or winner may review"})
		return
	}
	if a.WinnerUserID != nil && *a.WinnerUserID == uid && body.ToUserID != a.Item.SellerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "winner must review seller"})
		return
	}
	if a.Item.SellerID == uid {
		if a.WinnerUserID == nil || body.ToUserID != *a.WinnerUserID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "seller must review winner"})
			return
		}
	}
	rv := models.Review{
		AuctionID:  body.AuctionID,
		FromUserID: uid,
		ToUserID:   body.ToUserID,
		Rating:     body.Rating,
		Comment:    body.Comment,
	}
	if err := d.DB.Create(&rv).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already reviewed or invalid"})
		return
	}
	c.JSON(http.StatusCreated, rv)
}

func (d Deps) ListReviewsForUser(c *gin.Context) {
	var list []models.Review
	d.DB.Where("to_user_id = ?", c.Param("id")).Order("id desc").Limit(100).Find(&list)
	c.JSON(http.StatusOK, list)
}
