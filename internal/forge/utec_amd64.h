/*
 *   Copyright (c) 2026 forgezero-cli
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

#define UTEC_KIND 0
#define UTEC_FLAGS 2
#define UTEC_ID 4
#define UTEC_SRC_PTR 8
#define UTEC_DST_PTR 16
#define UTEC_LEN 24
#define UTEC_AUX1 32
#define UTEC_AUX2 40
#define UTEC_RET_VAL 48
#define UTEC_AUX3 56
#define UTEC_SIZE 64
#define DRIVER_IO_SQE 0
#define DRIVER_IO_ARRAY_SLOT 8
#define SYS_CHDIR 80
#define SYS_MOUNT 165
#define SYS_UMOUNT2 166
#define SYS_PIVOT_ROOT 155
#define SYS_CHROOT 161
#define SYS_UNSHARE 272