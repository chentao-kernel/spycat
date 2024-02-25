package symtab

import (
	"fmt"
	"sync"

	"github.com/chentao-kernel/spycat/pkg/util/genericlru"
)

type symbolCacheEntry struct {
	symbolTable SymbolTable
	roundNumber int
}

type pidKey uint32

type SymbolCache struct {
	pid2Cache *genericlru.GenericLRU[pidKey, symbolCacheEntry]
	mutex     sync.Mutex
}

func NewSymbolCache(cacheSize int) (*SymbolCache, error) {
	pid2Cache, err := genericlru.NewGenericLRU[pidKey, symbolCacheEntry](cacheSize, func(pid pidKey, e *symbolCacheEntry) {
		e.symbolTable.Close()
	})
	if err != nil {
		return nil, err
	}
	return &SymbolCache{
		pid2Cache: pid2Cache,
	}, nil
}

func (sc *SymbolCache) BccResolve(pid uint32, addr uint64, roundNumber int) Symbol {
	e := sc.GetOrCreateCacheEntry(pidKey(pid))
	staleCheck := false
	if roundNumber != e.roundNumber {
		e.roundNumber = roundNumber
		staleCheck = true
	}
	// kernelSymTab或者userSymTab
	return e.symbolTable.Resolve(addr, staleCheck)
}

func (sc *SymbolCache) GetOrCreateCacheEntry(pid pidKey) *symbolCacheEntry {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if cache, ok := sc.pid2Cache.Get(pid); ok {
		return cache
	}
	var symbolTable SymbolTable
	exe := fmt.Sprintf("/proc/%d/exe", pid)
	bcc := func() SymbolTable {
		return NewBCCSymbolTable(int(pid))
	}
	symbolTable, err := NewGoSymbolTable(exe, &bcc)
	if err != nil || symbolTable == nil {
		// bccSymTab pid=0或者非go程序
		symbolTable = bcc()
	}
	e := &symbolCacheEntry{symbolTable: symbolTable}
	sc.pid2Cache.Add(pid, e)
	return e
}

func (sc *SymbolCache) Clear() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	for _, pid := range sc.pid2Cache.Keys() {
		sc.pid2Cache.Remove(pid)
	}
}
