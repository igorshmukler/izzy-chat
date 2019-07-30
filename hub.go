package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"golang.org/x/net/websocket"
)

type Message struct {
	Username    string `json:"username"`
	MessageType string `json:"type"`
	Recepient   string `json:"recepient"`
	Payload     string `json:"payload"`
}

type ChatConn struct {
	Socket   *websocket.Conn
	Username string
}

type Hub struct {
	clients          map[string]*websocket.Conn
	addClientChan    chan ChatConn
	removeClientChan chan ChatConn
	broadcastChan    chan Message
	dbConnection     *sql.DB
}

func newHub(connection *sql.DB) *Hub {
	return &Hub{
		clients:          make(map[string]*websocket.Conn),
		addClientChan:    make(chan ChatConn),
		removeClientChan: make(chan ChatConn),
		broadcastChan:    make(chan Message),
		dbConnection:     connection,
	}
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.addClientChan:
			h.addClient(conn)
		case conn := <-h.removeClientChan:
			h.removeClient(conn)
		case m := <-h.broadcastChan:
			h.broadcastMessage(m)
		}
	}
}

func (h *Hub) removeClient(conn ChatConn) {
	username := conn.Username
	delete(h.clients, username)
}

func (h *Hub) addClient(conn ChatConn) {
	socket := conn.Socket
	username := conn.Username
	h.clients[username] = socket
}

