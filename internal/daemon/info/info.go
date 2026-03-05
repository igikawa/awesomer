package info

import "sync"

func init() {
	mu = sync.Mutex{}
	jail = make(map[int]bool)
}

var mu sync.Mutex
var jail map[int]bool

func InJail(pid int) bool {
	mu.Lock()
	defer mu.Unlock()

	if find, ok := jail[pid]; !ok || !find {
		return false
	}

	return true
}

func SetJail(pid int) {
	mu.Lock()
	defer mu.Unlock()

	jail[pid] = true
}

func DeleteFromJail(pid int) {
	mu.Lock()
	defer mu.Unlock()

	delete(jail, pid)
}
