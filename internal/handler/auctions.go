package handler

import (
	"auction/internal/database"
	"auction/internal/middleware"
	"auction/internal/models"
	"auction/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type auctionCreate struct {
	ItemID             uint   `json:"item_id" binding:"required"`
	RulesText          string `json:"rules_text"`
	StartAt            string `json:"start_at" binding:"required"` // RFC3339
	EndAt              string `json:"end_at" binding:"required"`
	StartingPriceCents int64  `json:"starting_price_cents" binding:"required"`
	MinIncrementCents  int64  `json:"min_increment_cents" binding:"required"`
	ExtendSeconds      int    `json:"extend_seconds"`
	ExtendThresholdSec int    `json:"extend_threshold_sec"`
	InitialStatus      string `json:"initial_status"` // scheduled (default) or active if window open
}

func (d Deps) AdminCreateAuction(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var body auctionCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var it models.Item
	if err := d.DB.First(&it, body.ItemID).Error; err != nil {
		if database.IsNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if it.Status != "approved" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item must be approved"})
		return
	}
	var exists int64
	d.DB.Model(&models.Auction{}).Where("item_id = ?", body.ItemID).Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "auction already exists for item"})
		return
	}
	start, err := time.Parse(time.RFC3339, body.StartAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_at RFC3339"})
		return
	}
	end, err := time.Parse(time.RFC3339, body.EndAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_at RFC3339"})
		return
	}
	if !end.After(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end must be after start"})
		return
	}
	if body.StartingPriceCents <= 0 || body.MinIncrementCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prices"})
		return
	}
	st := "scheduled"
	if body.InitialStatus == "active" {
		st = "active"
	}
	now := time.Now()
	if now.After(end) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end time already passed"})
		return
	}
	if st == "active" && now.Before(start) {
		st = "scheduled"
	}
	a := models.Auction{
		ItemID:             body.ItemID,
		Status:             st,
		RulesText:          body.RulesText,
		StartAt:            start,
		EndAt:              end,
		CurrentEndAt:       end,
		StartingPriceCents: body.StartingPriceCents,
		MinIncrementCents:  body.MinIncrementCents,
		ExtendSeconds:      body.ExtendSeconds,
		ExtendThresholdSec: body.ExtendThresholdSec,
		CreatedByAdminID:   adminID,
	}
	if err := d.DB.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = service.RefreshAuctionStatuses(d.DB, now)
	c.JSON(http.StatusCreated, a)
}

func (d Deps) AdminUpdateAuction(c *gin.Context) {
	var body auctionCreate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var a models.Auction
	if err := d.DB.First(&a, c.Param("id")).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if a.Status == "ended" || a.Status == "settled" || a.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auction closed"})
		return
	}
	start, err := time.Parse(time.RFC3339, body.StartAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_at"})
		return
	}
	end, err := time.Parse(time.RFC3339, body.EndAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_at"})
		return
	}
	a.RulesText = body.RulesText
	a.StartAt = start
	a.EndAt = end
	if a.CurrentEndAt.Before(end) {
		a.CurrentEndAt = end
	}
	a.StartingPriceCents = body.StartingPriceCents
	a.MinIncrementCents = body.MinIncrementCents
	a.ExtendSeconds = body.ExtendSeconds
	a.ExtendThresholdSec = body.ExtendThresholdSec
	if err := d.DB.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = service.RefreshAuctionStatuses(d.DB, time.Now())
	c.JSON(http.StatusOK, a)
}

func (d Deps) ListAuctions(c *gin.Context) {
	status := c.Query("status")
	tx := d.DB.Model(&models.Auction{}).Order("id desc")
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	var list []models.Auction
	tx.Find(&list)
	c.JSON(http.StatusOK, list)
}

func (d Deps) GetAuction(c *gin.Context) {
	_ = service.RefreshAuctionStatuses(d.DB, time.Now())
	var a models.Auction
	if err := d.DB.Preload("Item").First(&a, c.Param("id")).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (d Deps) AdminCancelAuction(c *gin.Context) {
	var a models.Auction
	if err := d.DB.First(&a, c.Param("id")).Error; err != nil {
		if database.IsNotFound(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if a.Status == "settled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already settled"})
		return
	}
	a.Status = "cancelled"
	if err := d.DB.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}
