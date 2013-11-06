package main

import(
	"os"
	"fmt"
	"flag"
	"syscall"
	"runtime"
	"regexp"
)

// Structure for requesting a lock with
type lock_request struct {
	lock string
	action int
	reply chan lock_reply
	client string
}

// Structure for a response generated during a lock request
type lock_reply struct {
	lock string
	response string
}

var cfg_port int
var cfg_pidfile string
var cfg_verbose bool
var cfg_ws int
var cfg_registry bool
var cfg_dump bool
var cfg_unix string

var rx_validate_remote_addr *regexp.Regexp

func main() {
	runtime.GOMAXPROCS( runtime.NumCPU() )

	flag.IntVar(&cfg_port, "port", 9999, "Listen on the following TCP ws. 0 Disables.")
	flag.IntVar(&cfg_ws, "ws", 9998, "Listen on the following TCP Port. 0 Disables.")
	flag.StringVar(&cfg_pidfile, "pidfile", "", "pidfile to use (required)")
	flag.StringVar(&cfg_unix, "unix", "", "Filesystem path to the unix socket to listen on.  '' Disables.")
	flag.BoolVar(&cfg_registry, "registry", true, "allow use of the registry.");
	flag.BoolVar(&cfg_dump, "dump", true, "Allow use of the dump, d, and sd commands.")
	flag.BoolVar(&cfg_verbose, "verbose", false, "be verbose about what's going on.");

	flag.Parse()

	if cfg_verbose == true {
		fmt.Printf( "cfg_port:     %+v\n", cfg_port )
		fmt.Printf( "cfg_ws:       %+v\n", cfg_ws )
		fmt.Printf( "cfg_pidfile:  %+v\n", cfg_pidfile )
		fmt.Printf( "cfg_unix:     %+v\n", cfg_unix )
		fmt.Printf( "cfg_registry: %+v\n", cfg_registry )
		fmt.Printf( "cfg_dump:     %+v\n", cfg_dump )
		fmt.Printf( "cfg_verbose:  %+v\n", cfg_verbose )
	}

	if cfg_pidfile == "" {
		println( "Please specify a pidfile" )
		os.Exit(2)
	}

	rx_validate_remote_addr = regexp.MustCompile(":\\d+$")

	pidfile, err1 := os.OpenFile(cfg_pidfile, os.O_CREATE | os.O_RDWR, 0666)
	err2 := syscall.Flock(int(pidfile.Fd()), syscall.LOCK_NB | syscall.LOCK_EX)
	if err1 != nil {
		fmt.Printf( "Error opening pidfile: %s: %v\n", cfg_pidfile, err1 )
		os.Exit(3)
	}
	if err2 != nil {
		fmt.Printf( "Error locking  pidfile: %s: %v\n", cfg_pidfile, err2 )
		os.Exit(4)
	}
	syscall.Ftruncate( int(pidfile.Fd()), 0 )
	syscall.Write( int(pidfile.Fd()), []byte(fmt.Sprintf( "%d", os.Getpid())) )

	// Spawn a goroutine for stats
	go mind_stats()
	// Spawn a goroutine for locks
	go mind_locks()
	// Spawn a goroutine for shared locks
	go mind_shared_locks()
	// Spawn a goroutine for the websockets interface
	go mind_websockets()
	// Spawn a goroutine for accepting and handling incoming unix socket connections
	go mind_unix()
	// Spawn a goroutine for accepting and handling incoming tcp connections
	go mind_tcp()

	if cfg_registry == true {
		// Spawn a goroutine for minding the user registry
		go mind_registry()
	}
	// Block indefinitely
	wait := make(chan bool)
	<-wait
}

