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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type AdditionalInfo = map[string]any

type Client struct {
	room *Room
	conn *websocket.Conn
	user User
	send chan []byte
}

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type Message struct {
	MessageType string    `json:"type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	User        User      `json:"user"`
}

type RoomResponse struct {
	ID             uint           `json:"id"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
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

var (
	version       string = "unknown"
	gitCommit     string = "unknown"
	gitBranch     string = "unknown"
	gitRepository string = "unknown"
	buildTime     string = "unknown"
)

// POST /rooms
func createRoomHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var additionalInfo AdditionalInfo
	err := decoder.Decode(&additionalInfo)
	if err != nil {
		additionalInfo = map[string]any{}
	}
	room := hub.CreateRoom(additionalInfo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]uint{"roomID": room.id})
}

// GET /rooms
func getAllRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms := hub.GetAllRoomIDs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]uint{"rooms": rooms})
}

// GET /rooms/{roomID}
func getRoomIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	payload := RoomResponse{
		ID:             room.id,
		AdditionalInfo: room.additionalInfo,
	}
	json.NewEncoder(w).Encode(payload)
}

// GET /join/{roomID}?user=""
func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	userName := r.URL.Query().Get("user")

	if userName == "" {
		userName = defaultUserName[rand.Intn(len(defaultUserName))]
	}

	user := User{
		ID:   uuid.New(),
		Name: userName,
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
	log.Printf("new client %s with id %s joined room %d", user.Name, user.ID, roomID)

	sysUser := User{
		ID:   uuid.New(),
		Name: "system",
	}
	hello := Message{
		MessageType: "system",
		Message:     fmt.Sprintf("%s joined room %d", user.Name, roomID),
		Timestamp:   time.Now(),
		User:        sysUser,
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
			User:        c.user,
		}
		b, _ := json.Marshal(payload)
		c.room.broadcast <- b
		log.Printf("new message recieved. message: '%s', user: '%s' ", payload.Message, c.user.ID)
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
	r.HandleFunc("/rooms/{roomID}", getRoomIDHandler).Methods("GET")
	r.HandleFunc("/join/{roomID}", wsHandler).Methods("GET")
	r.HandleFunc("/info", getInfoHandler).Methods("GET")

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
