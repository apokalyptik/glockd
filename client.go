package main

import(
	"fmt"
	"strings"
)

type lock_client_response struct {
	rsp []byte
	mylocks map [string] bool
	myshared map [string] bool
}

type lock_client_command struct {
	input []byte
	mylocks map [string] bool
	myshared map [string] bool
	my_client string
}

// valid command list
var commands = []string { "me", "iam", "who", "d", "sd","i", "si", "g", "sg", "r", "sr", "q", "dump" }

func client_disconnected(my_client string, mylocks map[string] bool, myshared map[string] bool) {
	// Since the client has disconnected... we need to release all of the locks that it held
	if cfg_verbose {
		fmt.Printf( "%s disconnected\n", my_client )
	}
	for lock, _ := range mylocks {
		if ( cfg_verbose ) {
			fmt.Printf( "%s orphaned lock %s\n", my_client, lock )
		}
		lock_req(lock, -1, false, my_client)
		stats_channel <- stat_bump{ stat: "orphans", val: 1 }
	}
	// We also need to release all the shared locks that it held
	for lock, _ := range myshared {
		if ( cfg_verbose ) {
			fmt.Printf( "%s orphaned shared lock %s\n", my_client, lock )
		}
		lock_req(lock, -1, true, my_client)
		stats_channel <- stat_bump{ stat: "shared_orphans", val: 1 }
	}
	registrar<- registration_request{ client: my_client }
	stats_channel <- stat_bump{ stat: "connections", val: -1 }
	// Nothing left to do... That's all the client had...
}

func lock_req(lock string, action int, shared bool, my_client string) ( []byte, string ) {
	// Create a channel on which the lock or shared lock goroutine can contact us back on
	var reply_channel = make(chan lock_reply, 1)
	// Send a non-blocking message to the proper goroutine about what we want
	if shared {
		shared_lock_channel <- lock_request{ lock:lock, action:action, reply:reply_channel, client:my_client }
	} else {
		lock_channel <- lock_request{ lock:lock, action:action, reply:reply_channel, client:my_client }
	}
	// Block until we recieve a reply
	rsp := <-reply_channel
	// Format and return our response
	var response = []byte(rsp.response)
	var terse = string(response[0])

	if cfg_verbose && terse != "0" {
		var display string
		if shared {
			display = "shared lock"
		} else {
			display = "lock"
		}
		switch action {
			case 1:
				fmt.Printf( "%s obtained %s for %s\n", my_client, display, lock )
			case -1:
				fmt.Printf( "%s released %s for %s\n", my_client, display, lock )
		}
	}

	return response, terse
}

