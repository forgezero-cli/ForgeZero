#include "fz_module.h"
#include <stdint.h>

static void my_entry(void *ctx) {
  int64_t *p = (int64_t *)ctx;
  if (p)
    *p = *p + 1;
}

fz_module_info fz_module = {"example", 1, my_entry};
