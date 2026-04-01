package handler

import (
	"auction/internal/database"
	"auction/internal/middleware"
	"auction/internal/models"
	"auction/internal/service"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type bidBody struct {
	AmountCents int64 `json:"amount_cents" binding:"required"`
}

func (d Deps) PlaceBid(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var body bidBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !d.BidLim.Allow(uid, uri.ID) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "bid rate limited"})
		return
	}
	bid, err := service.PlaceBid(d.DB, uri.ID, uid, body.AmountCents, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAuctionNotActive),
			errors.Is(err, service.ErrOutsideWindow):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrBidTooLow):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case database.IsNotFound(err):
			c.Status(http.StatusNotFound)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var a models.Auction
	_ = d.DB.First(&a, uri.ID).Error
	d.Hub.BroadcastJSON(uri.ID, gin.H{
		"type":         "bid",
		"auction_id":   uri.ID,
		"amount_cents": bid.AmountCents,
		"user_id":      uid,
		"current_high": a.CurrentHighCents,
		"ends_at":      a.CurrentEndAt,
	})
	c.JSON(http.StatusCreated, bid)
}

func (d Deps) ListBids(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var bids []models.Bid
	d.DB.Where("auction_id = ?", uri.ID).Order("id desc").Limit(200).Find(&bids)
	c.JSON(http.StatusOK, bids)
}
