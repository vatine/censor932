// The Censor 932 CPU package.
//
// After the machine description at:
//   http://www.veteranklubbenalfa.se/veteran/13q1/130215.pdf
//
// The cpu package includes instructions, instruction decoding, memory
// interfaces, etc.
package cpu

import (
	"fmt"
	
	"github.com/sirupsen/logrus"
)

// A struct providing upper and lower bounds for a MemoryBackend
type MemoryRange struct {
	Low, High uint32
}

// The general interface for MemoryBackend storage.
type MemoryBackend interface {
	// Retrieve a HalfWord from the specified memory address
	FetchHalfWord(uint32) uint16
	// Store a HalfWord to a specific memory address, return the
	// previous value stored there.
	WriteHalfWord(uint32, uint16) uint16
	// Retrieve a Word from the specified address and adress + 1
	FetchWord(uint32) uint32
	// Store a Word to the specified address (and address + 1),
	// return the Word that was previously stored there.
	WriteWord(uint32, uint32) uint32
}

// Rgeister a given MemoryBackend as the storage backend starting at
// Range.Low, ending at Range.High.
type MemoryPlugin struct {
	Range   MemoryRange
	Backend MemoryBackend
}

// Basic CPU data structure
type CPU struct {
	G      [16]uint32
	IC     uint32 		// This is technically an 18-bit entity
	PS     uint64
	MIR    uint32		// Actually a 24-bit entity
	Memory []MemoryPlugin
	CC     uint8
}

// Pull the upper 32 bits out of a 64-bit entity
func extractUpperWord(w uint64) uint32 {
	w1 := (w & 0xffffffff00000000) >> 32
	return uint32(w1)
}

// Pull the lower 32 bits out of a 64-bit entity
func extractLowerWord(w uint64) uint32 {
	w1 := w & 0xffffffff
	return uint32(w1)
}

// General Instruction abstraction
type Instruction interface {
	Execute(*CPU) uint32	// Return the next IC
}

type InstructionBuilder func(op uint8, r1, r2 uint8, rest uint16) Instruction

var instructionTable map[uint8]InstructionBuilder

// Register an opcode builder against a specific opcode
func registerFunction(opcode uint8, builder InstructionBuilder) {
	instructionTable[opcode] = builder
}

func init() {
	instructionTable = map[uint8]InstructionBuilder{}
	
	registerFunction(0x4e, BuildIHFunc)
	registerFunction(0x5e, BuildIWFunc)
	registerFunction(0xcd, BuildLDFunc)
	registerFunction(0x98, BuildLDFunc)
	registerFunction(0x68, BuildLDWFunc)
	registerFunction(0x48, BuildLHFunc)
	registerFunction(0xcb, BuildLNFunc)
	registerFunction(0xca, BuildLPFunc)
	registerFunction(0xb8, BuildLRSFunc)
	registerFunction(0xcc, BuildLTFunc)
	registerFunction(0x58, BuildLWFunc)
	registerFunction(0x4f, BuildRZHFunc)
	registerFunction(0x5f, BuildRZWFunc)
	registerFunction(0xb0, BuildSRSFunc)
	registerFunction(0x60, BuildSTDWFunc)
	registerFunction(0x40, BuildSTHFunc)
	registerFunction(0x50, BuildSTWFunc)
	registerFunction(0x9a, BuildADFunc)
	registerFunction(0x6a, BuildADWFunc)
	registerFunction(0x4a, BuildAHFunc)
	registerFunction(0x1a, BuildASFunc)
	registerFunction(0x2a, BuildATSFunc)
	registerFunction(0x5a, BuildAWFunc)
	registerFunction(0x99, BuildCDFunc)
	registerFunction(0x49, BuildCHFunc)
	registerFunction(0x59, BuildCWFunc)
	registerFunction(0x9d, BuildDHFunc)
	registerFunction(0x4d, BuildDHFunc)
	registerFunction(0x1d, BuildDSFunc)
	registerFunction(0x5d, BuildDWFunc)
	registerFunction(0x9c, BuildMDFunc)
	registerFunction(0x4c, BuildMHFunc)
	registerFunction(0x1c, BuildMSFunc)
	registerFunction(0x5c, BuildMWFunc)
	registerFunction(0x9b, BuildSDFunc)
	registerFunction(0x6b, BuildSDWFunc)
	registerFunction(0x2b, BuildSFSFunc)
	registerFunction(0x4b, BuildSHFunc)
	registerFunction(0x1b, BuildSSFunc)
	registerFunction(0x5b, BuildSWFunc)
	registerFunction(0x97, BuildCLDFunc)
	registerFunction(0x47, BuildCLHFunc)
	registerFunction(0x57, BuildCLWFunc)
	registerFunction(0x94, BuildNDFunc)
	registerFunction(0x44, BuildNHFunc)
	registerFunction(0x14, BuildNSFunc)
	registerFunction(0x24, BuildNTSFunc)
	registerFunction(0x54, BuildNWFunc)
	registerFunction(0x95, BuildNDFunc)
	registerFunction(0x45, BuildNHFunc)
	registerFunction(0x15, BuildOSFunc)
	registerFunction(0x25, BuildOTSFunc)
	registerFunction(0x55, BuildOWFunc)
	registerFunction(0x96, BuildXDFunc)
	registerFunction(0x46, BuildXHFunc)
	registerFunction(0x16, BuildXSFunc)
	registerFunction(0x26, BuildXTSFunc)
	registerFunction(0x56, BuildXWFunc)
	registerFunction(0x85, BuildRLSFunc)
	registerFunction(0x87, BuildRLDFunc)
	registerFunction(0x84, BuildRRSFunc)
	registerFunction(0x86, BuildRRDFunc)
	registerFunction(0x89, BuildSLAFunc)
	registerFunction(0x8b, BuildSLDAFunc)
	registerFunction(0x8d, BuildSLLFunc)
	registerFunction(0x8f, BuildSLDLFunc)
	registerFunction(0x88, BuildSRAFunc)
	registerFunction(0x8a, BuildSRDAFunc)
	registerFunction(0x8c, BuildSRLFunc)
	registerFunction(0x8e, BuildSRDLFunc)
	// registerFunction(0xc0, BuildCPFunc)a
	registerFunction(0xc1, BuildEXFunc)
	registerFunction(0x05, BuildJCFunc)
	registerFunction(0x02, BuildJOSFunc)
	registerFunction(0x03, BuildJTSFunc)
	registerFunction(0x04, BuildJOAFunc)
	registerFunction(0x01, BuildJSFunc)
	// registerFunction(0x06, BuildJSPFunc)
	// registerFunction(0c3, BuildLSKFunc)
	// registerFunction(0c2, BuildLSPFunc)
	registerFunction(0x00, BuildNOPFunc)
	// registerFunction(0x08, BuildSTSRFunc)
}

