/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"context"
)

func PreloadCache(ctx context.Context, cacheDir string) error {
	if cacheDir == "" || ctx.Err() != nil {
		return ctx.Err()
	}
	if _, loaded := preloadStart.LoadOrStore(cacheDir, struct{}{}); loaded {
		return nil
	}
	preloadWait.Add(1)
	go func() {
		preloadActionCache(cacheDir)
		preloadWait.Done()
	}()
	return nil
}
