section .data 
  msg db 'hello from NASM!', 0xa
  len equ $ - msg 

section .text 
  global _start 

_start:
  mov rax, 1
  mov rdi, 1
  mov rsi, msg 
  mov rdx, len 
  syscall 

  mov rax, 60
  xor rdi, rdi 
  syscall
