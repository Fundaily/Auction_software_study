package payment

import (
	"auction/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Gateway abstracts third-party checkout; MockGateway simulates async confirmation via webhook.
type Gateway interface {
	Name() string
	CreateCheckout(db *gorm.DB, pay *models.Payment) (externalRef string, err error)
}

type MockGateway struct{}

func (MockGateway) Name() string { return "mock" }

func (MockGateway) CreateCheckout(db *gorm.DB, pay *models.Payment) (string, error) {
	ref := "mock_" + uuid.NewString()[:8]
	return ref, db.Model(pay).Updates(map[string]any{
		"external_ref": ref,
		"provider":     "mock",
	}).Error
}

// ConfirmMock marks payment paid and settles the auction (simulates checkout / webhook success).
func ConfirmMock(db *gorm.DB, paymentID uint) error {
	now := time.Now()
	var p models.Payment
	if err := db.First(&p, paymentID).Error; err != nil {
		return err
	}
	if err := db.Model(&models.Payment{}).Where("id = ? AND status = ?", paymentID, "pending").Updates(map[string]any{
		"status":  "paid",
		"paid_at": now,
	}).Error; err != nil {
		return err
	}
	return db.Model(&models.Auction{}).Where("id = ?", p.AuctionID).Update("status", "settled").Error
}
