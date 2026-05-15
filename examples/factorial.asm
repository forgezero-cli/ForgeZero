; factorial_linux.asm
; NASM 64-bit, Linux syscall version

global _start

section .data
    msg db "Factorial of 5 is: "
    len equ $ - msg
    newline db 10

section .bss
    buffer resb 16   
section .text
_start:
    mov edi, 5
    call factorial      

    ; buffer
    mov rdi, buffer
    mov rsi, rax
    call int_to_str

    ; write(1, msg, len)
    mov eax, 1
    mov edi, 1
    mov rsi, msg
    mov edx, len
    syscall

    ; write(1, buffer, strlen)
    mov eax, 1
    mov edi, 1
    mov rsi, buffer
    mov edx, ecx
    syscall

    mov eax, 1
    mov edi, 1
    mov rsi, newline
    mov edx, 1
    syscall

    mov eax, 60
    xor edi, edi
    syscall

factorial:
    cmp edi, 1
    jle .base
    push rdi
    dec edi
    call factorial
    pop rdi
    imul rax, rdi
    ret
.base:
    mov eax, 1
    ret

int_to_str:
    mov rbx, 10
    mov rcx, rdi       
    add rdi, 15
    mov byte [rdi], 0     
    dec rdi
    test rsi, rsi
    jnz .loop
    mov byte [rdi], '0'
    mov ecx, 1
    mov rdi, rcx
    ret
.loop:
    xor rdx, rdx
    mov rax, rsi
    div rbx
    mov rsi, rax
    add dl, '0'
    mov [rdi], dl
    dec rdi
    test rsi, rsi
    jnz .loop
    inc rdi
    mov rdx, rcx
    sub rdx, rdi         
    mov rcx, rdi
    mov rdi, rdx
    push rdi
    mov rax, rdi
    sub rax, rcx
    neg rax
    mov ecx, eax
    pop rdi
    ret
