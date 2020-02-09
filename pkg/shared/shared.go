package shared
// A memory backend designed to be attached to multiple CPUs a the same time.

import (
	"github.com/sirupsen/logrus"
)

type op interface {
	execute(*sharedMemoryBackend)
}

type SharedMemory struct {
	cmd chan op
}

type sharedMemoryBackend struct {
	memory []uint16
	cmd chan op
}

type setHalfWord struct {
	addr  uint32
	value uint16
	ret   chan uint16
}

type getHalfWord struct {
	addr uint32
	ret  chan uint16
}

type setWord struct {
	addr  uint32
	value uint32
	ret   chan uint32
}

type getWord struct {
	addr uint32
	ret  chan uint32
}


func (c getWord) execute(b *sharedMemoryBackend) {
	fields := logrus.Fields{
		"addr": c.addr,
		"op": "get",
	}
	logrus.WithFields(fields).Info("get value")
	h0 := uint32(b.memory[c.addr])
	h1 := uint32(b.memory[c.addr + 1])

	c.ret <- ((h0 << 16) | h1)
}

func (c setWord) execute(b *sharedMemoryBackend) {
	fields := logrus.Fields{
		"addr": c.addr,
		"value": c.value,
		"op": "set",
	}
	logrus.WithFields(fields).Info("set value")
	h0 := uint32(b.memory[c.addr])
	h1 := uint32(b.memory[c.addr + 1])

	rv := (h0 << 16) | h1
	b.memory[c.addr] = uint16((c.value & 0xffff0000) >> 16)
	b.memory[c.addr + 1] = uint16(c.value & 0xffff)

	c.ret <- rv
}

func (c getHalfWord) execute(b *sharedMemoryBackend) {
	fields := logrus.Fields{
		"addr": c.addr,
		"op": "getHalf",
	}
	logrus.WithFields(fields).Info("get value")
	c.ret <- b.memory[c.addr]
}

func (c setHalfWord) execute (b *sharedMemoryBackend) {
	fields := logrus.Fields{
		"addr": c.addr,
		"value": c.value,
		"op": "setHalf",
	}
	logrus.WithFields(fields).Info("set value")
	rv := b.memory[c.addr]
	b.memory[c.addr] = c.value
	c.ret <- rv
}

func (b *sharedMemoryBackend) run() {
	for cmd := range b.cmd {
		cmd.execute(b)
	}
}

func NewSharedMemory(size uint32) SharedMemory {
	c := make(chan op)
	store := make([]uint16, size)
	backend := sharedMemoryBackend{cmd: c, memory: store}
	go backend.run()

	return SharedMemory{cmd: c}
}

func (s SharedMemory) FetchHalfWord(addr uint32) uint16 {
	fields := logrus.Fields{
		"op": "FetchHalfWord",
		"addr": addr,
	}
	logrus.WithFields(fields).Info("thing")
	c := make(chan uint16)
	op := getHalfWord{addr: addr, ret: c}
	s.cmd <- op
	rv := <- c
	close(c)
	return rv
}

func (s SharedMemory) FetchWord(addr uint32) uint32 {
	fields := logrus.Fields{
		"op": "FetchWord",
		"addr": addr,
	}
	logrus.WithFields(fields).Info("thing")
	c := make(chan uint32)
	op := getWord{addr: addr, ret: c}
	s.cmd <- op
	rv := <- c
	close(c)
	return rv
}

func (s SharedMemory) WriteHalfWord(addr uint32, data uint16) uint16 {
	fields := logrus.Fields{
		"op": "WriteHalfWord",
		"addr": addr,
	}
	logrus.WithFields(fields).Info("thing")
	c := make(chan uint16)
	op := setHalfWord{addr: addr, value: data, ret: c}
	s.cmd <- op
	rv := <- c
	close(c)
	return rv
}

func (s SharedMemory) WriteWord(addr uint32, data uint32) uint32 {
	c := make(chan uint32)
	fields := logrus.Fields{
		"op": "c",
		"addr": addr,
	}
	logrus.WithFields(fields).Info("thing")
	op := setWord{addr: addr, value: data, ret: c}
	s.cmd <- op
	rv := <- c
	close(c)
	return rv
}
