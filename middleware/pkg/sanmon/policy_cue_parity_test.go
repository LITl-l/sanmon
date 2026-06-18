package sanmon

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// agentPolicyCUEDir is the CUE package that mirrors StarterAgentPolicy(),
// relative to this package's directory (middleware/pkg/sanmon).
const agentPolicyCUEDir = "../../../policy/domains/agent"

// loadCUEStarterAgentPolicy builds the agent CUE package and decodes its
// `policy` value into an AgentPolicy.
func loadCUEStarterAgentPolicy(t *testing.T) AgentPolicy {
	t.Helper()
	insts := load.Instances([]string{"."}, &load.Config{Dir: agentPolicyCUEDir})
	if len(insts) == 0 {
		t.Fatalf("no CUE instances loaded from %s", agentPolicyCUEDir)
	}
	if err := insts[0].Err; err != nil {
		t.Fatalf("loading CUE: %v", err)
	}
	val := cuecontext.New().BuildInstance(insts[0])
	if err := val.Err(); err != nil {
		t.Fatalf("building CUE: %v", err)
	}
	var p AgentPolicy
	if err := val.LookupPath(cue.ParsePath("policy")).Decode(&p); err != nil {
		t.Fatalf("decoding CUE policy into AgentPolicy: %v", err)
	}
	return p
}

// TestStarterAgentPolicyMatchesCUE enforces the invariant the source comments
// only assert in prose: StarterAgentPolicy() in Go and the starter policy in
// policy/domains/agent/policy.cue must stay identical. This makes CUE an
// enforceable source of truth rather than a hand-maintained mirror that can
// silently drift.
func TestStarterAgentPolicyMatchesCUE(t *testing.T) {
	want := StarterAgentPolicy()
	got := loadCUEStarterAgentPolicy(t)
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("StarterAgentPolicy() (Go) and policy/domains/agent/policy.cue diverged.\n"+
			"Update whichever is stale so they match (-Go +CUE):\n%s", diff)
	}
}
