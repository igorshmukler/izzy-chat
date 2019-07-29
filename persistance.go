package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func findSessionToken(h *Hub, username string) string {
	res := ""
	selectUserSql := fmt.Sprintf("SELECT Token FROM dbo.Sessions WHERE Username = '%v'", username)
	fmt.Println("selectUserSql: ", selectUserSql)
	rows, err := h.dbConnection.Query(selectUserSql)
	if err != nil {
		log.Println(err)
		return res
	}
	for rows.Next() {
		var token string
		err = rows.Scan(&token)
		if err != nil {
			fmt.Println(err)
			return res
		}
		fmt.Printf("found session token %v for user: %v\n", token, username)
		res = token
	}
	return res
}

func setSessionToken(h *Hub, username string) string {
	// if there is an already session token, just return it
	token := findSessionToken(h, username)
	if len(token) > 0 {
		return token
	}
	// generate a new token
	token = randStringRunes(20)
	insertMessageSql := fmt.Sprintf("INSERT INTO dbo.Sessions VALUES('%v', '%v')", username, token)
	fmt.Println("insert statement: ", insertMessageSql)
	_, err := h.dbConnection.Exec(insertMessageSql)
	if err != nil {
		fmt.Println(err)
	}
	return token
}

// store and check session token
func isAuthorized(h *Hub, token string, username string) bool {
	if h.clients[username] == nil {
		return false
	}
	sessionToken := findSessionToken(h, username)
	return len(sessionToken) > 0
}

func checkCredentials(h *Hub, username string, passwordHash string) bool {
	if len(username) == 0 || len(passwordHash) == 0 {
		return false
	}
	selectUserSql := fmt.Sprintf("SELECT * FROM dbo.Users WHERE Username = '%v' AND PasswordHash = '%v'", username, passwordHash)
	fmt.Println("selectUserSql: ", selectUserSql)
	res, err := h.dbConnection.Exec(selectUserSql)
	if err != nil {
		log.Println(err)
		return false
	}
	rows, er := res.RowsAffected()
	if er != nil {
		log.Println(er)
		return false
	}
	return rows > 0
}

func saveUserMessage(h *Hub, m *Message) error {
	insertMessageSql := fmt.Sprintf("INSERT INTO dbo.Messages (Username, MessageType, Recepient, Payload, Stamp) VALUES('%v', '%v', '%v', '%v', '%v')",
		m.Username, "api.message.broadcast", m.Recepient, m.Payload, time.Now().Format("2006-01-02T15:04:05"))
	fmt.Println("insert statement: ", insertMessageSql)
	_, err := h.dbConnection.Exec(insertMessageSql)
	return err
}

func getUserChannels(h *Hub, username string) []string /*, error*/ {
	res := make([]string, 0)
	// XXX
	// ChannelType == 0 means public channel, available to everyone
	selectChannelsSql := fmt.Sprintf("SELECT DISTINCT dbo.Channels.Id, FriendlyName FROM dbo.Channels, dbo.Users, dbo.ChannelMembers WHERE (Username = '%v' AND dbo.Users.Id = UserId AND dbo.Channels.Id = ChannelId) OR ChannelType = 0", username)
	fmt.Println("select statement: ", selectChannelsSql)
	rows, err := h.dbConnection.Query(selectChannelsSql)
	if err != nil {
		log.Println(err)
		return res
	}
	for rows.Next() {
		var id int
		var channel string
		err = rows.Scan(&id, &channel)
		if err != nil {
			fmt.Println(err)
			return res
		}
		res = append(res, channel)
	}
	return res
}

func createNewChannel(h *Hub, username string, channel string) error {
	ownerId := usernameToId(h, username)
	// XXX
	// 0 - public channel
	insertMessageSql := fmt.Sprintf("INSERT INTO dbo.Channels VALUES('%v', '%v', '%v', '%v')",
		channel, 0, ownerId, "")
	fmt.Println("insert statement: ", insertMessageSql)
	_, err := h.dbConnection.Exec(insertMessageSql)
	return err
}

func archiveExistingChannel(h *Hub, username string, channel string) error {
	// XXX
	return errors.New("archiveExistingChannel: not implemented")
}

func getChannelMembers(h *Hub, channel string) []string {
	res := make([]string, 0)
	selectMembersSql := fmt.Sprintf("SELECT dbo.Channels.Id, FriendlyName, Username FROM dbo.Channels, dbo.Users, dbo.ChannelMembers WHERE FriendlyName = '%v' AND dbo.Users.Id = UserId AND dbo.Channels.Id = ChannelId", channel)
	fmt.Println("select statement: ", selectMembersSql)
	rows, err := h.dbConnection.Query(selectMembersSql)
	if err != nil {
		log.Println(err)
		return res
	}
	for rows.Next() {
		var id int
		var channel, user string
		err = rows.Scan(&id, &channel, &user)
		if err != nil {
			fmt.Println(err)
			return res
		}
		res = append(res, user)
	}
	return res
}

