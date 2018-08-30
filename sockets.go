package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
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
		unixConnectionCounter <- fmt.Sprintf("unix:%d", connectionCount)
	}
}

func mind_socket_accept(listener interface{}) chan net.Conn {
	connections := make(chan net.Conn, 1024)
	go func(l interface{}, connections chan net.Conn) {
		var listenerType string
		var listener net.Listener
		switch l.(type) {
		default:
			log.Printf("%#v", l)
			return
		case *net.UnixListener:
			listener = l.(net.Listener)
			listenerType = "unix"
		case *net.TCPListener:
			if cfg.SSLCfg != nil {
				ls := tls.NewListener(l.(net.Listener), cfg.SSLCfg)
				listener = ls
				listenerType = "tls"
			} else {
				listener = l.(net.Listener)
				listenerType = "tcp"
			}
		}
		for {
			conn, err := listener.Accept()
			// Got a connecting client Maybe
			if err != nil {
				println("Error accept:", err.Error())
				continue
			}
			// Seems legit. Spawn a goroutine to handle this new client
			switch listenerType {
			case "tcp":
				thisConn := conn.(interface{}).(*net.TCPConn)
				if err := thisConn.SetKeepAlive(true); err != nil {
					log.Printf("Error setting keepalive on %s: %s", thisConn.RemoteAddr().String(), err.Error())
				}
				connections <- thisConn
			case "tls":
				connections <- conn
			default:
				thisConn := conn.(interface{}).(*net.UnixConn)
				connections <- thisConn
			}
		}
	}(listener, connections)
	return connections
}

func mind_socket(listener net.Listener) {
	connections := mind_socket_accept(listener)
	// Loop forever
	for {
		select {
		case conn := <-connections:
			go socket_client(conn)
		}
	}
}

func mind_unix() {
	if cfg.Unix == "" {
		return
	}
	os.Remove(cfg.Unix)
	if listener, err := net.Listen("unix", cfg.Unix); err != nil {
		log.Fatalf("UNIX SOCKET Listener Error: %+v", err)
	} else {
		os.Chmod(cfg.Unix, 0766)
		go mindUnixConnectionCounter()
		mind_socket(listener)
	}
}

func mind_tcp() {
	if cfg.Port == 0 {
		return
	}
	// Fire up the tcpip listening port
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.Port))
	if err != nil {
		// Or, you know... die...
		println("error listening:", err.Error())
		os.Exit(1)
	}
	mind_socket(listener)
}

func is_valid_command(command string) bool {
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
	client := new(client)
	client.init(conn.RemoteAddr().String())
	if rx_validate_remote_addr.MatchString(client.me) == false {
		client.me = <-unixConnectionCounter
	}
	stats_channel <- stat_bump{stat: "connections", val: 1}

	if cfg.Verbose {
		fmt.Printf("%s connected\n", client.me)
	}
	// The following handles orphaning locks... It only runs after the
	// for true {} loop (which means on disconnect or error which are
	// the only things that breaks it)
	defer client.disconnect()

	// Accept commands loop
	for true {
		// Read from the client
		buf, _, err := bufio.NewReader(conn).ReadLine()
		if err != nil {
			// If we got an error just exit
			return
		}
		conn.Write(client.command(buf))
	}
}
