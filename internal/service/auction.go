package service

import (
	"auction/internal/models"
	"time"

	"gorm.io/gorm"
)

// RefreshAuctionStatuses promotes scheduled→active and active→ended, then copies highest bidder to winner.
func RefreshAuctionStatuses(db *gorm.DB, now time.Time) error {
	if err := db.Model(&models.Auction{}).
		Where("status = ? AND start_at <= ? AND current_end_at > ?", "scheduled", now, now).
		Update("status", "active").Error; err != nil {
		return err
	}
	if err := db.Model(&models.Auction{}).
		Where("status = ? AND current_end_at <= ?", "active", now).
		Update("status", "ended").Error; err != nil {
		return err
	}
	// Copy winning bidder to winner_user_id for ended auctions.
	if err := db.Exec(`
		UPDATE auctions
		SET winner_user_id = highest_bidder_id
		WHERE status = 'ended'
		  AND winner_user_id IS NULL
		  AND highest_bidder_id IS NOT NULL
	`).Error; err != nil {
		return err
	}
	return nil
}

func EnsureSettlementPayment(db *gorm.DB, auctionID uint) (*models.Payment, error) {
	var pay models.Payment
	if err := db.Where("auction_id = ?", auctionID).First(&pay).Error; err == nil {
		return &pay, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	var a models.Auction
	if err := db.First(&a, auctionID).Error; err != nil {
		return nil, err
	}
	if a.Status != "ended" && a.Status != "settled" {
		return nil, ErrAuctionNotActive
	}
	if a.WinnerUserID == nil || a.CurrentHighCents <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	pay = models.Payment{
		AuctionID:   auctionID,
		PayerID:     *a.WinnerUserID,
		AmountCents: a.CurrentHighCents,
		Status:      "pending",
		Provider:    "mock",
	}
	if err := db.Create(&pay).Error; err != nil {
		return nil, err
	}
	return &pay, nil
}
