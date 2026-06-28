/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

#ifndef FZ_MODULE_H
#define FZ_MODULE_H

typedef void (*fz_entry_t)(void*);

typedef struct {
    const char* name;
    int version;
    fz_entry_t entry;
} fz_module_info;

typedef struct {
    const char* plugin_path;
    const char* config_path;
    const char* source_path;
    const char* dir_path;
    const char* out_bin;
    const char* out_obj;
    const char* build_type;
    const char* target;
    const char* toolchain;
    const char* mode;
    const char* cc_flags;
    const char* ld_flags;
    const char* format;
    const char* isolation;
    const char** source_dirs;
    int source_dir_count;
} fz_context_t;

#endif