func NewCPU() *CPU {
	var rv CPU
	rv.Memory = []MemoryPlugin{}

	return &rv
}

// Set correct values for condition code, depending on value and type of operation.
func (c *CPU) setCC(opType int, value uint32) {
	switch {
	case opType == 1: // Artithmetic
		switch {
		case (value & 0x80000000) != 0:
			c.CC = 1
		case value == 0:
			c.CC = 0
		default:
			c.CC = 2
		}
	case opType == 2: // Logical
		if value == 0 {
			c.CC = 0
		} else {
			c.CC = 1
		}
	case opType == 3: // Compare
		switch {
		case value == 0:
			c.CC = 0
		case value & 0xf0000000 == 0:
			c.CC = 2
		default:
			c.CC = 3
		}
	}
}

// Compute effective address based on *should it be indirect" and
// "what index register should be used" (until a full manual is
// available, 0 is intepreted as "no index register". Note that the
// effective address is an 18-bit unit, so we have to manage it as a
// 32-bit unit and mask it to fit within the available address space.
func (c *CPU) computeEffective(addr uint16, indirect bool, ixReg uint8) uint32 {
	rv := uint32(addr)
	fmt.Printf("DEBUG: rv is %x, indirect is %v, ixreg is %d\n", rv, indirect, ixReg)

	rv = rv + c.IC
	rv = rv & 0x03FFFF
	fmt.Printf("DEBUG: rv+IC is %x\n", rv)

	if indirect {
		rv = c.FetchWord(rv)
		rv = rv & 0x03FFFF
		fmt.Printf("DEBUG: post-indirect, rv is %x\n", rv)
	}
	
	if (ixReg >= 1) && (ixReg <= 7) {
		rv += c.G[ixReg]
		rv = rv & 0x03FFFF
		fmt.Printf("DEBUG: post-index, rv is %x\n", rv)
	}
	
	rv = rv & 0x03FFFF

	return rv
}

func decodeWord(word uint32) Instruction {
	fields := logrus.Fields{
		"word": word,
	}
	opCode := uint8((word & 0xFF000000) >> 24)
	builder, ok := instructionTable[opCode]
	if !ok {
		logrus.WithFields(fields).Errorf("Non-existent instruction, %02x", opCode)
		return BuildNOPFunc(0,0,0,0)
	}

	r1 := uint8((word & 0x00f00000) >> 20)
	r2 := uint8((word & 0x000f0000) >> 16)
	rest := uint16(word & 0x0000ffff)

	return builder(opCode, r1, r2, rest)
}

// Make the CPU take another "step" (this is a fetch, execute, optionally stop)
func (c *CPU) Step() {
	fields := logrus.Fields{
		"IC": c.IC,
	}
	logrus.WithFields(fields).Debug("CPU Step")
	word := c.FetchWord(c.IC)


	c.IC = decodeWord(word).Execute(c)
}

// Return the memoryPluging that corresponds to a specific address
func (c *CPU) findMemory(address uint32) (MemoryBackend, uint32) {
	for _, mp := range c.Memory {
		if (mp.Range.Low <= address) && (address <= mp.Range.High) {
			return mp.Backend, address - mp.Range.Low
		}
	}
	return nil, 0
}

// Register a memory backend with a specific memory range. Return an
// error if the memory plugin is colliding with an alread-registered
// plugin.
func (c *CPU) RegisterMemory(r MemoryRange, m MemoryBackend) error {
	p := MemoryPlugin{Range: r, Backend: m}
	for _, tmp := range c.Memory {
		if (tmp.Range.Low <= r.High) && (r.Low <= tmp.Range.High) {
			return fmt.Errorf("Memory backend %v conflicting with already-registered plugin %v", m, tmp)
		}
	}
	c.Memory = append(c.Memory, p)
	return nil
}

// Fetch a 32-bit word from a specific address
func (c *CPU) FetchWord(address uint32) uint32 {
	mp, offset := c.findMemory(address)

	return mp.FetchWord(offset)
}

