.section .data
msg: .ascii "Hello from GAS!\n"
len = . - msg

.section .text
.global _start

_start:
    mov $1, %rax
    mov $1, %rdi
    mov $msg, %rsi
    mov $len, %rdx
    syscall

    mov $60, %rax
    xor %rdi, %rdi
    syscall