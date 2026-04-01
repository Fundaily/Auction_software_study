package handler

import (
	"auction/internal/middleware"
	"strings"

	"github.com/gin-gonic/gin"
)

func (d Deps) Mount(r *gin.Engine) {
	r.MaxMultipartMemory = 16 << 20

	if p := strings.TrimSuffix(d.Cfg.StaticURLPath, "/"); p != "" {
		r.Static(p, d.Cfg.UploadDir)
	}

	pub := r.Group("/api")
	{
		pub.POST("/register", d.Register)
		pub.POST("/login", d.Login)
		pub.GET("/auctions", d.ListAuctions)
		pub.GET("/auctions/:id", d.GetAuction)
		pub.GET("/items/:id", d.GetItem)
		pub.GET("/auctions/:id/bids", d.ListBids)
		pub.GET("/auctions/:id/ws", d.AuctionWS)
		pub.GET("/auctions/:id/stats", d.AuctionStats)
		pub.GET("/users/:id/reviews", d.ListReviewsForUser)
		pub.POST("/webhooks/payment", d.PaymentWebhook)
	}

	authz := r.Group("/api")
	authz.Use(middleware.JWT(d.Cfg.JWTSecret))
	{
		authz.POST("/items", d.CreateItem)
		authz.GET("/me/items", d.ListMyItems)
		authz.POST("/items/:id/images", d.UploadItemImages)
		authz.POST("/items/:id/submit-review", d.SubmitItemReview)

		authz.POST("/auctions/:id/bids", d.PlaceBid)

		authz.POST("/auctions/:id/payments", d.CreatePayment)
		authz.GET("/payments/:id", d.GetPayment)
		authz.POST("/payments/:id/confirm", d.ConfirmPayment)

		authz.POST("/reviews", d.CreateReview)
	}

	adm := r.Group("/api/admin")
	adm.Use(middleware.JWT(d.Cfg.JWTSecret), middleware.RequireAdmin)
	{
		adm.GET("/items/pending", d.AdminListPendingItems)
		adm.POST("/items/:id/review", d.AdminReviewItem)

		adm.POST("/auctions", d.AdminCreateAuction)
		adm.PATCH("/auctions/:id", d.AdminUpdateAuction)
		adm.POST("/auctions/:id/cancel", d.AdminCancelAuction)

		adm.GET("/auctions/:id/stats", d.AuctionStats)
		adm.GET("/auctions/:id/export.csv", d.ExportAuctionReport)
		adm.GET("/stats/summary", d.GlobalStats)
	}
}