// Fetch a 16-bit word from a specific address
func (c *CPU) FetchHalfWord(address uint32) uint16 {
	mp, offset := c.findMemory(address)

	return mp.FetchHalfWord(offset)
}

func (c *CPU) StoreWord(address, word uint32) uint32 {
	mp, offset := c.findMemory(address)

	return mp.WriteWord(offset, word)
}

func (c *CPU) StoreHalfWord(address uint32, word uint16) uint16 {
	mp, offset := c.findMemory(address)

	return mp.WriteHalfWord(offset, word)
}

// Various stuff for implementing "local" memory
type DirectMemory struct {
	memory []uint16
}

func (m *DirectMemory) FetchWord(address uint32) uint32 {
	w1 := uint32(m.memory[address])
	w2 := uint32(m.memory[address + 1])
	return (w1 << 16) | w2
}

func (m *DirectMemory) FetchHalfWord(address uint32) uint16 {
	return m.memory[address]
}

func (m *DirectMemory) WriteHalfWord(address uint32, data uint16) uint16 {
	old := m.memory[address]
	m.memory[address] = data
	return old
}

func (m *DirectMemory) WriteWord(address, data uint32) uint32 {
	old := m.FetchWord(address)
	m.memory[address] = uint16((data & 0xffff0000) >> 16)
	m.memory[address + 1] = uint16(data & 0xffff)
	return old
}

func NewDirectMemory(size uint32) *DirectMemory {
	var rv DirectMemory

	rv.memory = make([]uint16, size)

	return &rv
}

// Type1 instructions all have a index register reference. However,
// the CPu description document has no reference to how the index
// registers work, nor how to set them. One of the later models does
// have some load instructions that increment the index registers, but
// it is not clear, at all, how this works in the base model.
type type1 struct {
	op uint8
	r  uint8
	i  bool
	x  uint8
	as uint16
}

func buildType1(op uint8, r1, r2 uint8, rest uint16) type1 {
	rv := type1{op: op, r: r1, as: rest}
	rv.i = (r2 & 0x08) == 0x08
	rv.x = r2 & 0x07
	return rv
}

type type2 struct {
	op uint8
	r1 uint8
	r2 uint8
	as uint16
}

func buildType2(op, r1, r2 uint8, as uint16) type2 {
	return type2{
		op: op,
		r1: r1,
		r2: r2,
		as: as,
	}
}

type type3 struct {
	op uint8
	r1 uint8
	r2 uint8
	d  uint16
}

func buildType3(op, r1, r2 uint8, d uint16) type3 {
	return type3{
		op: op,
		r1: r1,
		r2: r2,
		d: d,
	}
}

// Interchange full word
type IW type1
func (i IW) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)

	c.G[i.r] = c.StoreWord(target, c.G[i.r])

	return c.IC + 2
}
func BuildIWFunc(op uint8, r1, r2 uint8, rest uint16) Instruction {
	return IW(buildType1(op, r1, r2, rest))
}

// Interchange half word
type IH type1
func (i IH) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)

	c.G[i.r] = uint32(c.StoreHalfWord(target, uint16(c.G[i.r] & 0x0000ffff)))

	return c.IC + 2
}
func BuildIHFunc(op uint8, r1, r2 uint8, rest uint16) Instruction {
	return IH(buildType1(op, r1, r2, rest))
}


// Load Complement
type LC type1
func (i LC) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	tmp := c.FetchWord(source)
	tmp = (tmp ^ 0xffffffff) + 1

	c.G[i.r] = tmp
	c.setCC(1, tmp)

	return c.IC + 2
}
func BuildLCFunc(op uint8, r1, r2 uint8, rest uint16) Instruction {
	return LC(buildType1(op, r1, r2, rest))
}

// Load Direct
type LD type3
func (i LD) Execute (c *CPU) uint32 {
	c.G[i.r1] = uint32(i.d)

	return c.IC + 2
}
func BuildLDFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return LD(buildType3(op, r1, r2, rest))
}


// Load Double Word
type LDW type1
func (i LDW) Execute (c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	c.G[i.r] = c.FetchWord(source)
	source = c.computeEffective(i.as + 2, i.i, i.x)
	c.G[i.r + 1] = c.FetchWord(source)

	return c.IC + 2
}
func BuildLDWFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return LDW(buildType1(op, r1, r2, rest))
}


// Load Half Word
type LH type1
func (i LH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	c.G[i.r] = uint32(c.FetchHalfWord(source))

	return c.IC + 2
}
func BuildLHFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return LH(buildType1(op, r1, r2, rest))
}

// Load Negative
type LN type1
func (i LN) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	tmp := c.FetchWord(source)
	if (tmp & 0x80000000) == 0 {
		tmp = (tmp ^ 0xffffffff) + 1
	}

	c.G[i.r] = tmp
	c.setCC(1, tmp)

	return c.IC + 2
}
func BuildLNFunc(op uint8, r1, r2 uint8, rest uint16) Instruction {
	return LN(buildType1(op, r1, r2, rest))
}


// LOC
// LOU
// LP
type LP type1
func (i LP) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	tmp := c.FetchWord(source)
	if (tmp & 0x80000000) != 0 {
		tmp = (tmp ^ 0xffffffff) + 1
	}

	c.G[i.r] = tmp
	c.setCC(1, tmp)

	return c.IC + 2
}
func BuildLPFunc(op uint8, r1, r2 uint8, rest uint16) Instruction {
	return LP(buildType1(op, r1, r2, rest))
}

