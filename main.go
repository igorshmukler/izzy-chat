package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	mssql "github.com/denisenkom/go-mssqldb"
	"golang.org/x/net/websocket"
)

var (
	port = flag.String("port", "8080", "port used for ws connection")
)

func server(port string) error {
	mux := http.NewServeMux()

	connector, err := mssql.NewConnector("server=mssql.site.net;user id=izzy;password=P@55w0rd;database=izzy;")
	if err != nil {
		log.Panic("Error opening db:", err.Error())
	}
	condb := sql.OpenDB(connector)
	fmt.Println("opened database connection.")

	h := newHub(condb)
	mux.Handle("/", websocket.Handler(func(ws *websocket.Conn) {
		handler(ws, h)
	}))

	s := http.Server{Addr: ":" + port, Handler: mux}
	return s.ListenAndServe()
}

func main() {
	flag.Parse()
	log.Fatal(server(*port))
}
