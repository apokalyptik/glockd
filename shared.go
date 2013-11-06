package main

import(
	"fmt"
)

// Shared locks request channel and data structure
var shared_lock_channel = make(chan lock_request, 8192)
var shared_locks = map[string] []string {}

func shared_locks_unset( lock string, index int ) {
	// This function exists to make the following hack (shamelessly stolen from https://code.google.com/p/go-wiki/wiki/SliceTricks)
	// readable since it was insanely long inline and indented with longer variable names...
	shared_locks[lock] = shared_locks[lock][:index+copy(shared_locks[lock][index:], shared_locks[lock][index+1:])]
}

func mind_shared_locks() {
	var client_present int
	var response string
	for true {
		// Block this specific goroutine until we get a request
		req := <-shared_lock_channel
		// Reset the state for client_present (the index of the 
		// client in the slice in the map for the shared lock)
		client_present = -1
		// fine out whether this lock even exists. This information
		// is used in essentially everything else we do here
		_, present := shared_locks[req.lock]
		// Reset our response variable. State flushing
		response = ""
		if present {
			// We only want to find the client_present index (if
			// any) in the lock slice is the lock slice exists :)
			for k, v := range shared_locks[req.lock] {
				if v == req.client {
					client_present = k
					break;
				}
			}
		}
		switch req.action {
			case -1:
				// Client wants to release a shared lock
				if present && client_present != -1 {
					// Since the lock exists and client_present is 
					// not -1 (which would be not present) we can 
					// Remove the client from the lock slice
					shared_locks_unset( req.lock, client_present )
					if len(shared_locks[req.lock]) == 0 {
						// Since the lock slice is now empty we can
						// remove the slice from the lock map
						delete(shared_locks, req.lock)
					}
					response = fmt.Sprintf("1 Shared Lock Release Success: %s\r\n", req.lock)
				} else {
					// But we can't because we have no such lock
					response = fmt.Sprintf("0 Shared Lock Release Failure: %s\r\n", req.lock)
				}
			case 0:
				// Client wants info about a lock
				if present {
					// Locked, give 'em a number
					response = fmt.Sprintf("%d Shared Lock Is Locked: %s\r\n", len(shared_locks[req.lock]), req.lock)
				} else {
					// Not locked. 0 because: sanity
					response = fmt.Sprintf( "0 Shared Lock Not Locked: %s\r\n", req.lock)
				}
			case 1:
				// Client wants to lock something
				if present {
					// This lock exists in the lock map
					if client_present == -1 {
						// And the client doesnt exist in the slice, add 'em in
						shared_locks[req.lock] = append( shared_locks[req.lock], req.client )
					}
				} else {
					// This lock doesnt exist in the lock map so create
					// it with a new slice containing this client
					shared_locks[req.lock] = []string{ req.client }
				}
				// This always works... So we just need to return a count
				response = fmt.Sprintf("%d Shared Lock Get Success: %s\r\n", len(shared_locks[req.lock]), req.lock)
		}
		// Reply back to the client on the channel it provided us with in the request
		req.reply <- lock_reply{ lock: req.lock, response: response }
	}
}

