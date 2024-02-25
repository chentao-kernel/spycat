package symtab

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/chentao-kernel/spycat/pkg/log"
)

type SymSession struct {
	SymCache    *SymbolCache
	roundNumber int
	cacheSize   int
}

func NewSymSession() (*SymSession, error) {
	var err error
	session := &SymSession{
		cacheSize:   100,
		roundNumber: 10,
	}
	session.SymCache, err = NewSymbolCache(session.cacheSize)
	if err != nil {
		log.Loger.Error("new sym cache failed")
		return nil, err
	}
	return session, nil
}

func (s *SymSession) WalkStack(line *bytes.Buffer, stack []byte, pid uint32, userspace bool) {
	if len(stack) == 0 {
		return
	}
	var stackFrames []string
	for i := 0; i < 127; i++ {
		it := stack[i*8 : i*8+8]
		ip := binary.LittleEndian.Uint64(it)
		if ip == 0 {
			break
		}
		// tt pid != 0 user stack , pid == 0 kernel stack
		sym := s.SymCache.BccResolve(pid, ip, s.roundNumber)
		if !userspace && sym.Name == "" {
			continue
		}
		name := sym.Name
		if sym.Name == "" {
			if sym.Module != "" {
				name = fmt.Sprintf("%s+0x%x", sym.Module, sym.Offset)
			} else {
				name = "[unknown]"
			}
		}
		//fmt.Printf("==%x, name:%s\n", ip, name)
		stackFrames = append(stackFrames, name+";")
	}
	reverse(stackFrames)
	for _, s := range stackFrames {
		line.Write([]byte(s))
	}
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
