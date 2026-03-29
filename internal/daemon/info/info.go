package info

import (
	"sync"
)

type JailState interface {
	InJail(pid int) bool
	SetJail(pid int)
	DeleteFromJail(pid int)
	PIDs() []int
}

type API struct {
	mu   sync.Mutex
	jail map[int]bool
}

func NewAPI() *API {
	return &API{
		mu:   sync.Mutex{},
		jail: make(map[int]bool),
	}
}

func (a *API) InJail(pid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if find, ok := a.jail[pid]; !ok || !find {
		return false
	}

	return true
}

func (a *API) SetJail(pid int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.jail[pid] = true
}

func (a *API) DeleteFromJail(pid int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.jail, pid)
}

func (a *API) PIDs() []int {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]int, 0, len(a.jail))
	for pid := range a.jail {
		out = append(out, pid)
	}

	return out
}
