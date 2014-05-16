package main

import (
	"fmt"
	"strings"
)

type lockmap map[string]struct{}

type client struct {
	ex lockmap
	sh lockmap
	me string
}

func (c *client) init(me string) {
	c.ex = make(lockmap)
	c.sh = make(lockmap)
	c.me = me
}

func (c *client) disconnect() {
	// Since the client has disconnected... we need to release all of the locks that it held
	if cfg.Verbose {
		fmt.Printf("%s disconnected\n", c.me)
	}
	for lock, _ := range c.ex {
		if cfg.Verbose {
			fmt.Printf("%s orphaned lock %s\n", c.me, lock)
		}
		lock_req(lock, -1, false, c.me)
		stats_channel <- stat_bump{stat: "orphans", val: 1}
	}
	// We also need to release all the shared locks that it held
	for lock, _ := range c.sh {
		if cfg.Verbose {
			fmt.Printf("%s orphaned shared lock %s\n", c.me, lock)
		}
		lock_req(lock, -1, true, c.me)
		stats_channel <- stat_bump{stat: "shared_orphans", val: 1}
	}
	registrar <- registration_request{client: c.me}
	stats_channel <- stat_bump{stat: "connections", val: -1}
	// Nothing left to do... That's all the client had...
}

func (c *client) doMe() []byte {
	iam := make(chan string, 1)
	registrar <- registration_request{client: c.me, reply: iam}
	return []byte(fmt.Sprintf("1 %s %s\n", c.me, <-iam))
}

func (c *client) doIam(name string) []byte {
	if cfg.Registry == true {
		registrar <- registration_request{client: c.me, name: name}
		if cfg.Verbose {
			fmt.Printf("%s changed their name to '%s'\n", c.me, name)
		}
		return []byte("1 ok\n")
	} else {
		return []byte("0 disabled\n")
	}
}

func (c *client) doWho(who string) []byte {
	rsp := []byte("")
	if cfg.Dump == true && cfg.Registry == true {
		rc := make(chan map[string]string)
		registrar <- registration_request{dump: rc}
		registry := <-rc
		for idx, val := range registry {
			if who == "" || who == val {
				rsp = []byte(string(rsp) + fmt.Sprintf("%s: %s\n", idx, val))
			}
		}
		close(rc)
		return rsp
	} else {
		return []byte("0 disabled\n")
	}
}

func (c *client) doStats() []byte {
	// loop over stats and generated a response
	rsp := []byte("")
	for _, idx := range stat_keys() {
		switch idx {
		case "locks":
			rsp = []byte(string(rsp) + fmt.Sprintf("%s: %d\n", idx, len(locks)))
			continue
		case "shared_locks":
			rsp = []byte(string(rsp) + fmt.Sprintf("%s: %d\n", idx, len(shared_locks)))
			continue
		}
		rsp = []byte(string(rsp) + fmt.Sprintf("%s: %d\n", idx, stats[idx]))
	}
	return rsp
}

func (c *client) doInspect(lock string) []byte {
	// does the lock exist locally?
	_, present := c.ex[lock]
	if present {
		// if we have the lock, don't bother the lock goroutine
		return []byte(fmt.Sprintf("1 Lock Is Locked: %s\n", lock))
	} else {
		// otherwise check the canonical source
		rsp, _ := lock_req(lock, 0, false, c.me)
		return rsp
	}
}

func (c *client) doGet(lock string) []byte {
	// does the lock exist locally?
	_, present := c.ex[lock]
	if present {
		// if we have the lock then the answer is always "got it"
		return []byte(fmt.Sprintf("1 Lock Get Success: %s\n", lock))
	} else {
		// otherwise request it from the canonical goroutine
		rsp, val := lock_req(lock, 1, false, c.me)
		if val == "1" {
			c.ex[lock] = struct{}{}
		}
		return rsp
	}
}

func (c *client) doRelease(lock string) []byte {
	// does the lock exist locally?
	_, present := c.ex[lock]
	if present {
		// We only request the lock release if it exists locally,
		// otherwise we have no permissions to unlock it
		rsp, val := lock_req(lock, -1, false, c.me)
		if val == "1" {
			// if we released the lock successfully then purge it
			// from this goroutines map.
			delete(c.ex, lock)
		}
		return rsp
	} else {
		return []byte(fmt.Sprintf("0 Lock Release Failure: %s\n", lock))
	}
}

func (c *client) doSharedInspect(lock string) []byte {
	// Since we always want an "up to date" and accurate count
	// (not just a boolean true/false like exclusive locks)
	// Always pass this through to the canonical source
	rsp, _ := lock_req(lock, 0, true, c.me)
	return rsp
}

