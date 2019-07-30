/******************************************************************************
* The MIT License
* Copyright (c) 2019 Igor Shmukler  github.com/igorshmukler
* 
* Permission is hereby granted, free of charge, to any person obtaining  a copy
* of this software and associated documentation files (the Software), to deal
* in the Software without restriction, including  without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell 
* copies of the Software, and to  permit persons to whom the Software is 
* furnished to do so, subject to the following conditions:
* 
* The above copyright notice and this permission notice shall be included in 
* all copies or substantial portions of the Software.
* 
* THE SOFTWARE IS PROVIDED AS IS, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR 
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, 
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER 
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
*******************************************************************************/
//
// main.go
//
// Author:
//   Igor Shmukler (igor.shmukler (AT) gmail.com)
//

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
