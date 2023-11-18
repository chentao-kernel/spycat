package core

import (
	"os"
)

type ContainerInfo struct {
	ContainerId string
	NameSpace   string
}

type SessionConfig struct {
	Container  ContainerInfo
	Pid        uint32
	StackInfo  bool
	Threshold  uint32
	SampleRate uint32
	Cpu        uint32
	RunTime    uint32 // hour unit
	BufferSize uint32
}

type BpfSpyer interface {
	Start() error
	Stop() error
	ConsumeEvent() error
	Name() string
}

type Session struct {
	name       string
	Config     *SessionConfig
	DataBuffer chan any
	Sig        chan os.Signal
}

func NewSession(name string, config *SessionConfig, bufSize uint32) *Session {
	return &Session{
		name:       name,
		Config:     config,
		DataBuffer: make(chan any, bufSize),
		Sig:        make(chan os.Signal, 1),
	}
}

func (s *Session) Start() error {
	return nil
}

func (s *Session) Stop() error {
	return nil
}

func (s *Session) Name() string {
	return s.name
}

func (s *Session) ConsumeEvent() error {
	return nil
}
