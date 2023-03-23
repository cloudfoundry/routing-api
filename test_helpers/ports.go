package test_helpers

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
)

var (
	lastPortUsed int
	portLock     sync.Mutex
	once         sync.Once
)

func NextAvailPort() int {
	portLock.Lock()
	defer portLock.Unlock()

	if lastPortUsed == 0 {
		once.Do(func() {
			const portRangeStart = 24000
			lastPortUsed = portRangeStart + GinkgoParallelProcess()
		})
	}

	suiteCfg, _ := GinkgoConfiguration()
	lastPortUsed += suiteCfg.ParallelTotal
	return lastPortUsed
}