func (c *client) doSharedGet(lock string) []byte {
	rsp, val := lock_req(lock, 1, true, c.me)
	if val != "0" {
		// Since we now have this lock... add it to the goroutine
		// lock map.  Used for orphaning
		c.sh[lock] = struct{}{}
	}
	return rsp
}

func (c *client) doSharedRelease(lock string) []byte {
	rsp, val := lock_req(lock, -1, true, c.me)
	if val == "1" {
		// Since we now no longer have this lock... remove it from
		// the goroutine lock map. No need to orphan it any longer
		delete(c.sh, lock)
	}
	return rsp
}

func (c *client) doDump(what string) []byte {
	if cfg.Dump == true {
		rsp := []byte("")
		// loop over all the locks
		for idx, val := range locks {
			// if we want all locks, or this specific lock matches the lock we
			// want then add it to the response output
			c := make(chan string, 1)
			if what == "" || what == idx {
				registrar <- registration_request{client: val, reply: c}
				val = <-c
				rsp = []byte(string(rsp) + fmt.Sprintf("%s: %s\n", idx, val))
			}
			close(c)
		}
		return rsp
	} else {
		return []byte("0 disabled\n")
	}
}

func (c *client) doSharedDump(what string) []byte {
	if cfg.Dump == true {
		rsp := []byte("")
		// loop over all the locks
		for idx, val := range shared_locks {
			// if we want all locks, or this specific lock matches the lock we
			// want then add it to the response output
			if what == "" || what == idx {
				c := make(chan string, 1)
				for _, locker := range val {
					registrar <- registration_request{client: locker, reply: c}
					locker = <-c
					rsp = []byte(string(rsp) + fmt.Sprintf("%s: %s\n", idx, locker))
				}
				close(c)
			}
		}
		return rsp
	} else {
		return []byte("0 disabled\n")
	}

}

func (c *client) doFullDump(what string) []byte {
	if cfg.Dump == true {
		if what == "shared" {
			return []byte(fmt.Sprintf("%v\n", shared_locks))
		} else {
			return []byte(fmt.Sprintf("%v\n", locks))
		}
	} else {
		return []byte("0 disabled\n")
	}
}

func (c *client) command(input []byte) []byte {
	// Lots of variables local to this goroutine. Because: reasons
	var command []string
	var what string

	command = strings.SplitN(strings.TrimSpace(string(input)), " ", 2)
	if false == is_valid_command(command[0]) {
		stats_channel <- stat_bump{stat: "invalid_commands", val: 1}
		if cfg.Verbose {
			fmt.Printf("%s invalid command '%s'\n", c.me, strings.Trim(string(input), string(0)))
		}
		// if we got an invalid command, skip it
		return []byte("")
	}

	// Always bump the command stats
	stats_channel <- stat_bump{stat: "command_" + command[0], val: 1}

	// We always want a lock, even if the lock is ""
	if len(command) == 1 {
		command = append(command, "")
	}

	// Nothing sane about assuming sanity
	what = strings.Join(strings.Fields(command[1]), " ")

	// Actually deal with the command now...
	switch command[0] {
	case "me":
		return c.doMe()
	case "iam":
		return c.doIam(what)
	case "who":
		return c.doWho(what)
	case "q":
		return c.doStats()
	case "i":
		return c.doInspect(what)
	case "g":
		return c.doGet(what)
	case "r":
		return c.doRelease(what)
	case "si":
		return c.doSharedInspect(what)
	case "sg":
		return c.doSharedGet(what)
	case "sr":
		return c.doSharedRelease(what)
	case "d":
		return c.doDump(what)
	case "sd":
		return c.doSharedDump(what)
	case "dump":
		return c.doFullDump(what)
	}
	return []byte("")
}

// valid command list
var commands = []string{"me", "iam", "who", "d", "sd", "i", "si", "g", "sg", "r", "sr", "q", "dump"}

func lock_req(lock string, action int, shared bool, my_client string) ([]byte, string) {
	// Create a channel on which the lock or shared lock goroutine can contact us back on
	var reply_channel = make(chan lock_reply, 1)
	// Send a non-blocking message to the proper goroutine about what we want
	if shared {
		shared_lock_channel <- lock_request{lock: lock, action: action, reply: reply_channel, client: my_client}
	} else {
		lock_channel <- lock_request{lock: lock, action: action, reply: reply_channel, client: my_client}
	}
	// Block until we recieve a reply
	rsp := <-reply_channel
	// Format and return our response
	var response = []byte(rsp.response)
	var terse = string(response[0])

	if cfg.Verbose && terse != "0" {
		var display string
		if shared {
			display = "shared lock"
		} else {
			display = "lock"
		}
		switch action {
		case 1:
			fmt.Printf("%s obtained %s for %s\n", my_client, display, lock)
		case -1:
			fmt.Printf("%s released %s for %s\n", my_client, display, lock)
		}
	}

	return response, terse
}