// LRS
type LRS type2
func (i LRS) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, false, 0)
	c.G[i.r1] = c.FetchWord(source)
	source = c.computeEffective(i.as + 2, false, 0)
	c.G[i.r2] = c.FetchWord(source)

	return c.IC + 2
}
func BuildLRSFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return LRS(buildType2(op, r1, r2, rest))
}

// LT
type LT type1
func (i LT) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	tmp := c.FetchWord(source)
	c.G[i.r] = tmp
	c.setCC(1, tmp)

	return c.IC + 2
}
func BuildLTFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return LT(buildType1(op, r1, r2, rest))
}

// LW
type LW type1
func (i LW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	c.G[i.r] = c.FetchWord(source)

	return c.IC + 2
}
func BuildLWFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return LW(buildType1(op, r1, r2, rest))
}

// RZH
type RZH type1
func (i RZH) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)
	c.StoreHalfWord(target, 0)
	return c.IC + 2
}
func BuildRZHFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return RZH(buildType1(op, r1, r2, rest))
}

// RZW
type RZW type1
func (i RZW) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)
	c.StoreWord(target, 0)
	return c.IC + 2
}
func BuildRZWFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return RZW(buildType1(op, r1, r2, rest))
}

// SIC
// SIU
// SRS
type SRS type2
func (i SRS) Execute(c *CPU) uint32 {
	extra := uint16(0)
	for r := i.r1; r <= i.r2; r++ {
		target := c.computeEffective(i.as + extra, false, 0)
		c.StoreWord(target, c.G[r])
		extra += 2 
	}
	return c.IC + 2
}
func BuildSRSFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return SRS(buildType2(op, r1, r2, rest))
}

// STDW
type STDW type1
func (i STDW) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)
	c.StoreWord(target, c.G[i.r])
	target = c.computeEffective(i.as + 2, i.i, i.x)
	c.StoreWord(target, c.G[i.r+1])

	return c.IC + 2
}
func BuildSTDWFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return STDW(buildType1(op, r1, r2, rest))
}

// STH
type STH type1
func (i STH) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)
	c.StoreHalfWord(target, uint16(c.G[i.r] & 0xffff))

	return c.IC + 2
}
func BuildSTHFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return STH(buildType1(op, r1, r2, rest))
}

// STW
type STW type1
func (i STW) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, i.i, i.x)
	c.StoreWord(target, c.G[i.r])

	return c.IC + 2
}
func BuildSTWFunc(op, r1, r2 uint8, rest uint16) Instruction {
	return STW(buildType1(op, r1, r2, rest))
}

// AD
type AD type3
func (i AD) Execute(c *CPU) uint32 {
	c.G[i.r1] = c.G[i.r2] + uint32(i.d)
	c.setCC(1, c.G[i.r1])

	return c.IC + 2
}
func BuildADFunc(op, r1, r2 uint8, d uint16) Instruction {
	return AD(buildType3(op, r1, r2, d))
}

// ADW
type ADW type1
func (i ADW) Execute(c *CPU) uint32 {
	tmp0 := c.G[i.r]
	tmp1 := c.G[i.r + 1]

	source := c.computeEffective(i.as + 2, i.i, i.x)
	tmp1 = tmp1 + c.FetchWord(source)
	var carry uint32
	if tmp1 < c.G[i.r + 1] {
		carry = 1
	}
	source = c.computeEffective(i.as, i.i, i.x)
	tmp0 = tmp0 + carry + c.FetchWord(source)

	if tmp0 == 0 {
		c.setCC(1, tmp1)
	} else {
		c.setCC(1, tmp0)
	}
	c.G[i.r] = tmp0
	c.G[i.r + 1] = tmp1

	return c.IC + 2
}
func BuildADWFunc(op, r, ix uint8, as uint16) Instruction {
	return ADW(buildType1(op, r, ix, as))
}

// AH
type AH type1
func (i AH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	result := c.G[i.r] + uint32(c.FetchHalfWord(source))
	c.setCC(1, result)
	c.G[i.r] = result
	return c.IC + 2
}

func BuildAHFunc(op, r, ix uint8, as uint16) Instruction {
	return AH(buildType1(op, r, ix, as))
}

// AS
type AS type2
func (i AS) Execute(c *CPU) uint32 {
	sum := c.G[i.r1] + c.G[i.r2]
	target := c.computeEffective(i.as, false, 0)
	c.setCC(2, sum)
	c.StoreWord(target, sum)
	return c.IC + 2
}

func BuildASFunc(op, r1, r2 uint8, as uint16) Instruction {
	return AS(buildType2(op, r1, r2, as))
}

// ATS
type ATS type1
func (i ATS) Execute(c *CPU) uint32 {
	effective := c.computeEffective(i.as, i.i, i.x)
	sum := c.FetchWord(effective) + c.G[i.r]
	c.setCC(1, sum)
	c.StoreWord(effective, sum)
	return c.IC + 2
}
func BuildATSFunc(op, r, ix uint8, as uint16) Instruction {
	return ATS(buildType1(op, r, ix, as))
}

// AW
type AW type1
func (i AW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	sum := c.G[i.r] + c.FetchWord(source)
	c.G[i.r] = sum
	c.setCC(1, sum)
	return c.IC + 2
}
func BuildAWFunc(op, r, ix uint8, as uint16) Instruction {
	return AW(buildType1(op, r, ix, as))
}


