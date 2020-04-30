package cpu

import (
	"testing"
)

func TestComputeEffective(t *testing.T) {
	var dm MemoryBackend
	var cases = []struct {
		addr     uint16
		indirect bool
		ixReg    uint8
		expected uint32
	}{
		{0, false, 0, 0},
		{0, false, 1, 0x10},
		{0, true, 1, 0x1244},
		{2, false, 0, 2},
		{2, false, 1, 0x12},
		{2, true, 1, 0x0010},
	}

	cpu := NewCPU()
	cpu.G[1] = 0x10
	dm = NewDirectMemory(16)
	mr := MemoryRange{Low: 0, High: 15}
	cpu.RegisterMemory(mr, dm)
	cpu.StoreWord(0, 0x1234)

	expected := uint32(0x00001234)
	seen := cpu.FetchWord(0)
	if seen != expected {
		t.Errorf("Memory has wrong value, saw %x expected %x", seen, expected)
	}
	for ix, c := range cases {
		cpu.IC = 0
		expected := c.expected
		seen := cpu.computeEffective(c.addr, c.indirect, c.ixReg)
		if seen != expected {
			t.Errorf("Case #%d, saw address %x, expeced %x", ix, seen, expected)
		}
	}
}

func TestDirectMemory(t *testing.T) {
	var dm MemoryBackend
	c := NewCPU()

	cases := []struct {
		address uint32
		expected uint32
	}{{0, 0x12345678},{1, 0x56780000}, {2, 0}}
	dm = NewDirectMemory(16)
	mr := MemoryRange{Low: 0, High: 15}
	c.RegisterMemory(mr, dm)
	dm.(*DirectMemory).memory[0] = 0x1234
	dm.(*DirectMemory).memory[1] = 0x5678

	{
		tmp, offset := c.findMemory(3)
		if tmp != dm {
			t.Errorf("Unexpected memory backend found.")
		}
		if offset != 3 {
			t.Errorf("Unexpected offset, saw %d, expected 3", offset)
		}
	}
	
	for ix, tc := range cases {
		seen := c.FetchWord(tc.address)
		if seen != tc.expected {
			t.Errorf("Case #%d, unexpected value from address %d, saw 0x%08x, expected 0x%08x", ix, tc.address, seen, tc.expected)
		}
	}

}

func TestBasicInstructions(t *testing.T) {
	var dm *DirectMemory
	dm = NewDirectMemory(16)
	c := NewCPU()
	c.RegisterMemory(MemoryRange{0, 15}, dm)

	dm.memory[0] = 0x9a12
	dm.memory[1] = 0x1234
	c.Step()

	if c.G[1] != 0x1234 {
		t.Errorf("c.G[1] is 0x%08x, expected 0x00001234", c.G[1])
	}
	if c.IC != 2 {
		t.Errorf("c.IC is %d, expected 2", c.IC)
	}
}
