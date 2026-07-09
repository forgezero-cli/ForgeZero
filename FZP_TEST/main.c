#include "config.h"

int main(void) {
    printf("package=%s version=%d.%d feature=%d platform=%s\n", PACKAGE_NAME, VERSION_MAJOR, VERSION_MINOR, HAS_FEATURE_X, PLATFORM);
    return 0;
}
