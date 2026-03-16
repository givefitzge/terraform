// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/ast"
)

func TestParseMigration(t *testing.T) {
	input := []byte(`{
		"name": "v3to4/rename_s3_bucket_object",
		"description": "Rename aws_s3_bucket_object to aws_s3_object",
		"match": {
			"block_type": "resource",
			"label": "aws_s3_bucket_object"
		},
		"actions": [
			{"action": "rename_resource", "to": "aws_s3_object"}
		]
	}`)

	got, err := ParseMigration(input)
	if err != nil {
		t.Fatal(err)
	}

	want := &Migration{
		Name:        "v3to4/rename_s3_bucket_object",
		Description: "Rename aws_s3_bucket_object to aws_s3_object",
		Match: Match{
			BlockType: "resource",
			Label:     "aws_s3_bucket_object",
		},
		Actions: []Action{
			{Action: "rename_resource", To: "aws_s3_object"},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ParseMigration mismatch (-want +got):\n%s", diff)
	}
}

func TestParseMigration_allActions(t *testing.T) {
	input := []byte(`{
		"name": "v4to5/multi_action",
		"description": "Test all action types",
		"match": {"block_type": "resource", "label": "aws_instance"},
		"actions": [
			{"action": "rename_attribute", "from": "ami", "to": "image_id"},
			{"action": "remove_attribute", "name": "vpc"},
			{"action": "rename_resource", "to": "aws_ec2_instance"},
			{"action": "add_comment", "text": "FIXME: check manually"},
			{"action": "set_attribute_value", "name": "engine", "value": "mysql"},
			{"action": "add_attribute", "name": "engine", "value": "aurora"},
			{"action": "replace_value", "name": "enabled", "old_value": "true", "new_value": "\"Enabled\""}
		]
	}`)

	got, err := ParseMigration(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Actions) != 7 {
		t.Fatalf("expected 7 actions, got %d", len(got.Actions))
	}
	if got.Actions[0].Action != "rename_attribute" || got.Actions[0].From != "ami" || got.Actions[0].To != "image_id" {
		t.Errorf("action 0: %+v", got.Actions[0])
	}
	if got.Actions[6].Action != "replace_value" || got.Actions[6].OldValue != "true" || got.Actions[6].NewValue != `"Enabled"` {
		t.Errorf("action 6: %+v", got.Actions[6])
	}
}

func TestParseMigration_invalidJSON(t *testing.T) {
	_, err := ParseMigration([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseMigration_missingName(t *testing.T) {
	input := []byte(`{
		"match": {"block_type": "resource", "label": "test"},
		"actions": [{"action": "remove_attribute", "name": "foo"}]
	}`)
	_, err := ParseMigration(input)
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestDiscoverMigrations(t *testing.T) {
	dir := t.TempDir()

	v3to4 := filepath.Join(dir, "v3to4")
	os.MkdirAll(v3to4, 0755)

	writeJSON(t, filepath.Join(v3to4, "rename_s3.json"), &Migration{
		Name:        "v3to4/rename_s3",
		Description: "Rename S3",
		Match:       Match{BlockType: "resource", Label: "aws_s3_bucket_object"},
		Actions:     []Action{{Action: "rename_resource", To: "aws_s3_object"}},
	})
	writeJSON(t, filepath.Join(v3to4, "remove_classic.json"), &Migration{
		Name:        "v3to4/remove_classic",
		Description: "Remove EC2-Classic",
		Match:       Match{BlockType: "resource", Label: "aws_instance"},
		Actions:     []Action{{Action: "remove_attribute", Name: "vpc_classic_link_id"}},
	})

	// Non-JSON file should be ignored
	os.WriteFile(filepath.Join(v3to4, "readme.txt"), []byte("ignore"), 0644)

	migrations, err := DiscoverMigrations(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}

	names := make([]string, len(migrations))
	for i, m := range migrations {
		names[i] = m.Name
	}
	if names[0] != "v3to4/remove_classic" || names[1] != "v3to4/rename_s3" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestFilterMigrations(t *testing.T) {
	all := []*Migration{
		{Name: "v3to4/rename_s3"},
		{Name: "v3to4/remove_classic"},
		{Name: "v4to5/rename_elasticache"},
	}

	got := FilterMigrations(all, "v3to4/*")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}

	got = FilterMigrations(all, "v4to5/*")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}

	got = FilterMigrations(all, "")
	if len(got) != 3 {
		t.Fatalf("expected 3 (no filter), got %d", len(got))
	}
}

func writeJSON(t *testing.T, path string, m *Migration) {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_renameAttribute(t *testing.T) {
	src := `resource "aws_instance" "example" {
  ami           = "abc-123"
  instance_type = "t2.micro"
}
`
	m := &Migration{
		Name:  "test/rename",
		Match: Match{BlockType: "resource", Label: "aws_instance"},
		Actions: []Action{
			{Action: "rename_attribute", From: "ami", To: "image_id"},
		},
	}

	got := executeMigration(t, m, src)
	want := `resource "aws_instance" "example" {
  image_id      = "abc-123"
  instance_type = "t2.micro"
}
`
	if got != want {
		t.Errorf("mismatch\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestExecute_removeAttribute(t *testing.T) {
	src := `resource "aws_instance" "example" {
  ami                 = "abc-123"
  vpc_classic_link_id = "vpc-123"
  instance_type       = "t2.micro"
}
`
	m := &Migration{
		Name:  "test/remove",
		Match: Match{BlockType: "resource", Label: "aws_instance"},
		Actions: []Action{
			{Action: "remove_attribute", Name: "vpc_classic_link_id"},
		},
	}

	got := executeMigration(t, m, src)
	if strings.Contains(got, "vpc_classic_link_id") {
		t.Errorf("expected vpc_classic_link_id to be removed, got:\n%s", got)
	}
}

func TestExecute_renameResource(t *testing.T) {
	src := `resource "aws_s3_bucket_object" "example" {
  bucket = "my-bucket"
  key    = "my-key"
}

output "obj_id" {
  value = aws_s3_bucket_object.example.id
}
`
	m := &Migration{
		Name:  "test/rename_resource",
		Match: Match{BlockType: "resource", Label: "aws_s3_bucket_object"},
		Actions: []Action{
			{Action: "rename_resource", To: "aws_s3_object"},
		},
	}

	got := executeMigration(t, m, src)
	if !strings.Contains(got, `resource "aws_s3_object" "example"`) {
		t.Errorf("expected resource type renamed, got:\n%s", got)
	}
	if !strings.Contains(got, "aws_s3_object.example.id") {
		t.Errorf("expected references renamed, got:\n%s", got)
	}
}

func TestExecute_addComment(t *testing.T) {
	src := `resource "aws_db_security_group" "example" {
  name = "my-sg"
}
`
	m := &Migration{
		Name:  "test/comment",
		Match: Match{BlockType: "resource", Label: "aws_db_security_group"},
		Actions: []Action{
			{Action: "add_comment", Text: "FIXME: aws_db_security_group is removed in v5"},
		},
	}

	got := executeMigration(t, m, src)
	if !strings.Contains(got, "# FIXME: aws_db_security_group is removed in v5") {
		t.Errorf("expected comment added, got:\n%s", got)
	}
}

func TestExecute_setAttributeValue(t *testing.T) {
	src := `resource "aws_rds_cluster" "example" {
  cluster_identifier = "my-cluster"
}
`
	m := &Migration{
		Name:  "test/set_value",
		Match: Match{BlockType: "resource", Label: "aws_rds_cluster"},
		Actions: []Action{
			{Action: "set_attribute_value", Name: "engine", Value: "aurora"},
		},
	}

	got := executeMigration(t, m, src)
	if !strings.Contains(got, "engine") || !strings.Contains(got, "aurora") {
		t.Errorf("expected engine attribute set, got:\n%s", got)
	}
}

func TestExecute_addAttribute_onlyIfMissing(t *testing.T) {
	src := `resource "aws_rds_cluster" "existing" {
  engine = "aurora-mysql"
}

resource "aws_rds_cluster" "missing" {
  cluster_identifier = "my-cluster"
}
`
	m := &Migration{
		Name:  "test/add_attr",
		Match: Match{BlockType: "resource", Label: "aws_rds_cluster"},
		Actions: []Action{
			{Action: "add_attribute", Name: "engine", Value: "aurora"},
		},
	}

	got := executeMigration(t, m, src)
	if !strings.Contains(got, "aurora-mysql") {
		t.Errorf("expected existing value preserved, got:\n%s", got)
	}
	if strings.Count(got, "engine") != 2 {
		t.Errorf("expected engine to appear twice, got:\n%s", got)
	}
}

func TestExecute_replaceValue(t *testing.T) {
	src := `resource "aws_s3_bucket_versioning" "example" {
  enabled = true
}

resource "aws_s3_bucket_versioning" "other" {
  enabled = false
}
`
	m := &Migration{
		Name:  "test/replace_value",
		Match: Match{BlockType: "resource", Label: "aws_s3_bucket_versioning"},
		Actions: []Action{
			{Action: "replace_value", Name: "enabled", OldValue: "true", NewValue: `"Enabled"`},
		},
	}

	got := executeMigration(t, m, src)
	if !strings.Contains(got, `"Enabled"`) {
		t.Errorf("expected true replaced with Enabled, got:\n%s", got)
	}
	if !strings.Contains(got, "false") {
		t.Errorf("expected false to remain, got:\n%s", got)
	}
}

func TestExecute_multipleActions(t *testing.T) {
	src := `resource "aws_instance" "example" {
  ami                 = "abc-123"
  vpc_classic_link_id = "vpc-123"
}
`
	m := &Migration{
		Name:  "test/multi",
		Match: Match{BlockType: "resource", Label: "aws_instance"},
		Actions: []Action{
			{Action: "rename_attribute", From: "ami", To: "image_id"},
			{Action: "remove_attribute", Name: "vpc_classic_link_id"},
		},
	}

	got := executeMigration(t, m, src)
	if !strings.Contains(got, "image_id") {
		t.Errorf("expected ami renamed to image_id, got:\n%s", got)
	}
	if strings.Contains(got, "vpc_classic_link_id") {
		t.Errorf("expected vpc_classic_link_id removed, got:\n%s", got)
	}
}

// executeMigration is a test helper that parses HCL, runs a migration, and returns the result.
func executeMigration(t *testing.T, m *Migration, src string) string {
	t.Helper()
	f, err := ast.ParseFile([]byte(src), "test.tf", nil)
	if err != nil {
		t.Fatal(err)
	}
	mod := ast.NewModule([]*ast.File{f}, "", true, nil)
	if err := Execute(m, mod); err != nil {
		t.Fatal(err)
	}
	result := mod.Bytes()
	return string(result["test.tf"])
}
