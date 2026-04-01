package service

import "errors"

var (
	ErrAuctionNotActive = errors.New("auction not active")
	ErrOutsideWindow    = errors.New("outside bidding window")
	ErrBidTooLow        = errors.New("bid below minimum")
	ErrItemNotApproved  = errors.New("item not approved")
	ErrInvalidStatus    = errors.New("invalid status transition")
)
