package middleware

import (
	"auction/internal/auth"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const ctxUID = "userID"
const ctxAdmin = "isAdmin"
const ctxClaims = "claims"

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		raw := strings.TrimSpace(h[7:])
		cl, err := auth.ParseToken(secret, raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ctxClaims, cl)
		c.Set(ctxUID, cl.UserID)
		c.Set(ctxAdmin, cl.IsAdmin)
		c.Next()
	}
}

func RequireAdmin(c *gin.Context) {
	if v, ok := c.Get(ctxAdmin); ok {
		if b, _ := v.(bool); b {
			c.Next()
			return
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
}

func GetUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get(ctxUID)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// BidRateLimiter: token bucket per (user, auction) to mitigate bid brushing.
type BidRateLimiter struct {
	mu     sync.Mutex
	lim    map[string]*rate.Limiter
	burst  int
	every  rate.Limit
}

func NewBidLimiter(burst int, minGap time.Duration) *BidRateLimiter {
	ev := rate.Every(minGap)
	if minGap <= 0 {
		ev = rate.Inf
	}
	return &BidRateLimiter{
		lim:   make(map[string]*rate.Limiter),
		burst: burst,
		every: ev,
	}
}

func (b *BidRateLimiter) key(uid, aid uint) string {
	return fmt.Sprintf("%d:%d", uid, aid)
}

func (b *BidRateLimiter) Allow(uid, aid uint) bool {
	k := b.key(uid, aid)
	b.mu.Lock()
	defer b.mu.Unlock()
	limiter, ok := b.lim[k]
	if !ok {
		limiter = rate.NewLimiter(b.every, b.burst)
		b.lim[k] = limiter
	}
	return limiter.Allow()
}
