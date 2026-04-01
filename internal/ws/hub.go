package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.Mutex
	socks map[uint][]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{socks: make(map[uint][]*websocket.Conn)}
}

func (h *Hub) Register(auctionID uint, c *websocket.Conn) {
	h.mu.Lock()
	h.socks[auctionID] = append(h.socks[auctionID], c)
	h.mu.Unlock()
}

func (h *Hub) BroadcastJSON(auctionID uint, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.socks[auctionID]
	keep := list[:0]
	for _, c := range list {
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			_ = c.Close()
			continue
		}
		keep = append(keep, c)
	}
	h.socks[auctionID] = keep
}
