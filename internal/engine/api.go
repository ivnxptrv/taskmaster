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

func (m *Manager) Status(name string, index int) ProcInfo {
	reply := make(chan ProcInfo)
	m.events <- Status{Name: name, Index: index, Reply: reply}
	return <-reply
}

func (m *Manager) Start(name string, index int) error {
	reply := make(chan struct{})
	m.events <- Start{Name: name, Index: index, Reply: reply}
	<-reply
	return nil
}

func (m *Manager) Stop(name string, index int) error {
	reply := make(chan struct{})
	m.events <- Stop{Name: name, Index: index, Reply: reply}
	<-reply
	return nil
}

func (m *Manager) Restart(name string, index int) error {
	reply := make(chan struct{})
	m.events <- Restart{Name: name, Index: index, Reply: reply}
	<-reply
	return nil
}

func (m *Manager) Reload() error {
	return nil
}

func (m *Manager) Shutdown() error {
	reply := make(chan struct{})
	m.events <- Shutdown{Reply: reply}
	<-reply
	return nil
}

// -----------------------------------------------
