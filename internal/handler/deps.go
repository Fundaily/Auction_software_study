package handler

import (
	"auction/internal/config"
	"auction/internal/middleware"
	"auction/internal/ws"

	"gorm.io/gorm"
)

type Deps struct {
	DB     *gorm.DB
	Cfg    config.Config
	BidLim *middleware.BidRateLimiter
	Hub    *ws.Hub
}
