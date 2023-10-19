package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	webSocketUpgrader = websocket.Upgrader{CheckOrigin: checkOrigin, ReadBufferSize: 1024, WriteBufferSize: 1024}
)

type Manager struct {
	clients ClientList
	sync.RWMutex
	otps RetentionMap

	handlers map[string]EventHandler
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		handlers: make(map[string]EventHandler),
		otps:     NewRetentionMap(ctx, 5*time.Second),
	}

	m.setupEventHandlers()

	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handlers, ok := m.handlers[event.Type]; ok {
		if err := handlers(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("no such event type")
	}
}

func SendMessage(event Event, c *Client) error {
	var chatEvent SendMessageEvent
	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return err
	}
	var boarMessage NewMessageEvent
	boarMessage.Sent = time.Now()
	boarMessage.Message = chatEvent.Message
	boarMessage.From = chatEvent.From

	data, err := json.Marshal(boarMessage)

	if err != nil {
		return err
	}

	outgoingEvent := Event{
		Type:    EventNewMessage,
		Payload: data,
	}

	for client := range c.manager.clients {
		client.egress <- outgoingEvent
	}

	fmt.Println(event)
	return nil
}

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !m.otps.VerifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("new connection")

	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := NewClient(conn, m)
	m.addClient(client)

	go client.readMessage()
	go client.writeMessages()
}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Username == "percy" && req.Password == "123" {
		type responce struct {
			OTP string `json:"otp"`
		}

		otp := m.otps.NewOTP()

		resp := responce{OTP: otp.Key}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return

	}
	w.WriteHeader(http.StatusUnauthorized)

}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()
	m.clients[client] = true

}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}

}

func checkOrigin(r *http.Request) bool {

	origin := r.Header.Get("Origin")
	switch origin {
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}