// CD
type CD type3
func (i CD) Execute(c *CPU) uint32 {
	c.setCC(3, c.G[i.r2] - uint32(i.d))
	return c.IC + 2
}
func BuildCDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return CD(buildType3(op, r1, r2, d))
}

// CH
type CH type1
func (i CH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	contents := uint32(c.FetchHalfWord(source))
	c.setCC(2, c.G[i.r] - contents)

	return c.IC + 2
}
func BuildCHFunc(op, r, ix uint8, as uint16) Instruction {
	return CH(buildType1(op, r, ix, as))
}

// CW
type CW type1
func (i CW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	contents := c.FetchWord(source)
	c.setCC(2, c.G[i.r] - contents)

	return c.IC + 2
}
func BuildCWFunc(op, r, ix uint8, as uint16) Instruction {
	return CW(buildType1(op, r, ix, as))
}

// DD
type DD type3
func (i DD) Execute(c *CPU) uint32 {
	d := uint32(i.d)
	c.G[i.r1] = c.G[i.r2] / d
	return c.IC + 2
}

func BuildDDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return DD(buildType3(op, r1, r2, d))
}

// DH
type DH type1
func (i DH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	h := uint32(c.FetchHalfWord(source))
	c.G[i.r] = c.G[i.r] / h

	return c.IC + 2
}
func BuildDHFunc(op, r, ix uint8, as uint16) Instruction {
	return DH(buildType1(op, r, ix, as))
}

// DS
type DS type2
func (i DS) Execute(c *CPU) uint32 {
	upper := uint64(c.G[i.r1]) << 32 + uint64(c.G[i.r1 + 1])
	result := upper / uint64(c.G[i.r2])
	target := c.computeEffective(i.as, false, 0)
	c.StoreWord(target, extractUpperWord(result))
	target = c.computeEffective(i.as + 2, false, 0)
	c.StoreWord(target, extractLowerWord(result))

	return c.IC + 2
}
func BuildDSFunc(op, r1, r2 uint8, as uint16) Instruction {
	return DS(buildType2(op, r1, r2, as))
}

// DW
type DW type1
func (i DW) Execute(c *CPU) uint32 {
	value := uint64(c.G[i.r] << 32)
	value = value + uint64(c.G[i.r + 1])
	source := c.computeEffective(i.as, i.i, i.x)
	dividend := uint64(c.FetchWord(source))
	result := value / dividend
	c.G[i.r] = extractUpperWord(result)
	c.G[i.r + 1] = extractLowerWord(result)

	return c.IC + 2
}
func BuildDWFunc(op, r, ix uint8, as uint16) Instruction {
	return DW(buildType1(op, r, ix, as))
}

// MD
type MD type3
func (i MD) Execute(c *CPU) uint32 {
	c.G[i.r1] = c.G[i.r2] * uint32(i.d)
	return c.IC + 2
}
func BuildMDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return MD(buildType3(op, r1, r2, d))
}

// MH
type MH type1
func (i MH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	value := uint32(c.FetchHalfWord(source))
	c.G[i.r] = c.G[i.r] * value

	return c.IC + 2
}
func BuildMHFunc(op, r, ix uint8, as uint16) Instruction {
	return MH(buildType1(op, r, ix, as))
}

// MS
type MS type2
func (i MS) Execute(c *CPU) uint32 {
	result := uint64(c.G[i.r1]) * uint64(c.G[i.r2])
	target := c.computeEffective(i.as, false, 0)
	c.StoreWord(target, extractUpperWord(result))
	target = c.computeEffective(i.as + 2, false, 0)
	c.StoreWord(target , extractLowerWord(result))
	return c.IC + 2
}
func BuildMSFunc(op, r1, r2 uint8, as uint16) Instruction {
	return MS(buildType2(op, r1, r2, as))
}

// MW
type MW type1
func(i MW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	m1 := uint64(c.G[i.r])
	m2 := uint64(c.FetchWord(source))
	result := m1 * m2
	c.G[i.r] = extractUpperWord(result)
	c.G[i.r + 1] = extractLowerWord(result)

	return c.IC + 2
}
func BuildMWFunc(op, r, ix uint8, as uint16) Instruction {
	return MW(buildType1(op, r, ix, as))
}

// SD
type SD type3
func (i SD) Execute(c *CPU) uint32 {
	c.G[i.r1] = c.G[i.r2] - uint32(i.d)
	c.setCC(1, c.G[i.r1])

	return c.IC + 2
}
func BuildSDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return SD(buildType3(op, r1, r2, d))
}

// SDW
type SDW type1
func (i SDW) Execute(c *CPU) uint32 {
	v1 := uint64(c.G[i.r] << 32) + uint64(c.G[i.r + 1])
	source := c.computeEffective(i.as, i.i, i.x)
	v2 := uint64(c.FetchWord(source) << 32)
	source = c.computeEffective(i.as + 2, i.i, i.x)
	v2 += uint64(c.FetchWord(source))

	result := v1 - v2

	c.G[i.r] = extractUpperWord(result)
	c.G[i.r + 1] = extractLowerWord(result)
	if c.G[i.r] == 1 {
		c.setCC(1, c.G[i.r + 1])
	} else {
		c.setCC(1, c.G[i.r])
	}

	return c.IC + 2
}
func BuildSDWFunc(op, r, ix uint8, as uint16) Instruction {
	return SDW(buildType1(op, r, ix, as))
}

