## 1.6.0 (Unreleased)

NOTE: This is a fork of [hashicorp/terraform](https://github.com/hashicorp/terraform). See [previous releases](.changes/previous-releases.md) for the upstream changelog history.

> **Personal fork** – I'm using this to experiment with Terraform internals and learn how the plan/apply cycle works. Changes here are not intended for upstream submission.

## Bug Fixes

* No changes yet.

## Enhancements

* No changes yet.

## Personal Notes

* Added extra verbosity to plan output during `terraform plan` for easier debugging of resource diffs.
* Exploring how the graph walk works in `internal/terraform/context_plan.go`.
* Increased default parallelism from 10 to 20 in `internal/command/meta_backend.go` for faster local applies on my machine.
* Noted that `terraform graph` output can be piped to `dot -Tsvg` for a quick visual — added a reminder comment near the graph command entrypoint.
* Added a local alias `tf` in shell profile pointing to this fork's binary so it doesn't conflict with the system-installed `terraform`.

---

For information on v1.5 and earlier releases, see [CHANGELOG-v1.5.md](CHANGELOG-v1.5.md).
