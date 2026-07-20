================================================================================
X86-64 ASSEMBLY LANGUAGE REFERENCE (INTEL NASM SYNTAX)
================================================================================
Target Architecture: x86-64 (AMD64 / Intel 64)
Assembler: NASM (The Netwide Assembler)
Language: English
================================================================================

1. # REGISTERS

## 1.1 General Purpose Registers (GPR)

+---------+---------+---------+-------------+-----------------------------------+
| 64-bit | 32-bit | 16-bit | 8-bit | Purpose |
+---------+---------+---------+-------------+-----------------------------------+
| rax | eax | ax | al / ah | Return values, accumulator |
| rcx | ecx | cx | cl / ch | Loop counter (loop, shift) |
| rdx | edx | dx | dl / dh | Data, remainder of division |
| rbx | ebx | bx | bl / bh | Callee-saved register |
| rsp | esp | sp | spl | Stack pointer |
| rbp | ebp | bp | bpl | Stack frame base pointer |
| rsi | esi | si | sil | Source (arg #2 in Linux SysV) |
| rdi | edi | di | dil | Destination (arg #1 in Linux) |
| r8-r15 | r8d-15d | r8w-15w | r8b-15b | Extended registers |
+---------+---------+---------+-------------+-----------------------------------+

## 1.2 RFLAGS (Status Flags)

CF - Carry Flag (unsigned overflow/borrow)
ZF - Zero Flag (result is zero)
SF - Sign Flag (result is negative, MSB is 1)
OF - Overflow Flag (signed overflow)
AF - Auxiliary Carry Flag (half-byte carry, for BCD)
PF - Parity Flag (even number of 1s in low byte)

2. # DATA MOVEMENT

## 2.1 mov - Move Data

Syntax: mov destination, source
Flags: None affected.

+----------------+-------+-----------------------------------------+
| Form | Bytes | Notes |
+----------------+-------+-----------------------------------------+
| mov reg, imm | 2-10 | Move immediate to register |
| mov reg, reg | 2-3 | Move register to register |
| mov reg, [mem] | 3-7 | Load from memory to register |
| mov [mem], reg | 3-7 | Store register to memory |
| mov reg64,imm64| 10 | ONLY instruction with 64-bit immediate |
+----------------+-------+-----------------------------------------+

## 2.2 push / pop - Stack Operations

Syntax: push src / pop dest
push: rsp = rsp - 8; [rsp] = src
pop: dest = [rsp]; rsp = rsp + 8

+----------------+-------+-----------------------------------------+
| Instruction | Bytes | Notes |
+----------------+-------+-----------------------------------------+
| push reg | 1-2 | Push 64-bit register (4 bytes in 32-bit)|
| push imm | 2-6 | Push 8/16/32-bit immediate (sign-ext) |
| pop reg | 1-2 | Pop into 64-bit register |
+----------------+-------+-----------------------------------------+

## 2.3 lea - Load Effective Address

Syntax: lea reg, [mem]
Flags: None affected.
Calculates the memory address and stores it in the register.
Does NOT access memory. Useful for fast arithmetic (e.g., lea rax, [rbx+rcx*4+8]).

3. # ARITHMETIC

## 3.1 add / sub - Addition / Subtraction

Syntax: add dest, src / sub dest, src
Flags: CF, ZF, SF, OF, AF, PF

## 3.2 inc / dec - Increment / Decrement

Syntax: inc dest / dec dest
Flags: ZF, SF, OF, AF, PF (NOTE: CF is NOT affected!)

## 3.3 mul - Unsigned Multiplication

Syntax: mul src (implicit destination)
+---------+-------+-------------------------+-----------------------+
| Operand | Bytes | Input | Result |
+---------+-------+-------------------------+-----------------------+
| 8-bit | 1 | AL _ src | AX |
| 16-bit | 2 | AX _ src | DX:AX |
| 32-bit | 4 | EAX _ src | EDX:EAX |
| 64-bit | 8 | RAX _ src | RDX:RAX |
+---------+-------+-------------------------+-----------------------+
Flags: CF, OF (set if high half of result is not zero).

## 3.4 div - Unsigned Division

Syntax: div src (implicit dividend)
+---------+-------+-------------------------+-----------------------+
| Operand | Bytes | Dividend | Quotient / Remainder |
+---------+-------+-------------------------+-----------------------+
| 8-bit | 1 | AX | AL / AH |
| 16-bit | 2 | DX:AX | AX / DX |
| 32-bit | 4 | EDX:EAX | EAX / EDX |
| 64-bit | 8 | RDX:RAX | RAX / RDX |
+---------+-------+-------------------------+-----------------------+
WARNING: You MUST zero out rdx (xor rdx, rdx) before 64-bit div!
Otherwise, if rdx >= src, it causes Divide Error (SIGFPE).
Flags: Undefined.

## 3.5 imul / idiv - Signed Multiplication / Division

Syntax: imul reg (1-op) OR imul dest, src (2/3-op)
idiv src
Works exactly like mul/div, but treats operands as signed integers.
imul 2-operand form: imul rax, rbx (rax = rax \* rbx).

4. # LOGIC AND BITWISE OPERATIONS

## 4.1 and, or, xor, not - Logical Operations

Syntax: and dest, src
Flags: ZF, SF, PF. OF=0, CF=0. (NOTE: 'not' does NOT affect flags).
xor reg, reg is the standard way to zero a register (faster than mov reg, 0).

## 4.2 Shifts

Syntax: shl dest, count (count can be 1, cl, or imm8)
+---------+-----------+---------------------------------------------+
| Instr | Direction | Fill Bit |
+---------+-----------+---------------------------------------------+
| shl/sal | Left | 0 |
| shr | Right | 0 (Logical) |
| sar | Right | Sign bit (Arithmetic, preserves sign) |
+---------+-----------+---------------------------------------------+
Flags: CF (last bit shifted out), ZF, SF, OF, PF.

## 4.3 Rotates

Syntax: rol dest, count
+---------+-----------+---------------------------------------------+
| Instr | Direction | Description |
+---------+-----------+---------------------------------------------+
| rol | Left | Rotate left (CF gets last shifted bit) |
| ror | Right | Rotate right |
| rcl | Left | Rotate through carry left |
| rcr | Right | Rotate through carry right |
+---------+-----------+---------------------------------------------+

5. # FLOW CONTROL

## 5.1 jmp - Unconditional Jump

Syntax: jmp label
Changes RIP. Short jump (2 bytes, -128 to +127 offset), Near jump (5 bytes).

## 5.2 cmp - Compare

Syntax: cmp a, b
Subtracts b from a (a - b) and sets flags, but discards the result.
Flags: CF, ZF, SF, OF, AF, PF.

## 5.3 Conditional Jumps (jcc)

Must be preceded by cmp, test, or an arithmetic instruction.
+-------------+-------------------+---------------------------------------------+
| Instruction | Condition | Flag State |
+-------------+-------------------+---------------------------------------------+
| je / jz | Equal / Zero | ZF = 1 |
| jne / jnz | Not Equal | ZF = 0 |
| jg / jnle | Greater (signed) | ZF = 0 AND SF = OF |
| jge / jnl | Greater/Equal(sgn)| SF = OF |
| jl / jnge | Less (signed) | SF != OF |
| jle / jng | Less/Equal (sgn) | ZF = 1 OR SF != OF |
| ja / jnbe | Above (unsigned) | CF = 0 AND ZF = 0 |
| jae / jnb | Above/Equal (uns) | CF = 0 |
| jb / jnae | Below (unsigned) | CF = 1 |
| jbe / jna | Below/Equal (uns) | CF = 1 OR ZF = 1 |
| js | Sign (negative) | SF = 1 |
| jns | Not Sign | SF = 0 |
| jo / jno | Overflow / No OF | OF = 1 / OF = 0 |
+-------------+-------------------+---------------------------------------------+

## 5.4 call / ret - Subroutines

call label: push RIP; RIP = label
call reg: push RIP; RIP = reg
ret: pop RIP

6. # STRING OPERATIONS

These instructions use implicit registers: RSI (source), RDI (destination), RCX (count).
Direction Flag (DF) in RFLAGS determines if pointers increment (DF=0, cld) or decrement (DF=1, std).

## 6.1 movs - Move String

movsb (byte), movsw (word), movsd (dword), movsq (qword).
Copies [RSI] to [RDI]. Updates RSI and RDI.

## 6.2 stos - Store String

stosb / stosw / stosd / stosq.
Stores AL/AX/EAX/RAX into [RDI]. Updates RDI.

## 6.3 scas - Scan String

scasb / scasw / scasd / scasq.
Compares AL/AX/EAX/RAX with [RDI]. Updates RDI and flags.

## 6.4 cmps - Compare String

cmpsb / cmpsw / cmpsd / cmpsq.
Compares [RSI] with [RDI]. Updates RSI, RDI, and flags.

## 6.5 rep - Repeat Prefix

Syntax: rep movsb, repe scasb, repne cmpsb, etc.
Repeats the string instruction RCX times. Decrements RCX until 0.
repe/repz: repeats while ZF=1 (Equal/Zero)
repne/repnz: repeats while ZF=0 (Not Equal/Not Zero)

7. # SYSTEM INSTRUCTIONS (OS DEVELOPMENT)

## 7.1 syscall - System Call (64-bit Long Mode)

Bytes: 0F 05 (2 bytes)
Args: RAX (syscall number), RDI, RSI, RDX, R10, R8, R9.
Note: R10 is used instead of RCX because syscall overwrites RCX with RIP.
Clobbers: RCX, R11.

## 7.2 int - Software Interrupt (Legacy / Real Mode)

Syntax: int imm8 (e.g., int 0x80 for 32-bit Linux, int 0x10 for BIOS).
Pushes FLAGS, CS, IP to stack.

## 7.3 cli / sti - Interrupt Flag Control

cli: Clear Interrupt flag (disable hardware interrupts).
sti: Set Interrupt flag (enable hardware interrupts).

## 7.4 hlt - Halt

Stops the CPU until the next external interrupt is fired.
Used in OS idle loops.

## 7.5 cpuid - CPU Identification

Input: EAX (and ECX for some leaves).
Output: EAX, EBX, ECX, EDX.
Example: eax=0 returns max basic leaf and vendor string in ebx:edx:ecx.

## 7.6 rdmsr / wrmsr - Model Specific Registers

Read/Write MSR. MSR number in ECX. Value in EDX:EAX.
Requires Ring 0 (Kernel mode).

## 7.7 in / out - Port I/O

in al, dx (Read byte from port DX into AL)
out dx, al (Write byte from AL to port DX)

## 7.8 lgdt / lidt - Load Descriptor Tables

lgdt [mem] : Load Global Descriptor Table register (GDTR).
lidt [mem] : Load Interrupt Descriptor Table register (IDTR).
Required for transitioning to Protected/Long mode.

## 7.9 ltr - Load Task Register

ltr reg16 : Loads the Task Register (TR) for hardware task switching / TSS.

## 7.10 swapgs - Swap GS Base (64-bit)

Swaps user GS base with kernel GS base (IA32_KERNEL_GS_BASE).
Used on syscall entry/exit to access per-CPU kernel data.

8. # MEMORY MANAGEMENT & ATOMICS

## 8.1 Segment Registers

mov ds, ax / mov ax, ds
In 64-bit mode, segment bases for CS, SS, DS, ES are ignored (forced to 0).
FS and GS are still used (e.g., FS for Thread Local Storage in glibc).

## 8.2 xchg - Exchange

Syntax: xchg reg, reg / xchg reg, [mem]
Swaps operands. If memory is involved, it is implicitly locked (atomic).
xchg eax, eax is the old NOP, but 'nop' is preferred now.

## 8.3 cmpxchg - Compare and Exchange (CAS)

Syntax: cmpxchg dest, src
Compares RAX with dest. If equal, dest = src. If not equal, RAX = dest.
Used for lock-free algorithms. Add 'lock' prefix for multi-core atomicity.

## 8.4 lock - Bus Lock Prefix

Syntax: lock add [mem], 1
Asserts the LOCK# signal. Ensures the CPU has exclusive ownership of the
cache line for the duration of the instruction. Mandatory for multi-core sync.

9. # FPU (FLOATING POINT UNIT - x87)

Uses an 8-level register stack: st(0) to st(7). st(0) is the top of the stack.

+---------+-----------------------------------+-----------------------------+
| Instr | Purpose | Notes |
+---------+-----------------------------------+-----------------------------+
| fld | Load real/integer onto stack | Pushes to st(0) |
| fst | Store st(0) to memory/reg | Does not pop |
| fstp | Store st(0) and pop | Pops st(0) |
| fadd | Add | fadd st(0), st(1) |
| fsub | Subtract | |
| fmul | Multiply | |
| fdiv | Divide | |
| fcom | Compare st(0) with st(1)/mem | Sets FPU status flags |
| finit | Initialize FPU | Resets FPU state |
| fwait | Wait for FPU | Syncs FPU and CPU pipelines |
+---------+-----------------------------------+-----------------------------+
Note: Modern code prefers SSE/AVX over x87 FPU.

10. # SIMD: MMX, SSE, AVX

## 10.1 MMX (Legacy, 64-bit integer)

Registers: mm0 - mm7 (alias the x87 FPU stack, cannot mix easily).
Instr: paddb, paddw, paddd, paddq (packed add), pmuludq, etc.

## 10.2 SSE / SSE2 / SSE3 / SSE4 (128-bit)

Registers: xmm0 - xmm15 (128-bit).
+-------------+-----------------------------------+---------------------------+
| Instr | Purpose | Data Type |
+-------------+-----------------------------------+---------------------------+
| movdqa | Move Aligned Packed Integers | 128-bit integer |
| movdqu | Move Unaligned Packed Integers | 128-bit integer |
| movaps | Move Aligned Packed Singles | 4x 32-bit float |
| movapd | Move Aligned Packed Doubles | 2x 64-bit float |
| addps/addpd | Add Packed Singles/Doubles | Float math |
| mulps/mulpd | Multiply Packed Singles/Doubles | Float math |
| addss/addsd | Add Scalar Single/Double | Only lowest element |
| sqrtss/sqrtsd| Square Root Scalar | Float math |
+-------------+-----------------------------------+---------------------------+

## 10.3 AVX / AVX2 / AVX-512 (256-bit / 512-bit)

Registers: ymm0-ymm15 (256-bit), zmm0-zmm31 (512-bit in AVX-512).
Uses 'v' prefix for 3-operand non-destructive syntax:
vaddps ymm0, ymm1, ymm2 (ymm0 = ymm1 + ymm2, ymm1/2 unchanged).

11. # CALLING CONVENTIONS

## 11.1 System V AMD64 ABI (Linux, macOS, BSD, Solaris)

Arguments (first 6): rdi, rsi, rdx, rcx, r8, r9
Return value: rax (and rdx for 128-bit)
Callee-saved: rbx, rbp, rsp, r12, r13, r14, r15
Scratch (volatile): rax, rcx, rdx, rsi, rdi, r8, r9, r10, r11
Stack alignment: 16-byte aligned before 'call' instruction.

## 11.2 Microsoft x64 Calling Convention (Windows)

Arguments (first 4): rcx, rdx, r8, r9
Return value: rax
Shadow Space: Caller MUST allocate 32 bytes on stack before call.
Callee-saved: rbx, rbp, rdi, rsi, rsp, r12-r15
Scratch (volatile): rax, rcx, rdx, r8, r9, r10, r11

12. # REFERENCES AND FURTHER READING

1. Intel 64 and IA-32 Architectures Software Developer's Manual (SDM)
   - Volume 1: Basic Architecture
   - Volume 2A & 2B: Instruction Set Reference, A-Z (The ultimate source)
   - Volume 3: System Programming Guide (OS Dev, Paging, Protected Mode)

1. NASM Documentation
   - Chapter 3: Programmer's Instructions (NASM syntax specifics)
   - Appendix A: x86 Instruction Cross-Reference

1. Agner Fog's Optimization Guides
   - Instruction tables: Lists latencies, throughputs, and uops for all CPUs.
   - Essential for high-performance assembly optimization.

1. x86-64 Machine Level (Cheat Sheets)
   - Quick reference cards for registers, syscalls, and instructions.

================================================================================
END OF DOCUMENT
================================================================================