// SFS
type SFS type1
func (i SFS) Execute(c *CPU) uint32 {
	addr := c.computeEffective(i.as, i.i, i.x)
	result := c.FetchWord(addr) - c.G[i.r]
	c.setCC(1, result)
	c.StoreWord(addr, result)

	return c.IC + 2
}
func BuildSFSFunc(op, r, ix uint8, as uint16) Instruction {
	return SFS(buildType1(op, r, ix, as))
}

// SH
type SH type1
func (i SH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	c.G[i.r] = c.G[i.r] - uint32(c.FetchHalfWord(source))
	c.setCC(1, c.G[i.r])

	return c.IC + 2
}
func BuildSHFunc(op, r, ix uint8, as uint16) Instruction {
	return SH(buildType1(op, r, ix, as))
}

// SS
type SS type2
func (i SS) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, false, 0)
	result := c.G[i.r1] - c.G[i.r2]
	c.setCC(1, result)
	c.StoreWord(target, result)

	return c.IC + 2
}
func BuildSSFunc(op, r1, r2 uint8, as uint16) Instruction {
	return SS(buildType2(op, r1, r2, as))
}

// SW
type SW type1
func (i SW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	c.G[i.r] = c.G[i.r] - c.FetchWord(source)
	c.setCC(1, c.G[i.r])

	return c.IC + 2
}
func BuildSWFunc(op, r, ix uint8, as uint16) Instruction {
	return SW(buildType1(op, r, ix, as))
}

// CLD
type CLD type3
func (i CLD) Execute(c *CPU) uint32 {
	x := uint32(i.d)
	y := c.G[i.r2]
	c.setCC(2, y - x)
	return c.IC + 2
}
func BuildCLDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return CLD(buildType3(op, r1, r2, d))
}

// CLH
type CLH type1
func (i CLH) Execute(c *CPU) uint32 {
	x := c.G[i.r]
	source := c.computeEffective(i.as, i.i, i.x)
	y := uint32(c.FetchHalfWord(source))
	c.setCC(2, x - y)
	return c.IC + 2
}
func BuildCLHFunc(op, r, ix uint8, as uint16) Instruction {
	return CLH(buildType1(op, r, ix, as))
}

// CLW
type CLW type1
func (i CLW) Execute(c *CPU) uint32 {
	x := c.G[i.r]
	source := c.computeEffective(i.as, i.i, i.x)
	y := c.FetchWord(source)
	c.setCC(2, x - y)
	return c.IC + 2
}
func BuildCLWFunc(op, r, ix uint8, as uint16) Instruction {
	return CLW(buildType1(op, r, ix, as))
}

// ND
type ND type3
func (i ND) Execute(c *CPU) uint32 {
	d := uint32(i.d)
	c.G[i.r1] = c.G[i.r2] & d
	c.setCC(3, c.G[i.r1])
	return c.IC + 2
}
func BuildNDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return ND(buildType3(op, r1, r2, d))
}

// NH
type NH type1
func (i NH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	s := uint32(c.FetchHalfWord(source))
	c.G[i.r] = c.G[i.r] & s
	c.setCC(3, c.G[i.r])
	return c.IC + 2
}
func BuildNHFunc(op, r, ix uint8, as uint16) Instruction {
	return NH(buildType1(op, r, ix, as))
}

// NS
type NS type2
func (i NS) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, false, 0)
	result := c.G[i.r1] & c.G[i.r2]
	c.setCC(3, result)
	c.StoreWord(target, result)
	return c.IC + 2
}
func BuildNSFunc(op, r1, r2 uint8, as uint16) Instruction {
	return NS(buildType2(op, r1, r2, as))
}

// NTS
type NTS type1
func (i NTS) Execute(c *CPU) uint32 {
	location := c.computeEffective(i.as, i.i, i.x)
	s := c.FetchWord(location)
	result := s & c.G[i.r]
	c.setCC(3, result)
	c.StoreWord(location, result)
	return c.IC + 2
}
func BuildNTSFunc(op, r, ix uint8, as uint16) Instruction {
	return NTS(buildType1(op, r, ix, as))
}

// NW
type NW type1
func (i NW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.r)
	c.G[i.r] = c.G[i.r] & c.FetchWord(source)
	c.setCC(3, c.G[i.r])

	return c.IC + 2
}
func BuildNWFunc(op, r, ix uint8, as uint16) Instruction {
	return NW(buildType1(op, r, ix, as))
}

// OD
type OD type3
func (i OD) Execute(c *CPU) uint32 {
	d := uint32(i.d)
	c.G[i.r1] = c.G[i.r2] | d
	c.setCC(3, c.G[i.r1])
	return c.IC + 2
}
func BuildODFunc(op, r1, r2 uint8, d uint16) Instruction {
	return OD(buildType3(op, r1, r2, d))
}

// OH
type OH type1
func (i OH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	s := uint32(c.FetchHalfWord(source))
	c.G[i.r] = c.G[i.r] | s
	c.setCC(3, c.G[i.r])
	return c.IC + 2
}
func BuildOHFunc(op, r, ix uint8, as uint16) Instruction {
	return OH(buildType1(op, r, ix, as))
}

// OS
type OS type2
func (i OS) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, false, 0)
	result := c.G[i.r1] | c.G[i.r2]
	c.setCC(3, result)
	c.StoreWord(target, result)
	return c.IC + 2
}
func BuildOSFunc(op, r1, r2 uint8, as uint16) Instruction {
	return OS(buildType2(op, r1, r2, as))
}

