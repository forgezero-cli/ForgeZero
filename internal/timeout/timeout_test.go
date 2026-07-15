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

package timeout

import (
	"testing"
	"time"
)

func TestRunTimeout_Timeout(t *testing.T) {
	start := time.Now()
	err := Run(1, func() error {
		time.Sleep(2 * time.Second)
		return nil
	})
	if !IsTimeout(err) {
		t.Fatalf("expected timeout, got %v", err)
	}
	if time.Since(start) < time.Second {
		t.Fatalf("timeout fired too early")
	}
}

func TestRunTimeout_Success(t *testing.T) {
	err := Run(2, func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestRunDurationTimeout_Timeout(t *testing.T) {
	if !IsTimeout(RunDuration(50*time.Millisecond, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})) {
		t.Fatalf("expected timeout")
	}
}
