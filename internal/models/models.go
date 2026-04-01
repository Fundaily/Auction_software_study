package models

import "time"

type User struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Username     string `gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	DisplayName  string `gorm:"size:128"`
	IsAdmin      bool   `gorm:"default:false;index"`
}

type Item struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SellerID     uint   `gorm:"index;not null"`
	Title        string `gorm:"size:255;not null"`
	Description  string `gorm:"type:text"`
	ImagePaths   string `gorm:"type:text"` // JSON array of relative paths
	Status       string `gorm:"size:32;index;not null"` // draft,pending_review,approved,rejected
	RejectReason string `gorm:"type:text"`
}

type Auction struct {
	ID                 uint `gorm:"primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ItemID             uint      `gorm:"uniqueIndex;not null"`
	Status             string    `gorm:"size:32;index;not null"` // scheduled,active,ended,cancelled,settled
	RulesText          string    `gorm:"type:text"`
	StartAt            time.Time `gorm:"index"`
	EndAt              time.Time // planned end (immutable reference)
	CurrentEndAt       time.Time // may extend with late bids
	StartingPriceCents int64     `gorm:"not null"`
	MinIncrementCents  int64     `gorm:"not null"`
	ExtendSeconds      int       // extend CurrentEndAt by this many seconds
	ExtendThresholdSec int       // if bid placed within this many seconds before CurrentEndAt, extend
	CurrentHighCents   int64     `gorm:"default:0"`
	HighestBidderID    *uint     `gorm:"index"`
	WinnerUserID       *uint     `gorm:"index"`
	CreatedByAdminID   uint      `gorm:"not null"`
	Item               Item      `gorm:"foreignKey:ItemID"`
}

type Bid struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	AuctionID    uint  `gorm:"index;not null"`
	UserID       uint  `gorm:"index;not null"`
	AmountCents  int64 `gorm:"not null"`
	IsOutbid     bool  `gorm:"default:false"` // optional marker for analytics
}

type Payment struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AuctionID   uint       `gorm:"uniqueIndex;not null"`
	PayerID     uint       `gorm:"not null"`
	AmountCents int64      `gorm:"not null"`
	Status      string     `gorm:"size:32;index;not null"` // pending,paid,failed,refunded
	Provider    string     `gorm:"size:64"`                // mock,third_party
	ExternalRef string     `gorm:"size:255"`
	PaidAt      *time.Time
}

type Review struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	AuctionID  uint   `gorm:"uniqueIndex:idx_review_pair;not null"`
	FromUserID uint   `gorm:"uniqueIndex:idx_review_pair;not null"`
	ToUserID   uint   `gorm:"not null"`
	Rating     int    `gorm:"not null"`
	Comment    string `gorm:"type:text"`
}