// OTS
type OTS type1
func (i OTS) Execute(c *CPU) uint32 {
	location := c.computeEffective(i.as, i.i, i.x)
	s := c.FetchWord(location)
	result := s | c.G[i.r]
	c.setCC(3, result)
	c.StoreWord(location, result)
	return c.IC + 2
}
func BuildOTSFunc(op, r, ix uint8, as uint16) Instruction {
	return OTS(buildType1(op, r, ix, as))
}

// OW
type OW type1
func (i OW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.r)
	c.G[i.r] = c.G[i.r] | c.FetchWord(source)
	c.setCC(3, c.G[i.r])

	return c.IC + 2
}
func BuildOWFunc(op, r, ix uint8, as uint16) Instruction {
	return OW(buildType1(op, r, ix, as))
}

// XD
type XD type3
func (i XD) Execute(c *CPU) uint32 {
	d := uint32(i.d)
	c.G[i.r1] = c.G[i.r2] & d
	c.setCC(3, c.G[i.r1])
	return c.IC + 2
}
func BuildXDFunc(op, r1, r2 uint8, d uint16) Instruction {
	return XD(buildType3(op, r1, r2, d))
}

// XH
type XH type1
func (i XH) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	s := uint32(c.FetchHalfWord(source))
	c.G[i.r] = c.G[i.r] ^ s
	c.setCC(3, c.G[i.r])
	return c.IC + 2
}
func BuildXHFunc(op, r, ix uint8, as uint16) Instruction {
	return XH(buildType1(op, r, ix, as))
}

// XS
type XS type2
func (i XS) Execute(c *CPU) uint32 {
	target := c.computeEffective(i.as, false, 0)
	result := c.G[i.r1] ^ c.G[i.r2]
	c.setCC(3, result)
	c.StoreWord(target, result)
	return c.IC + 2
}
func BuildXSFunc(op, r1, r2 uint8, as uint16) Instruction {
	return XS(buildType2(op, r1, r2, as))
}

// XTS
type XTS type1
func (i XTS) Execute(c *CPU) uint32 {
	location := c.computeEffective(i.as, i.i, i.x)
	s := c.FetchWord(location)
	result := s ^ c.G[i.r]
	c.setCC(3, result)
	c.StoreWord(location, result)
	return c.IC + 2
}
func BuildXTSFunc(op, r, ix uint8, as uint16) Instruction {
	return XTS(buildType1(op, r, ix, as))
}

// XW
type XW type1
func (i XW) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.r)
	c.G[i.r] = c.G[i.r] ^ c.FetchWord(source)
	c.setCC(3, c.G[i.r])

	return c.IC + 2
}
func BuildXWFunc(op, r, ix uint8, as uint16) Instruction {
	return XW(buildType1(op, r, ix, as))
}

// RLS
type RLS type1
func (i RLS) Execute(c *CPU) uint32 {
	r := c.G[i.r]
	c.G[i.r] = ((r & 0x80000000) >> 31) | ((r & 0x7fffffff) << 1)
	return c.IC + 2
}
func BuildRLSFunc(op, r, ix uint8, as uint16) Instruction {
	return RLS(buildType1(op,r, ix, as))
}

// RLD
type RLD type1
func (i RLD) Execute(c *CPU) uint32 {
	r0 := c.G[i.r]
	r1 := c.G[i.r + 1]

	c.G[i.r + 1] = ((r0 & 0x80000000) >> 31) | ((r1 & 0x7fffffff) << 1)
	c.G[i.r] = ((r1 & 0x80000000) >> 31) | ((r0 & 0x7fffffff) << 1)

	return c.IC + 2
}
func BuildRLDFunc(op, r, ix uint8, as uint16) Instruction {
	return RLD(buildType1(op, r, ix, as))
}

// RRS
type RRS type1
func (i RRS) Execute(c *CPU) uint32 {
	r := c.G[i.r]
	c.G[i.r] = ((r & 1) << 31) | ((r & 0x7fffffff) >> 1)
	return c.IC + 2
}
func BuildRRSFunc(op, r, ix uint8, as uint16) Instruction {
	return RRS(buildType1(op, r, ix, as))
}

// RRD
type RRD type1
func (i RRD) Execute(c *CPU) uint32 {
	r0 := c.G[i.r]
	r1 := c.G[i.r + 1]

	c.G[i.r + 1] = ((r0 & 0x1) << 31) | ((r1 & 0x7fffffff) >> 1)
	c.G[i.r + 0] = ((r1 & 0x1) << 31) | ((r0 & 0x7fffffff) >> 1)

	return c.IC + 2
}
func BuildRRDFunc(op, r, ix uint8, as uint16) Instruction {
	return RRD(buildType1(op, r, ix, as))
}

// SLA
type SLA type1
func (i SLA) Execute(c *CPU) uint32 {
	r := c.G[i.r]
	c.G[i.r] = (r & 0x80000000) | (((r & 0x7fffffff) << 1) & 0x7fffffff)
	c.setCC(1, c.G[i.r])
	return c.IC + 2
}
func BuildSLAFunc(op, r, ix uint8, as uint16) Instruction {
	return SLA(buildType1(op, r, ix, as))
}

// SLDA
type SLDA type1
func (i SLDA) Execute(c *CPU) uint32 {
	r0 := c.G[i.r]
	r1 := c.G[i.r + 1]

	c.G[i.r] = ((r1 & 0x80000000) >> 31) | ((r0 & 0x7fffffff) << 1)
	c.G[i.r] = (r1 & 0x7fffffff) << 1
	if c.G[i.r] == 0 {
		c.setCC(1, c.G[i.r + 1])
	} else {
		c.setCC(1, c.G[i.r])
	}

	return c.IC + 2
	
}
func BuildSLDAFunc(op, r, ix uint8, as uint16) Instruction {
	return SLDA(buildType1(op, r, ix, as))
}

