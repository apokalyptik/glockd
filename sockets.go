package main

import(
	"fmt"
	"net"
	"os"
	"bufio"
	"log"
)

const (
	RECV_BUF_LEN = 1024
)

var unixConnectionCounter chan string
func mindUnixConnectionCounter() {
	var connectionCount uint64
	unixConnectionCounter = make(chan string, 1024)
	for {
		connectionCount++
		unixConnectionCounter<- fmt.Sprintf("unix:%d", connectionCount)
	}
}

func mind_socket(listener net.Listener) {
	// Loop forever
	for {
		// Got a connecting client
		conn, err := listener.Accept()
		// Maybe
		if err != nil {
			println("Error accept:", err.Error())
			continue
		}
		// Seems legit. Spawn a goroutine to handle this new client
		go socket_client(conn)
	}
}

func mind_unix() {
	if cfg_unix == "" {
		return
	}
	os.Remove(cfg_unix)
	if listener, err := net.Listen("unix", cfg_unix); err != nil {
		log.Fatalf( "UNIX SOCKET Listener Error: %+v", err );
	} else {
		os.Chmod(cfg_unix, 0766)
		go mindUnixConnectionCounter()
		mind_socket(listener)
	}
}

func mind_tcp() {
	if cfg_port == 0 {
		return
	}

	// Fire up the tcpip listening port
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg_port) )
	if err != nil {
		// Or, you know... die...
		println("error listening:", err.Error())
		os.Exit(1)
	}
	mind_socket(listener)
}

func is_valid_command( command string ) bool {
	// Just a helper function to determine if a command is valid or not.
	for _, ele := range commands {
		if ele == command {
			// valid
			return true
		}
	}
	// not
	return false
}

func socket_client(conn net.Conn) {
	my_client := conn.RemoteAddr().String()
	if rx_validate_remote_addr.MatchString(my_client) == false {
		my_client = <-unixConnectionCounter
	}
	mylocks := make(map [string] bool)
	myshared := make(map [string] bool)

	stats_channel <- stat_bump{ stat: "connections", val: 1 }

	if cfg_verbose {
		fmt.Printf( "%s connected\n", my_client )
	}
	// The following handles orphaning locks... It only runs after the 
	// for true {} loop (which means on disconnect or error which are
	// the only things that breaks it)
	defer client_disconnected( my_client, mylocks, myshared )

	// Accept commands loop
	for true {

		// Read from the client
		buf, _, err := bufio.NewReader(conn).ReadLine()
		if err != nil {
			// If we got an error just exit
			return
		}

		rsp := process_lock_client_command( lock_client_command{ buf, mylocks, myshared, my_client } )
		mylocks = rsp.mylocks
		myshared = rsp.myshared

		// Write our response back to the client
		_, _ = conn.Write( rsp.rsp );

	}
}

