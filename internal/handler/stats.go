package handler

import (
	"auction/internal/models"
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type auctionStats struct {
	AuctionID          uint      `json:"auction_id"`
	Status             string    `json:"status"`
	StartingPriceCents int64     `json:"starting_price_cents"`
	CurrentHighCents   int64     `json:"current_high_cents"`
	BidCount           int64     `json:"bid_count"`
	UniqueBidders      int64     `json:"unique_bidders"`
	WinnerUserID       *uint     `json:"winner_user_id"`
	EndsAt             time.Time `json:"ends_at"`
}

func (d Deps) AuctionStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var a models.Auction
	if err := d.DB.First(&a, id).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	var bidCount, uniq int64
	d.DB.Model(&models.Bid{}).Where("auction_id = ?", id).Count(&bidCount)
	d.DB.Model(&models.Bid{}).Where("auction_id = ?", id).Distinct("user_id").Count(&uniq)

	c.JSON(http.StatusOK, auctionStats{
		AuctionID:          uint(id),
		Status:             a.Status,
		StartingPriceCents: a.StartingPriceCents,
		CurrentHighCents:   a.CurrentHighCents,
		BidCount:           bidCount,
		UniqueBidders:      uniq,
		WinnerUserID:       a.WinnerUserID,
		EndsAt:             a.CurrentEndAt,
	})
}

func (d Deps) ExportAuctionReport(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var a models.Auction
	if err := d.DB.First(&a, id).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	var bids []models.Bid
	d.DB.Where("auction_id = ?", id).Order("id asc").Find(&bids)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"auction_id", "status", "starting_cents", "high_cents", "bid_id", "user_id", "amount_cents", "created_at"})
	for _, b := range bids {
		_ = w.Write([]string{
			strconv.FormatUint(uint64(a.ID), 10),
			a.Status,
			strconv.FormatInt(a.StartingPriceCents, 10),
			strconv.FormatInt(a.CurrentHighCents, 10),
			strconv.FormatUint(uint64(b.ID), 10),
			strconv.FormatUint(uint64(b.UserID), 10),
			strconv.FormatInt(b.AmountCents, 10),
			b.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	name := "auction_" + strconv.FormatUint(id, 10) + "_bids.csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "text/csv", buf.Bytes())
}

func (d Deps) GlobalStats(c *gin.Context) {
	var totalAuctions, ended, settled int64
	d.DB.Model(&models.Auction{}).Count(&totalAuctions)
	d.DB.Model(&models.Auction{}).Where("status = ?", "ended").Count(&ended)
	d.DB.Model(&models.Auction{}).Where("status = ?", "settled").Count(&settled)
	var vol int64
	d.DB.Raw(`SELECT IFNULL(SUM(amount_cents), 0) FROM payments WHERE status = ?`, "paid").Scan(&vol)
	c.JSON(http.StatusOK, gin.H{
		"total_auctions":    totalAuctions,
		"ended":             ended,
		"settled":           settled,
		"paid_volume_cents": vol,
	})
}
