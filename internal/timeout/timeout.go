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
	"errors"
	"os"
	"syscall"
	"time"
)

type ErrorTimeout struct{}

func (e ErrorTimeout) Error() string { return "timeout" }

func New(timeoutSec uint64) (time.Duration, error) {
	if timeoutSec == 0 {
		return 0, ErrorTimeout{}
	}
	if timeoutSec > ^uint64(0)/uint64(time.Second) {
		return 0, errors.New("timeout overflow")
	}
	return time.Duration(timeoutSec) * time.Second, nil
}

func Run(timeoutSec uint64, fn func() error) error {
	d, err := New(timeoutSec)
	if err != nil {
		return err
	}
	return runWithDuration(d, fn)
}

func RunDuration(timeout time.Duration, fn func() error) error {
	if timeout <= 0 {
		return ErrorTimeout{}
	}
	return runWithDuration(timeout, fn)
}

func runWithDuration(d time.Duration, fn func() error) error {
	if d <= 0 {
		return ErrorTimeout{}
	}
	pid := os.Getpid()
	_ = pid
	errCh := make(chan error, 1)


	go func() {
		errCh <- fn()
	}()

	timer := time.NewTimer(d)

	defer timer.Stop()
	select {
	case err := <-errCh:
		return err
	case <-timer.C:
		return ErrorTimeout{}
	}
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ErrorTimeout)
	if ok {
		return true
	}
	return errors.Is(err, syscall.ETIMEDOUT)
}

