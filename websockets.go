package main

import(
	"github.com/garyburd/go-websocket/websocket"
	"net/http"
	"io/ioutil"
	"os"
	"fmt"
)

func websockets_client(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r.Header, nil, 1024, 1024)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mylocks := make(map [string] bool)
	myshared := make(map [string] bool)
	my_client := r.RemoteAddr

	defer conn.Close()
	defer client_disconnected( my_client, mylocks, myshared )

	stats_channel <- stat_bump{ stat: "connections", val: 1 }
	if cfg_verbose {
		fmt.Printf( "%s connected\n", my_client )
	}

	for {
		op, r, err := conn.NextReader()
		if err != nil {
			return
		}
		if op != websocket.OpBinary && op != websocket.OpText {
			continue
		}

		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return
		}

		rsp := process_lock_client_command( lock_client_command{ buf, mylocks, myshared, my_client } )
		mylocks = rsp.mylocks
		myshared = rsp.myshared

		conn.WriteMessage(op, rsp.rsp)
	}
}

func mind_websockets() {
	if cfg_ws == 0 {
		return
	}
	http.HandleFunc("/", websockets_client)
	err := http.ListenAndServe( fmt.Sprintf(":%d", cfg_ws), nil)
	if err != nil {
		fmt.Printf( "HTTP Listening Error: %+v", err );
		os.Exit(1)
	}
}

