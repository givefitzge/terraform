// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WorkspaceSelect represent flags and arguments specific to the `terraform workspace new` command.
type WorkspaceSelect struct {
	Workspace

	// Flags
	OrCreate bool

	// Positional arguments
	Name string
}

// ParseWorkspaceSelect processes CLI arguments, returning a WorkspaceSelect value and errors.
// If errors are encountered, an WorkspaceSelect value is still returned representing
// the best effort interpretation of the arguments.
func ParseWorkspaceSelect(args []string) (*WorkspaceSelect, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var orCreate bool
	cmdFlags := defaultFlagSet("workspace select")
	cmdFlags.BoolVar(&orCreate, "or-create", false, "create workspace if it does not exist")
	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	// `workspace select` takes only one positional argument: workspace name.
	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(errors.New("Expected a single argument: NAME.")) // Recreating pre-existing error from command package
	}

	// Obtain and validate name argument, but only if there is the expected number of arguments.
	var name string
	if len(args) == 1 {
		name = args[0]
		if !ValidWorkspaceName(name) {
			diags = diags.Append(fmt.Errorf(EnvInvalidName, name))
		}
	}

	return &WorkspaceSelect{
		Workspace: Workspace{ViewType: ViewHuman},
		OrCreate:  orCreate,
		Name:      name,
	}, diags
}
