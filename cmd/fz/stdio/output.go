/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package stdio

import (
	"errors"
	"os"
)

func WriteOut(fd int, s string) {
	switch fd {
	case 1:
		if _, err := os.Stdout.WriteString(s); err != nil {
			_, _ = os.Stderr.WriteString("stdout write failed: ")
			_, _ = os.Stderr.WriteString(err.Error())
			_, _ = os.Stderr.WriteString("\n")
		}
	case 2:
		if _, err := os.Stderr.WriteString(s); err != nil {
		}
	default:
		f := os.NewFile(uintptr(fd), "")
		if f != nil {
			if _, err := f.WriteString(s); err != nil {
				_, _ = os.Stderr.WriteString("fd write failed: ")
				_, _ = os.Stderr.WriteString(err.Error())
				_, _ = os.Stderr.WriteString("\n")
			}
			if err := f.Close(); err != nil {
				_, _ = os.Stderr.WriteString("fd close failed: ")
				_, _ = os.Stderr.WriteString(err.Error())
				_, _ = os.Stderr.WriteString("\n")
			}
		}
	}
}

func AppendInt(dst []byte, v int64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	neg := v < 0
	var u uint64
	if neg {
		u = uint64(-(v + 1))
		u++
	} else {
		u = uint64(v)
	}
	var tmp [20]byte
	i := len(tmp)
	for u > 0 {
		i--
		tmp[i] = byte('0' + u%10)
		u /= 10
	}
	if neg {
		i--
		tmp[i] = '-'
	}
	return append(dst, tmp[i:]...)
}

func AppendUint(dst []byte, v uint64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, tmp[i:]...)
}

func AppendAny(dst []byte, v any) []byte {
	switch x := v.(type) {
	case string:
		return append(dst, x...)
	case error:
		return append(dst, x.Error()...)
	case int:
		return AppendInt(dst, int64(x))
	case int64:
		return AppendInt(dst, x)
	case uint:
		return AppendUint(dst, uint64(x))
	case uint64:
		return AppendUint(dst, x)
	case bool:
		if x {
			return append(dst, "true"...)
		}
		return append(dst, "false"...)
	default:
		return append(dst, "<unsupported>"...)
	}
}

func AppendHex(dst []byte, v uint64, upper bool) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	var tmp [32]byte
	i := len(tmp)
	for v > 0 {
		i--
		d := byte(v & 0xf)
		if d < 10 {
			tmp[i] = '0' + d
		} else if upper {
			tmp[i] = 'A' + d - 10
		} else {
			tmp[i] = 'a' + d - 10
		}
		v >>= 4
	}
	return append(dst, tmp[i:]...)
}

func FormatAppend(dst []byte, format string, a ...any) []byte {
	argIndex := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			dst = append(dst, format[i])
			continue
		}
		i++
		switch format[i] {
		case '%':
			dst = append(dst, '%')
		case 's', 'v':
			if argIndex < len(a) {
				dst = AppendAny(dst, a[argIndex])
				argIndex++
			}
		case 'd':
			if argIndex < len(a) {
				switch x := a[argIndex].(type) {
				case int:
					dst = AppendInt(dst, int64(x))
				case int64:
					dst = AppendInt(dst, x)
				case uint:
					dst = AppendUint(dst, uint64(x))
				case uint64:
					dst = AppendUint(dst, x)
				default:
					dst = AppendAny(dst, a[argIndex])
				}
				argIndex++
			}
		case 'x', 'X':
			if argIndex < len(a) {
				switch x := a[argIndex].(type) {
				case int:
					dst = AppendHex(dst, uint64(x), format[i] == 'X')
				case int64:
					dst = AppendHex(dst, uint64(x), format[i] == 'X')
				case uint:
					dst = AppendHex(dst, uint64(x), format[i] == 'X')
				case uint64:
					dst = AppendHex(dst, x, format[i] == 'X')
				default:
					dst = AppendAny(dst, a[argIndex])
				}
				argIndex++
			}
		default:
			dst = append(dst, '%')
			dst = append(dst, format[i])
		}
	}
	return dst
}

func WriteFmt(fd int, format string, a ...any) {
	var buf [4096]byte
	b := FormatAppend(buf[:0], format, a...)
	switch fd {
	case 1:
		if _, err := os.Stdout.Write(b); err != nil {
			_, _ = os.Stderr.WriteString("stdout write failed: ")
			_, _ = os.Stderr.WriteString(err.Error())
			_, _ = os.Stderr.WriteString("\n")
		}
	case 2:
		if _, err := os.Stderr.Write(b); err != nil {
		}
	default:
		f := os.NewFile(uintptr(fd), "")
		if f != nil {
			if _, err := f.Write(b); err != nil {
				_, _ = os.Stderr.WriteString("fd write failed: ")
				_, _ = os.Stderr.WriteString(err.Error())
				_, _ = os.Stderr.WriteString("\n")
			}
			if err := f.Close(); err != nil {
				_, _ = os.Stderr.WriteString("fd close failed: ")
				_, _ = os.Stderr.WriteString(err.Error())
				_, _ = os.Stderr.WriteString("\n")
			}
		}
	}
}

func WriteStdout(s string) {
	WriteOut(1, s)
}

func WriteStderr(s string) {
	WriteOut(2, s)
}

func Errorf(format string, a ...any) error {
	var buf [4096]byte
	b := FormatAppend(buf[:0], format, a...)
	return errors.New(string(b))
}
