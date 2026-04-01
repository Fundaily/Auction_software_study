package handler

import (
	"auction/internal/database"
	"auction/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (d Deps) AdminListPendingItems(c *gin.Context) {
	var items []models.Item
	d.DB.Where("status = ?", "pending_review").Order("id asc").Find(&items)
	c.JSON(http.StatusOK, items)
}

type adminItemReviewBody struct {
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

func (d Deps) AdminReviewItem(c *gin.Context) {
	var body adminItemReviewBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if it.Status != "pending_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item not pending review"})
		return
	}
	if body.Approve {
		it.Status = "approved"
		it.RejectReason = ""
	} else {
		it.Status = "rejected"
		it.RejectReason = body.Reason
	}
	if err := d.DB.Save(&it).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, it)
}