func getMessageHistory(h *Hub, channel string) []Message /*, error*/ {
	res := []Message{}
	selectMessagesHistorySql := fmt.Sprintf("SELECT Id, Username, Payload FROM dbo.Messages WHERE Recepient = '%v' ORDER BY Stamp", channel)
	fmt.Println("select statement: ", selectMessagesHistorySql)
	rows, err := h.dbConnection.Query(selectMessagesHistorySql)
	if err != nil {
		log.Println(err)
		return res
	}
	for rows.Next() {
		var id int
		var username, payload string
		err = rows.Scan(&id, &username, &payload)
		if err != nil {
			fmt.Println(err)
			return res
		}
		res = append(res, Message{Username: username, MessageType: "api.message.broadcast", Recepient: channel, Payload: payload})
	}
	return res
}

func ensureDirectChannel(h *Hub, username string, counterparty string) error {
	selectDirectChannelSql := fmt.Sprintf("SELECT dbo.Channels.Id FROM dbo.Channels, dbo.Users u, dbo.Users c, dbo.ChannelMembers um, dbo.ChannelMembers cm WHERE ChannelType = 1 AND dbo.Channels.Id = cm.ChannelId AND dbo.Channels.Id = um.ChannelId AND u.Username = '%v' AND c.Username = '%v' AND u.Id = um.UserId AND c.Id = cm.UserId", username, counterparty)
	fmt.Println("select statement: ", selectDirectChannelSql)
	res, err := h.dbConnection.Exec(selectDirectChannelSql)
	if err != nil {
		log.Println(err)
		return err
	}
	rows, er := res.RowsAffected()
	if er != nil {
		log.Println(er)
		return er
	}
	if rows > 0 {
		return nil
	}
	ownerId := usernameToId(h, username)
	channel := fmt.Sprintf("%v-%v", username, counterparty)
	// XXX
	// 1 - direct private channel
	insertMessageSql := fmt.Sprintf("INSERT INTO dbo.Channels VALUES('%v', '%v', '%v', '%v')",
		channel, 1, ownerId, "")
	fmt.Println("insert statement: ", insertMessageSql)
	_, err = h.dbConnection.Exec(insertMessageSql)
	if err != nil {
		return err
	}
	channelId := channelToId(h, channel, 1)
	insertMessageSql = fmt.Sprintf("INSERT INTO dbo.ChannelMembers VALUES('%v', '%v')",
		channelId, ownerId)
	_, err = h.dbConnection.Exec(insertMessageSql)
	if err != nil {
		return err
	}
	insertMessageSql = fmt.Sprintf("INSERT INTO dbo.ChannelMembers VALUES('%v', '%v')",
		channelId, usernameToId(h, counterparty))
	_, err = h.dbConnection.Exec(insertMessageSql)
	return err
}

func removeUserSession(h *Hub, username string) error {
	deleteUserSessionSql := fmt.Sprintf("DELETE FROM dbo.Sessions WHERE Username = '%v'", username)
	_, err := h.dbConnection.Exec(deleteUserSessionSql)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func addChannelMember(h *Hub, channel string, username string) error {
	// XXX
	channelType := 0
	err := addUserToChannel(h, channel, username, channelType)
	return err
}

func addUserToChannel(h *Hub, channel string, username string, channelType int) error {
	userId := usernameToId(h, username)
	channelId := channelToId(h, channel, channelType)
	if userId == 0 || channelId == 0 {
		return errors.New("addUserToChannel: internal error")
	}
	insertChannelMemberSql := fmt.Sprintf("INSERT INTO dbo.ChannelMembers VALUES(%v, %v)", channelId, userId)
	_, err := h.dbConnection.Exec(insertChannelMemberSql)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func usernameToId(h *Hub, username string) int {
	selectUserIdSql := fmt.Sprintf("SELECT Id FROM dbo.Users WHERE Username = '%v'", username)
	fmt.Println("select statement: ", selectUserIdSql)
	rows, err := h.dbConnection.Query(selectUserIdSql)
	if err != nil {
		log.Println(err)
		return 0
	}
	if rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			fmt.Println(err)
			return 0
		}
		return id
	}
	// found nothing, should raise an error
	return 0
}

func channelToId(h *Hub, channelName string, channelType int) int {
	selectChannelIdSql := fmt.Sprintf("SELECT Id FROM dbo.Channels WHERE FriendlyName = '%v' AND ChannelType = %v", channelName, channelType)
	fmt.Println("select statement: ", selectChannelIdSql)
	rows, err := h.dbConnection.Query(selectChannelIdSql)
	if err != nil {
		log.Println(err)
		return 0
	}
	if rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			fmt.Println(err)
			return 0
		}
		return id
	}
	// found nothing, should raise an error
	return 0
}
