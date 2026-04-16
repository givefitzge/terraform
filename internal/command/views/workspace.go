// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceList view is used for the `workspace list` subcommand.
type WorkspaceList interface {
	List(selected string, list []string, diags tfdiags.Diagnostics)
	Diagnostics(tfdiags.Diagnostics)
}

// NewWorkspace returns the Workspace implementation for the given ViewType.
func NewWorkspaceList(vt arguments.ViewType, view *View) WorkspaceList {
	switch vt {
	case arguments.ViewJSON:
		return &WorkspaceJSON{
			view: view,
		}
	case arguments.ViewHuman:
		// TODO: Allow use of WorkspaceHuman here when we remove use of cli.Ui from workspace commands.
		panic("human readable output via Views is a breaking change, so this code path shouldn't be used until that's possible.")
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The WorkspaceJSON implementation renders machine-readable logs, suitable for
// integrating with other software.
type WorkspaceJSON struct {
	view *View
}

var _ WorkspaceList = (*WorkspaceJSON)(nil)

type WorkspaceListOutput struct {
	Workspaces  []WorkspaceOutput       `json:"workspaces"`
	Diagnostics []*viewsjson.Diagnostic `json:"diagnostics"`
}

type WorkspaceOutput struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
}

// List is used to log the list of present workspaces and indicate which is currently selected
func (v *WorkspaceJSON) List(current string, list []string, diags tfdiags.Diagnostics) {
	v.output(current, list, diags)
}

// Diagnostics is used to log diagnostics when an error occurs before we have a list of workspaces to show
// The JSON output of Diagnostics matches the structure of List, but with empty data.
//
// THIS COMMAND SHOULD ONLY BE USED BEFORE RETURNING A NON-ZERO ERROR CODE.
func (v *WorkspaceJSON) Diagnostics(diags tfdiags.Diagnostics) {
	// JSON output describing errors must be in the same structure as the output of List.
	// There is no list data or selected workspace to show when we're using this method.
	v.output("", nil, diags)
}

func (v *WorkspaceJSON) output(current string, list []string, diags tfdiags.Diagnostics) {
	output := WorkspaceListOutput{}

	for _, item := range list {
		workspace := WorkspaceOutput{
			Name:      item,
			IsCurrent: item == current,
		}
		output.Workspaces = append(output.Workspaces, workspace)
	}

	if output.Workspaces == nil {
		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		output.Workspaces = []WorkspaceOutput{}
	}

	configSources := v.view.configSources()
	for _, diag := range diags {
		output.Diagnostics = append(output.Diagnostics, viewsjson.NewDiagnostic(diag, configSources))
	}

	if output.Diagnostics == nil {
		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		output.Diagnostics = []*viewsjson.Diagnostic{}
	}

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(fmt.Sprintf("failed to marshal workspace list json output: %v", err))
	}

	v.view.streams.Println(string(jsonOutput))
}

// The WorkspaceHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
//
// This is currently not used, but is implemented here for when we remove use of
// workspaceListHuman in the command package.
type WorkspaceHuman struct {
	view *View
}

var _ WorkspaceList = (*WorkspaceHuman)(nil)

func (v *WorkspaceHuman) List(selected string, list []string, diags tfdiags.Diagnostics) {
	// Print diags above output
	v.view.Diagnostics(diags)

	// Print list
	if len(list) > 0 {
		var out bytes.Buffer
		for _, s := range list {
			if s == selected {
				out.WriteString("* ")
			} else {
				out.WriteString("  ")
			}
			out.WriteString(s + "\n")
		}
		v.output(out.String())
	}
}

func (v *WorkspaceHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *WorkspaceHuman) output(msg string) string {
	return v.view.colorize.Color(strings.TrimSpace(msg))
}
