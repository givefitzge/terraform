// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

// Workspace represents the command-line arguments common between all workspace subcommands.
//
// Subcommands that accept additional arguments should have a specific struct that embeds this struct.
type Workspace struct {
	// ViewType specifies which output format to use
	ViewType ViewType
}
