#include "fz_module.h"

void fz_init_module(void *ctx) {
  fz_context_t *context = (fz_context_t *)ctx;
  if (!context) {
    return;
  }
  (void)context;
}
