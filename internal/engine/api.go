package engine

import (
	"time"
)

// -----------------------------------------------

type ProcInfo struct {
	name      string
	index     int
	state     string
	pid       int
	startedAt time.Time
}

func (m *Manager) Status(name string) ProcInfo {
	// chanResponse := make(chan string)
	// d.events <- Status{Name: name, chanResponse: chanResponse}
	// return <-chanResponse

	// get real info
	// or send event with chanel back to get result
	return ProcInfo{}
}

func (m *Manager) Start(name string) error {
	// m.submitEvent(Start{Name: name})
	return nil
}

func (m *Manager) Stop(name string) error {
	// m.submitEvent(Stop{Name: name})
	return nil
}

func (m *Manager) Restart(name string) error {
	return nil
}

func (m *Manager) Reload() error {
	return nil
}

func (m *Manager) Shutdown() error {
	// m.submitEvent(Shutdown{})
	return nil
}

// -----------------------------------------------
