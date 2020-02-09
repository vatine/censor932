package shared

import (
	"testing"

	"github.com/vatine/censor932/pkg/cpu"
)

func TestShared(t *testing.T) {
	s := NewSharedMemory(16)

	c1 := cpu.NewCPU()
	c2 := cpu.NewCPU()

	r := cpu.MemoryRange{Low: 0, High: 15}

	c1.RegisterMemory(r, s)
	c2.RegisterMemory(r, s)
	
	c1.StoreHalfWord(0, 0x1234)
	c1.StoreHalfWord(1, 0x5678)

	v := c2.FetchWord(0)

	if v != 0x12345678 {
		t.Errorf("Expected 0x12345678, saw 0x%08x", v)
	}
}
