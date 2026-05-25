#ifndef FZ_MODULE_H
#define FZ_MODULE_H

typedef void (*fz_entry_t)(void*);

typedef struct {
    const char* name;
    int version;
    fz_entry_t entry;
} fz_module_info;

#endif
