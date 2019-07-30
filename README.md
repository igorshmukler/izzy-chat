## Golang chat server

This was developed as an excercise in building network servers in Golang. The server upgrades HTTP connection
to the WebSockets protocol, and all further communcations are done using a binary protocol with payload being
sent as ArrayBuffer. The server is conceptually similar to Slack/HipChat -like servers. It supports private
and public channels/rooms, direct communications between users, keeps messages history and more. Currently, server supports MS SQL Server as the store.

Covered by MIT license.

### TODO

- Switch from `golang.org/x/net/websocket` to `github.com/gobwas/ws` to significantly lower memory overhead
- Make server settings configurable
- Convert queries to stored procedures
- Support multiple databases, starting with Postgres