func (h *Hub) broadcastMessage(m Message) {
	var members []string
	// system messages are delivered directly to users, all others to channels
	if "system" == m.Username {
		members = []string{m.Recepient}
	} else {
		members = getChannelMembers(h, m.Recepient)
		// is this our first message to the channel
		if !contains(members, m.Username) {
			err := addChannelMember(h, m.Recepient, m.Username)
			if err != nil {
				fmt.Printf("cannot add user %v to channel %v\n", m.Username, m.Recepient)
				return
			}
		}
	}
	for name, conn := range h.clients {
		// Should send to all channel members, except ourselves
		if contains(members, name) && m.Username != name {
			fmt.Println("broadcasting message: ", m)
			err := websocket.Message.Send(conn, messageToArrayBuffer(&m))
			if err != nil {
				fmt.Printf("Error broadcasting message to user: %v error: %v", name, err)
			}
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func uintToString(arr []byte) string {
	decoded, err := base64.StdEncoding.DecodeString(string(arr))
	if err != nil {
		fmt.Println("error decoding string")
		return ""
	}
	return string(decoded)
}

func messageToArrayBuffer(m *Message) []byte {
	senderArray := base64.StdEncoding.EncodeToString([]byte(m.Username))
	recepientArray := base64.StdEncoding.EncodeToString([]byte(m.Recepient))
	payloadArray := base64.StdEncoding.EncodeToString([]byte(m.Payload))
	var opcode byte
	switch m.MessageType {
	case "api.user.login":
		opcode = 1
	case "api.channels.add":
		opcode = 2
	case "api.channels.list":
		opcode = 3
	case "api.user.authorized":
		opcode = 4
	case "api.user.deauthorized":
		opcode = 5
	case "api.channels.all":
		opcode = 6
	case "api.channels.created":
		opcode = 7
	case "api.message.new":
		opcode = 8
	case "api.message.broadcast":
		opcode = 9
	case "api.login.error":
		opcode = 10
	case "api.message.history":
		opcode = 11
	case "api.message.archive":
		opcode = 12
	case "api.channels.direct":
		opcode = 13
	case "api.user.signup":
		opcode = 14
	case "api.user.created":
		opcode = 15
	case "api.user.online":
		opcode = 16
	case "api.user.count":
		opcode = 17
	default:
		fmt.Println("Unsupported message type: ", m.MessageType)
		return make([]byte, 0)
	}
	// our messages are sent as ArrayBuffer
	// header
	// - 3 bytes/UInt8 positions: opcode, encoded sender username size, encoded receipent size
	// - encoded sender username
	// - encoded recepient
	// body
	// - payload as a base64 encoded string, packed in UInt8 bytes
	headerArray := []byte{opcode, byte(len(senderArray)), byte(len(recepientArray))}
	tmp := append(headerArray, []byte(senderArray)...)
	tmp = append(tmp, []byte(recepientArray)...)
	tmp = append(tmp, []byte(payloadArray)...)
	return tmp
}

func arrayBufferToMessage(p *[]byte, m *Message) {
	// our messages are sent as ArrayBuffer
	// header
	// - 3 bytes/UInt8 positions: opcode, encoded sender username size, encoded receipent size
	// - encoded sender username
	// - encoded recepient
	// body
	// - payload as a base64 encoded string, packed in UInt8 bytes
	opcode := int((*p)[0])
	senderSize := int((*p)[1])
	recepientSize := int((*p)[2])
	sender := append((*p)[3 : senderSize+3])
	recepient := append((*p)[3+senderSize : 3+senderSize+recepientSize])
	payload := append((*p)[3+senderSize+recepientSize:])
	switch opcode {
	case 1:
		m.MessageType = "api.user.login"
	case 2:
		m.MessageType = "api.channels.add"
	case 3:
		m.MessageType = "api.channels.list"
	case 4:
		m.MessageType = "api.user.authorized"
	case 5:
		m.MessageType = "api.user.deauthorized"
	case 6:
		m.MessageType = "api.channels.all"
	case 7:
		m.MessageType = "api.channels.created"
	case 8:
		m.MessageType = "api.message.new"
	case 9:
		m.MessageType = "api.message.broadcast"
	case 10:
		m.MessageType = "api.login.error"
	case 11:
		m.MessageType = "api.message.history"
	case 12:
		m.MessageType = "api.message.archive"
	case 13:
		m.MessageType = "api.channels.direct"
	case 14:
		m.MessageType = "api.user.signup"
	case 15:
		m.MessageType = "api.user.created"
	case 16:
		m.MessageType = "api.user.online"
	case 17:
		m.MessageType = "api.user.count"
	default:
		fmt.Println("unrecognized message format, opcode: ", opcode)
		return
	}
	m.Username = uintToString(sender)
	m.Recepient = uintToString(recepient)
	m.Payload = uintToString(payload)
}

func handler(ws *websocket.Conn, h *Hub) {
	go h.run()

	for {
		var m Message
		var p []byte
		er := websocket.Message.Receive(ws, &p)
		arrayBufferToMessage(&p, &m)
		fmt.Println("message: ", m)
		if er != nil {
			fmt.Println("Error receiving message: ", er)
			return
		}
		config := ws.Config()
		query := config.Location.RawQuery
		q, err := url.ParseQuery(query)
		if err != nil {
			fmt.Println("Error parsing query string: ", err)
			return
		}
		var token string
		if len(q["accessToken"]) > 0 {
			token = q["accessToken"][0]
		} else {
			token = findSessionToken(h, m.Username)
		}
		switch m.MessageType {
		case "api.message.new":
			if !isAuthorized(h, token, m.Username) {
				fmt.Println("api.message.new: missing a valid token.")
				return
			}
			h.broadcastChan <- Message{
				Username:    m.Username,
				MessageType: "api.message.broadcast",
				Recepient:   m.Recepient,
				Payload:     m.Payload}
			err = saveUserMessage(h, &m)
			if err != nil {
				log.Println(err)
				return
			}
		case "api.user.login":
			fmt.Println("api.login username:", m.Username)
			// check credentials
			if !checkCredentials(h, m.Username, m.Payload) {
				fmt.Println("Invalid credentials.")
				err = websocket.Message.Send(ws, messageToArrayBuffer(&Message{
					Username:    "system",
					MessageType: "api.login.error",
					Recepient:   m.Username,
					Payload:     "invalid credentials"}))
				if err != nil {
					fmt.Println("Error sending login message: ", err)
					return
				}
				return
			}
			h.addClientChan <- ChatConn{ws, m.Username}
			sessionToken := setSessionToken(h, m.Username)
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.user.authorized",
				Recepient:   m.Username,
				Payload:     sessionToken}
		case "api.user.logout":
			if isAuthorized(h, token, m.Username) {
				h.broadcastChan <- Message{
					Username:    "system",
					MessageType: "api.user.deauthorized",
					Recepient:   m.Username,
					Payload:     "user requested logout"}
				conn := ChatConn{ws, m.Username}
				// remove client socket and cookie/hash
				err := removeUserSession(h, m.Username)
				if err != nil {
					fmt.Println("Error removing user session: ", err)
				}
				h.removeClient(conn)
			}
		case "api.channels.list":
			if !isAuthorized(h, token, m.Username) {
				fmt.Println("api.channels.list: missing a valid token.")
				return
			}
			channels := getUserChannels(h, m.Username)
			channelsJson, _ := json.Marshal(channels)
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.channels.all",
				Recepient:   m.Username,
				Payload:     string(channelsJson)}
		case "api.channels.add":
			if !isAuthorized(h, token, m.Username) {
				fmt.Println("api.channels.add: missing a valid token.")
				return
			}
			fmt.Printf("adding channel %v created by user %v\n", m.Payload, m.Username)
			err = createNewChannel(h, m.Username, m.Payload)
			if err != nil {
				fmt.Printf("error creating channel %v for user %v\n", m.Payload, m.Username)
				return
			}
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.channels.created",
				Recepient:   m.Username,
				Payload:     m.Payload}
		case "api.channels.remove":
			fmt.Println("api.channels.remove")
			if !isAuthorized(h, token, m.Username) {
				fmt.Println("api.channels.remove: missing a valid token.")
				return
			}
			err = archiveExistingChannel(h, m.Username, m.Payload)
			if err != nil {
				fmt.Printf("error archiving channel %v for user %v\n", m.Payload, m.Username)
				return
			}
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.channels.archived",
				Recepient:   m.Username,
				Payload:     m.Payload}
		case "api.message.history":
			// fmt.Println("api.message.history for channel: ", m.Payload)
			if !isAuthorized(h, token, m.Username) {
				fmt.Println("api.message.history: missing a valid token.")
				return
			}
			messages := getMessageHistory(h, m.Payload)
			messagesJson, _ := json.Marshal(messages)
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.message.archive",
				Recepient:   m.Username,
				Payload:     string(messagesJson)}
		case "api.channels.direct":
			fmt.Println("api.channels.direct")
			if !isAuthorized(h, token, m.Username) {
				fmt.Println("api.channels.direct: missing a valid token.")
				return
			}
			err = ensureDirectChannel(h, m.Username, m.Payload)
			if err != nil {
				fmt.Printf("error ensuring direct channel %v for user %v\n", m.Payload, m.Username)
				return
			}
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.channels.created",
				Recepient:   m.Username,
				Payload:     m.Payload}
		case "api.user.signup":
			fmt.Println("api.user.signup payload: " + m.Payload)
			var payload map[string]interface{}
			err := json.Unmarshal([]byte(m.Payload), &payload)
			if err != nil {
				fmt.Printf("Error unmarshaling: %v - %v\n", []byte(m.Payload), err)
				return
			}
			fmt.Printf("Email: %v Hash %v\n", payload["email"], payload["passwordHash"])
			email := fmt.Sprintf("%v", payload["email"])
			passwordHash := fmt.Sprintf("%v", payload["passwordHash"])
			err = createNewUser(h, m.Username, email, passwordHash)
			if err != nil {
				fmt.Println("error creating user: ", m.Username)
				return
			}
			err = websocket.Message.Send(ws, messageToArrayBuffer(&Message{
				Username:    "system",
				MessageType: "api.user.created",
				Recepient:   m.Username,
				Payload:     "signup success"}))
			if err != nil {
				fmt.Println("Error sending signup success message: ", err)
				return
			}
		case "api.user.online":
			fmt.Println("api.user.online stub")
			count := len(h.clients)
			fmt.Printf("found %v connected clients", count)
			h.broadcastChan <- Message{
				Username:    "system",
				MessageType: "api.user.count",
				Recepient:   m.Username,
				Payload:     fmt.Sprintf("%v", count)}
		default:
			fmt.Println("unrecognized message format. type: ", m.MessageType)
			if isAuthorized(h, token, m.Username) {
				h.broadcastChan <- Message{
					Username:    "system",
					MessageType: "api.user.logout",
					Recepient:   m.Username,
					Payload:     "protocol error"}
				conn := ChatConn{ws, m.Username}
				h.removeClient(conn)
				err := removeUserSession(h, m.Username)
				if err != nil {
					fmt.Println("Error removing user session: ", err)
				}
			}
		}
	}
}
