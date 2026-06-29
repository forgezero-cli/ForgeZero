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

package subcmd

import (
	"context"
	"os"

	"github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"
	"github.com/forgezero-cli/ForgeZero/internal/pkgman"
)

func HandlePackageManager(ctx context.Context, args []string) {
	if len(args) < 3 {
		stdio.WriteFmt(1, "%s\n", "Usage: fz pm <add|remove|list|update|catalog|search|install> [args]")
		return
	}
	subcmd := args[2]
	switch subcmd {
	case "add":
		if len(args) < 4 {
			stdio.WriteFmt(1, "%s\n", "Usage: fz pm add <repo-url> [version]")
			return
		}
		pkgURL := args[3]
		ver := ""
		if len(args) > 4 {
			ver = args[4]
		}
		if err := pkgman.Add(ctx, pkgURL, ver); err != nil {
			stdio.WriteFmt(2, "error: %v\n", err)
			os.Exit(1)
		}
	case "remove":
		if len(args) < 4 {
			stdio.WriteFmt(1, "%s\n", "Usage: fz pm remove <repo-url>")
			return
		}
		if err := pkgman.Remove(ctx, args[3]); err != nil {
			stdio.WriteFmt(2, "error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if len(args) == 3 {
			if err := pkgman.List(); err != nil {
				stdio.WriteFmt(2, "error: %v\n", err)
				os.Exit(1)
			}
		} else if args[3] == "catalog" {
			if err := pkgman.ListCatalog(ctx); err != nil {
				stdio.WriteFmt(2, "error: %v\n", err)
				os.Exit(1)
			}
		} else {
			stdio.WriteFmt(1, "%s\n", "Usage: fz pm list [catalog]")
		}
	case "update":
		if err := pkgman.Update(ctx); err != nil {
			stdio.WriteFmt(2, "error: %v\n", err)
			os.Exit(1)
		}
	case "catalog":
		if err := pkgman.ListCatalog(ctx); err != nil {
			stdio.WriteFmt(2, "error: %v\n", err)
			os.Exit(1)
		}
	case "search":
		if len(args) < 4 {
			stdio.WriteFmt(1, "%s\n", "Usage: fz pm search <keyword>")
			return
		}
		if err := pkgman.SearchCatalog(ctx, args[3]); err != nil {
			stdio.WriteFmt(2, "error: %v\n", err)
			os.Exit(1)
		}
	case "install":
		if len(args) < 4 {
			stdio.WriteFmt(1, "%s\n", "Usage: fz pm install <catalog-package-name>")
			return
		}
		if err := pkgman.InstallFromCatalog(ctx, args[3]); err != nil {
			stdio.WriteFmt(2, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		stdio.WriteFmt(1, "Unknown pm subcommand: %s\n", subcmd)
	}
}
