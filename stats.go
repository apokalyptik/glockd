package main

import (
	"sort"
)

// Structure for bumping a stat
type stat_bump struct {
	stat string
	val  int
}

// stats bump channel and data structure
var stats_channel = make(chan stat_bump, 4096)
var stats = map[string]int{
	"command_d":        0,
	"command_sd":       0,
	"command_i":        0,
	"command_si":       0,
	"command_g":        0,
	"command_sg":       0,
	"command_r":        0,
	"command_sr":       0,
	"command_q":        0,
	"command_dump":     0,
	"connections":      0,
	"locks":            0,
	"shared_locks":     0,
	"orphans":          0,
	"shared_orphans":   0,
	"invalid_commands": 0,
}

func mind_stats() {
	// This function produces no output, it simply mutates the state
	for true {
		// Block this specific goroutine until we have a message incoming about a stats bump
		bump := <-stats_channel
		// Bump that stat
		stats[bump.stat] += bump.val
	}
}

func stat_keys() []string {
	mk := make([]string, len(stats))
	i := 0
	for k, _ := range stats {
		mk[i] = k
		i++
	}
	sort.Strings(mk)
	return mk
}
