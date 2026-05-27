package engine

import (
	"os/exec"
	"syscall"
	"taskmaster/internal/fileutil"
	"time"
)

type SpawnRequest struct {
	process    *Process
	eventsChan chan event
	resultChan chan error
}

type Spawner struct {
	requestQueue chan SpawnRequest
}

func NewSpawner() *Spawner {
	s := &Spawner{
		requestQueue: make(chan SpawnRequest, 10),
	}
	// start listener for spawn commands
	go s.listen()
	return s
}

func (s *Spawner) listen() {
	// process each new spawn event
	for req := range s.requestQueue {
		req.resultChan <- s.processRequest(req)
	}

}

func (s *Spawner) processRequest(req SpawnRequest) error {
	p := req.process
	prg := p.parent
	cmd := createCmd(prg)

	// apply umask and spawn
	oldUmask := syscall.Umask(int(prg.umask))
	err := cmd.Start()
	syscall.Umask(oldUmask)
	if err != nil {
		return err
	}

	p.state = Starting
	p.cmd = cmd
	now := time.Now()
	p.startedAt = &now

	// spawn listener for starttime
	p.startTimer = time.AfterFunc(prg.starttime, func() {
		req.eventsChan <- StartTimerFired{proc: p}
	})

	// spawn listener for child exit
	go func() {
		err := cmd.Wait()
		req.eventsChan <- ProcExited{proc: p, err: err}
	}()

	return nil

}

func createCmd(prg *Program) *exec.Cmd {
	cmd := exec.Command(string(prg.bin), prg.args...)
	cmd.Dir = string(prg.workingdir)
	cmd.Env = prg.env
	cmd.Stdout, _ = fileutil.OpenOutput(string(prg.stdout))
	cmd.Stderr, _ = fileutil.OpenOutput(string(prg.stderr))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func (s *Spawner) spawn(p *Process) error {
	var err error

	if err = p.parent.validate(); err != nil {
		return err
	}

	resChan := make(chan error, 1)
	s.requestQueue <- SpawnRequest{
		process:    p,
		eventsChan: p.parent.manager.events,
		resultChan: resChan,
	}
	return <-resChan
}
