package handler

import (
	"auction/internal/middleware"
	"auction/internal/models"
	"auction/internal/payment"
	"auction/internal/service"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (d Deps) CreatePayment(c *gin.Context) {
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
	_ = service.RefreshAuctionStatuses(d.DB, time.Now())
	pay, err := service.EnsureSettlementPayment(d.DB, uri.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no payable auction"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pay.PayerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "not winner"})
		return
	}
	gw := payment.MockGateway{}
	ref, err := gw.CreateCheckout(d.DB, pay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = d.DB.First(pay, pay.ID).Error
	c.JSON(http.StatusOK, gin.H{
		"payment":       pay,
		"external_ref":  ref,
		"checkout_hint": fmt.Sprintf("POST /api/payments/%d/confirm with admin or winner token (dev)", pay.ID),
	})
}

func (d Deps) GetPayment(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var p models.Payment
	if err := d.DB.First(&p, id64).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (d Deps) ConfirmPayment(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var p models.Payment
	if err := d.DB.First(&p, id64).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	admin, _ := c.Get("isAdmin")
	isAdmin, _ := admin.(bool)
	if !isAdmin && p.PayerID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if err := payment.ConfirmMock(d.DB, uint(id64)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = d.DB.First(&p, id64).Error
	c.JSON(http.StatusOK, p)
}

type webhookBody struct {
	ExternalRef string `json:"external_ref" binding:"required"`
}

func (d Deps) PaymentWebhook(c *gin.Context) {
	secret := c.GetHeader("X-Webhook-Secret")
	if d.Cfg.PaymentWebhook != "" && secret != d.Cfg.PaymentWebhook {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid secret"})
		return
	}
	var body webhookBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var p models.Payment
	if err := d.DB.Where("external_ref = ?", body.ExternalRef).First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment"})
		return
	}
	if err := payment.ConfirmMock(d.DB, p.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