// SLL
type SLL type1
func (i SLL) Execute(c *CPU) uint32 {
	r := c.G[i.r]
	c.G[i.r] = r << 1
	return c.IC + 2
}
func BuildSLLFunc(op, r, ix uint8, as uint16) Instruction {
	return SLL(buildType1(op, r, ix, as))
}

// SLDL
type SLDL type1
func (i SLDL) Execute(c *CPU) uint32 {
	r0 := c.G[i.r]
	r1 := c.G[i.r + 1]
	c.G[i.r] = ((r0 &0x7fffffff) << 1) | ((r1 & 0x80000000) << 31)
	c.G[i.r + 1] = ((r1 &0x7fffffff) << 1)
	return c.IC + 2
}
func BuildSLDLFunc(op, r, ix uint8, as uint16) Instruction {
	return SLDL(buildType1(op, r, ix, as))
}

// SRA
type SRA type1
func (i SRA) Execute(c *CPU) uint32 {
	r := c.G[i.r]
	c.G[i.r] = (r & 0x80000000) | (r >> 1)
	return c.IC + 2
}
func BuildSRAFunc(op, r, ix uint8, as uint16) Instruction {
	return SRA(buildType1(op, r, ix, as))
}

// SRDA
type SRDA type1
func (i SRDA) Execute(c *CPU) uint32 {
	r0 := c.G[i.r]
	r1 := c.G[i.r + 1]
	c.G[i.r] = (r0 & 0x80000000) | (r0 >> 1)
	c.G[i.r + 1] = ((r0 & 1) << 31) | (r1 >> 1)
	
	return c.IC + 2
}
func BuildSRDAFunc(op, r, ix uint8, as uint16) Instruction {
	return SRDA(buildType1(op, r, ix, as))
}

// SRL
type SRL type1
func (i SRL) Execute(c *CPU) uint32 {
	c.G[i.r] = c.G[i.r] >> 1
	return c.IC + 2
}
func BuildSRLFunc(op, r, ix uint8, as uint16) Instruction {
	return SRL(buildType1(op, r, ix, as))
}

// SRDL
type SRDL type1
func (i SRDL) Execute(c *CPU) uint32 {
	r0 := c.G[i.r]
	r1 := c.G[i.r + 1]
	c.G[i.r] = (r0 >> 1)
	c.G[i.r + 1] = ((r0 & 1) << 31) | (r1 >> 1)
	
	return c.IC + 2
}
func BuildSRDLFunc(op, r, ix uint8, as uint16) Instruction {
	return SRDL(buildType1(op, r, ix, as))
}

// CP
type CP type1
// It is not obvious what PCR is?

// EX
type EX type1
func (i EX) Execute(c *CPU) uint32 {
	source := c.computeEffective(i.as, i.i, i.x)
	value := c.FetchWord(source)
	return decodeWord(value).Execute(c)
}
func BuildEXFunc(op, r, ix uint8, as uint16) Instruction {
	return EX(buildType1(op, r, ix, as))
}

// JC
type JC type1
func (i JC) Execute(c *CPU) uint32 {
	check := c.CC & i.r
	if check == 0 {
		return c.IC + 2
	}
	return c.computeEffective(i.as, i.i, i.x)
}
func BuildJCFunc(op, r, ix uint8, as uint16) Instruction {
	return JC(buildType1(op, r, ix, as))
}

// JOS
type JOS type1
func (i JOS) Execute(c *CPU) uint32 {
	c.G[i.r] -= 1
	if c.G[i.r] == 0 {
		return c.IC + 2
	}
	return c.computeEffective(i.as, i.i, i.x)
}
func BuildJOSFunc(op, r, ix uint8, as uint16) Instruction {
	return JOS(buildType1(op, r, ix, as))
}

// JTS
type JTS type1
func (i JTS) Execute(c *CPU) uint32 {
	c.G[i.r] -= 2
	if c.G[i.r] == 0 {
		return c.IC + 2
	}
	return c.computeEffective(i.as, i.i, i.x)
}
func BuildJTSFunc(op, r, ix uint8, as uint16) Instruction {
	return JTS(buildType1(op, r, ix, as))
}


// JOA
type JOA type1
func (i JOA) Execute(c *CPU) uint32 {
	c.G[i.r] += 1
	if c.G[i.r] == 0 {
		return c.IC + 2
	}
	return c.computeEffective(i.as, i.i, i.x)
}
func BuildJOAFunc(op, r, ix uint8, as uint16) Instruction {
	return JOA(buildType1(op, r, ix, as))
}

// JS
type JS type1
func (i JS) Execute(c *CPU) uint32 {
	c.G[i.r] = c.IC + 2
	return c.computeEffective(i.as, i.i, i.x)
}
func BuildJSFunc(op, r, ix uint8, as uint16) Instruction {
	return JS(buildType1(op, r, ix, as))
}

// JSP

// LSK

// LPC

// NOP
type NOP type1
func (i NOP) Execute(c *CPU) uint32 {
	return c.IC + 2
}
func BuildNOPFunc(op, r, ix uint8, as uint16) Instruction {
	return NOP(buildType1(op, r, ix, as))
}

// STSR
