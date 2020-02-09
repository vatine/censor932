# Censor 932 emulator

This is a best-effort implementation of the Censor 932 CPU and architecture in Go.

It is based on information from http://fht.nu/Dokument/Flygvapnet/flyg_publ_dok_rrgc_f_del%202_bilaga_1.pdf but there's a fair few assumptions made tha may or may not be correct.

## Word sizes

- Double-word: 64 bits
- Word: 32 bits
- Half-word: 16 bits

Based on general information gleaned, the smllest addressable unit is a half-word, with registers being words.

## Endinaness

The documentation, as far as I can tell, does not specify endianness. The current implementation splits a 32-bit word into two 16-bit half-words, storing the high bits at the lower address and low bits ar the higher address.

Similarly when breaking a double-word into words, the top half goes into the lower address and the lower half into the higher address.

This choice was made primarily based on how the symbolic meaning in the instruction table was written.

## Memory

The Censor 932 architecture provides for memory controllers mapping parts of the memory space to modules shared by multiple CPUs. This is why the emulated CPU interacts with memory through a MemoryPlugin abstraction, where it is possible to register a single memory instantiation to multiple CPUs (or, to multiple places in the address space of a single CPU).

However, the provided DirectMemory plugin is not suitable for this, as no locking is performed.

## I/O

For the moment, none of the I/O instructions have been implemented, due to not having enough infrmation to even make educated guesses.