func process_lock_client_command( c lock_client_command ) lock_client_response {
	// Lots of variables local to this goroutine. Because: reasons
	var command []string
	var rsp []byte
	var val string
	var lock string

	command = strings.SplitN( strings.TrimSpace(string(c.input)), " ", 2 )
	if false == is_valid_command(command[0]) {
		stats_channel <- stat_bump{ stat: "invalid_commands", val: 1 }
		if cfg_verbose {
			fmt.Printf( "%s invalid command '%s'\n", c.my_client, strings.Trim( string(c.input), string(0) ) )
		}
		// if we got an invalid command, skip it
		return lock_client_response{ []byte(""), c.mylocks, c.myshared }
	}

	// Always bump the command stats
	stats_channel <- stat_bump{ stat: "command_"+command[0], val: 1 }

	// We always want a lock, even if the lock is ""
	if len(command) == 1 {
		command  = append(command, "")
	}

	// Nothing sane about assuming sanity
	lock = strings.Join( strings.Fields( command[1] ), " ")

	// Actually deal with the command now...
	switch command[0] {
		case "me":
			iam := make(chan string, 1)
			registrar<- registration_request{ client: c.my_client, reply: iam }
			rsp = []byte(fmt.Sprintf("1 %s %s\n", c.my_client, <-iam))
		case "iam":
			if cfg_registry == true {
				registrar<- registration_request{ client: c.my_client, name: lock }
				if cfg_verbose {
					fmt.Printf( "%s changed their name to '%s'\n", c.my_client, lock )
				}
				rsp = []byte("1 ok\n")
			} else {
				rsp = []byte("0 disabled\n")
			}
		case "who":
			if cfg_dump == true && cfg_registry == true {
				c := make( chan map[string] string )
				registrar<- registration_request{ dump: c }
				registry := <-c
				for idx, val := range registry {
					if lock == "" || lock == val {
						rsp = []byte( string(rsp) + fmt.Sprintf("%s: %s\n", idx, val))
					}
				}
				close(c)
			} else {
				rsp = []byte("0 disabled\n")
			}
		case "q":
			// loop over stats and generated a response
			rsp = []byte("")
			for _, idx := range stat_keys() {
				switch idx {
					case "locks":
						rsp = []byte( string(rsp) + fmt.Sprintf("%s: %d\n", idx, len(locks)) )
						continue
					case "shared_locks":
						rsp = []byte( string(rsp) + fmt.Sprintf("%s: %d\n", idx, len(shared_locks)) )
						continue
				}
				rsp = []byte( string(rsp) + fmt.Sprintf("%s: %d\n", idx, stats[idx]) )
			}
		case "i":
			// does the lock exist locally?
			_, present := c.mylocks[lock]
			if present {
				// if we have the lock, don't bother the lock goroutine
				rsp = []byte(fmt.Sprintf("1 Lock Is Locked: %s\n", lock))
			} else {
				// otherwise check the canonical source
				rsp, _ = lock_req( lock, 0, false, c.my_client )
			}
		case "g":
			// does the lock exist locally?
			_, present := c.mylocks[lock]
			if present {
				// if we have the lock then the answer is always "got it"
				rsp = []byte(fmt.Sprintf("1 Lock Get Success: %s\n", lock))
			} else {
				// otherwise request it from the canonical goroutine
				rsp, val = lock_req( lock, 1, false, c.my_client )
				if val == "1" {
					c.mylocks[lock] = true
				}
			}
		case "r":
			// does the lock exist locally?
			_, present := c.mylocks[lock]
			if present {
				// We only request the lock release if it exists locally, 
				// otherwise we have no permissions to unlock it
				rsp, val = lock_req( lock, -1, false, c.my_client )
				if val == "1" {
					// if we released the lock successfully then purge it 
					// from this goroutines map.
					delete(c.mylocks, lock )
				}
			} else {
				rsp = []byte(fmt.Sprintf("0 Lock Release Failure: %s\n", lock))
			}
		case "si":
			// Since we always want an "up to date" and accurate count
			// (not just a boolean true/false like exclusive locks)
			// Always pass this through to the canonical source
			rsp, val = lock_req( lock, 0, true, c.my_client )
		case "sg":
			rsp, val = lock_req( lock, 1, true, c.my_client )
			if val != "0" {
				// Since we now have this lock... add it to the goroutine
				// lock map.  Used for orphaning
				c.myshared[lock] = true
			}
		case "sr":
			rsp, val = lock_req( lock, -1, true, c.my_client )
			if val == "1" {
				// Since we now no longer have this lock... remove it from
				// the goroutine lock map. No need to orphan it any longer
				delete(c.myshared, lock )
			}
		case "d":
			if cfg_dump == true {
				rsp = []byte("")
				// loop over all the locks
				for idx, val := range locks {
					// if we want all locks, or this specific lock matches the lock we 
					// want then add it to the response output
					c := make(chan string, 1)
					if lock == "" || lock == idx {
						registrar<- registration_request{ client: val, reply: c }
						val = <-c
						rsp = []byte( string(rsp) + fmt.Sprintf("%s: %s\n", idx, val))
					}
					close(c)
				}
			} else {
				rsp = []byte("0 disabled\n")
			}
		case "sd":
			if cfg_dump == true {
				rsp = []byte("")
				// loop over all the locks
				for idx, val := range shared_locks {
					// if we want all locks, or this specific lock matches the lock we
					// want then add it to the response output
					if lock == "" || lock == idx {
						c := make(chan string, 1)
						for _, locker := range val {
							registrar<- registration_request{ client: locker, reply: c }
							locker = <-c
							rsp = []byte( string(rsp) + fmt.Sprintf("%s: %s\n", idx, locker))
						}
						close(c)
					}
				}
			} else {
				rsp = []byte("0 disabled\n")
			}
		case "dump":
			if cfg_dump == true {
				if lock == "shared" {
					rsp = []byte( fmt.Sprintf("%v\n", shared_locks) )
				} else {
					rsp = []byte( fmt.Sprintf("%v\n", locks) )
				}
			} else {
				rsp = []byte("0 disabled\n")
			}
	}

	return lock_client_response{ rsp, c.mylocks, c.myshared }
}

