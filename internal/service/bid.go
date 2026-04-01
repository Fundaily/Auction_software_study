package service

import (
	"auction/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PlaceBid(db *gorm.DB, auctionID, userID uint, amountCents int64, now time.Time) (*models.Bid, error) {
	if err := RefreshAuctionStatuses(db, now); err != nil {
		return nil, err
	}

	var created *models.Bid
	err := db.Transaction(func(tx *gorm.DB) error {
		var a models.Auction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, auctionID).Error; err != nil {
			return err
		}
		if a.Status != "active" {
			return ErrAuctionNotActive
		}
		if now.Before(a.StartAt) || !now.Before(a.CurrentEndAt) {
			return ErrOutsideWindow
		}

		minNext := a.StartingPriceCents
		if a.CurrentHighCents > 0 {
			minNext = a.CurrentHighCents + a.MinIncrementCents
		}
		if amountCents < minNext {
			return ErrBidTooLow
		}

		bid := models.Bid{AuctionID: auctionID, UserID: userID, AmountCents: amountCents}
		if err := tx.Create(&bid).Error; err != nil {
			return err
		}

		uid := userID
		a.CurrentHighCents = amountCents
		a.HighestBidderID = &uid

		if a.ExtendThresholdSec > 0 && a.ExtendSeconds > 0 {
			remaining := a.CurrentEndAt.Sub(now)
			if remaining <= time.Duration(a.ExtendThresholdSec)*time.Second {
				a.CurrentEndAt = a.CurrentEndAt.Add(time.Duration(a.ExtendSeconds) * time.Second)
			}
		}

		if err := tx.Save(&a).Error; err != nil {
			return err
		}
		created = &bid
		return nil
	})
	return created, err
}
