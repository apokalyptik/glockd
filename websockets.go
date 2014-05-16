package main

import (
	"fmt"
	"log"
	"net/http"

	"code.google.com/p/go.net/websocket"
)

func websockets_client(conn *websocket.Conn) {
	client := new(client)
	client.init(conn.Request().RemoteAddr)
	defer conn.Close()
	defer client.disconnect()

	stats_channel <- stat_bump{stat: "connections", val: 1}
	if cfg.Verbose {
		fmt.Printf("%s connected\n", client.me)
	}

	for {
		var input []byte
		err := websocket.Message.Receive(conn, &input)
		if err != nil {
			return
		}
		_, err = conn.Write(client.command(input))
		if err != nil {
			return
		}
	}
}

func mind_websockets() {
	if cfg.Ws == 0 {
		return
	}
	http.Handle("/", websocket.Handler(websockets_client))
	err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Ws), nil)
	if err != nil {
		log.Fatalf("HTTP Listening Error: %+v", err)
	}
}
