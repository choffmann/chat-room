package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Room struct {
	id         uint
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	closed     chan struct{}
}

type Client struct {
	room *Room
	conn *websocket.Conn
	user string
	send chan []byte
}

type Message struct {
	MessageType string    `json:"type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	User        *string   `json:"user"`
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[uint]*Room
}

var (
	hub      = &Hub{rooms: make(map[uint]*Room)}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	roomCounter     = 0
	roomMu          sync.Mutex
	defaultUserName = []string{
		"Toni Tester",
		"Harald Hüftschmerz",
		"Andre Android",
		"Hans Hotfix",
		"Peter Push",
		"Rebase Randy",
		"Prof. Prokrastination",
	}
)

func newRoomID() uint {
	roomMu.Lock()
	defer roomMu.Unlock()
	roomCounter++
	return uint(roomCounter)
}

func (h *Hub) CreateRoom() *Room {
	id := newRoomID()
	room := &Room{
		id:         id,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
	}

	h.mu.Lock()
	h.rooms[id] = room
	h.mu.Unlock()

	go room.run()
	return room
}

func (h *Hub) GetRoom(id uint) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[id]
	return r, ok
}

func (h *Hub) GetAllRoomIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]uint, 0, len(h.rooms))
	for _, room := range h.rooms {
		rooms = append(rooms, room.id)
	}
	return rooms
}

func (h *Hub) DeleteRoom(id uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (r *Room) run() {
	defer close(r.closed)

	for {
		select {
		case c := <-r.register:
			r.clients[c] = true
		case c := <-r.unregister:
			if _, ok := r.clients[c]; ok {
				delete(r.clients, c)
				close(c.send)
				if len(r.clients) == 0 {
					hub.DeleteRoom(r.id)
					return
				}
			}

		case msg := <-r.broadcast:
			for c := range r.clients {
				select {
				case c.send <- msg:
				default:
					delete(r.clients, c)
					close(c.send)
				}
			}
		}
	}
}

// POST /rooms
func createRoomHandler(w http.ResponseWriter, r *http.Request) {
	room := hub.CreateRoom()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]uint{"roomID": room.id})
}

// GET /rooms
func getAllRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms := hub.GetAllRoomIDs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]uint{"rooms": rooms})
}

// GET /join/{roomID}?user=""
func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		http.Error(w, "can't parse room id to uint", http.StatusNotFound)
		return
	}

	user := r.URL.Query().Get("user")

	if user == "" {
		user = defaultUserName[rand.Intn(len(defaultUserName))]
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	client := &Client{
		room: room,
		conn: conn,
		user: user,
		send: make(chan []byte, 256),
	}

	room.register <- client

	sysUser := "system"
	hello := Message{
		MessageType: "system",
		Message:     fmt.Sprintf("%s joined room %d", user, roomID),
		Timestamp:   time.Now(),
		User:        &sysUser,
	}

	b, _ := json.Marshal(hello)
	for c := range room.clients {
		c.send <- b
	}

	go client.writePump()
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(8192)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		mt, message, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Println("read:", err)
			}
			break
		}
		if mt != websocket.TextMessage {
			continue
		}

		payload := Message{
			MessageType: "message",
			Message:     string(message),
			Timestamp:   time.Now(),
			User:        &c.user,
		}
		b, _ := json.Marshal(payload)
		c.room.broadcast <- b
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/rooms", createRoomHandler).Methods("POST")
	r.HandleFunc("/rooms", getAllRoomsHandler).Methods("GET")
	r.HandleFunc("/join/{roomID}", wsHandler).Methods("GET")

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())
}
