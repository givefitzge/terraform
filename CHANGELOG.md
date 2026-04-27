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

---

For information on v1.5 and earlier releases, see [CHANGELOG-v1.5.md](CHANGELOG-v1.5.md).
