package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// other import tests can be found in context_apply2_test.go
func TestContextApply_import_in_module(t *testing.T) {
	m := testModule(t, "import-block-in-module")

	p := simpleMockProvider()
	p.ImportResourceStateFn = func(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
		return providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: "test_object",
					State: cty.ObjectVal(map[string]cty.Value{
						"test_string": cty.StringVal("importable"),
					}),
				},
			},
		}
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("importable"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	if !p.ImportResourceStateCalled {
		t.Fatal("resource not imported")
	}

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.child.test_object.bar"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}
}

func TestContextApply_import_in_nested_module(t *testing.T) { // more nested than the test above. nesteder.
	m := testModule(t, "import-block-in-nested-module")

	p := simpleMockProvider()
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("importable"),
				}),
			},
		},
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("importable"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.child.module.kinder.test_object.bar"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("resources not imported")
	}
}

func TestContextApply_import_in_expanded_module(t *testing.T) { // count AND for each!
	m := testModule(t, "import-block-in-module-with-expansion")

	p := simpleMockProvider()
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_object",
				State: cty.ObjectVal(map[string]cty.Value{
					"test_string": cty.StringVal("importable"),
				}),
			},
		},
	}
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.ObjectVal(map[string]cty.Value{
			"test_string": cty.StringVal("importable"),
		}),
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.NewState(), &PlanOpts{
		Mode: plans.NormalMode,
	})
	tfdiags.AssertNoErrors(t, diags)

	state, diags := ctx.Apply(plan, m, nil)
	tfdiags.AssertNoErrors(t, diags)

	rs := state.ResourceInstance(mustResourceInstanceAddr("module.count_child[0].test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	rs = state.ResourceInstance(mustResourceInstanceAddr("module.count_child[1].test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	rs = state.ResourceInstance(mustResourceInstanceAddr("module.for_each_child[\"a\"].test_object.foo"))
	if rs == nil {
		t.Fatal("imported resource not found in module")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("resources not imported")
	}
}
