package simulate_test

import (
	"testing"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/fixture"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/normalize"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/resolve"
	simulatepkg "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/simulate"
)

func TestSimulatePublicWebServerCases(t *testing.T) {
	authoring, err := fixture.LoadAuthoringPolicy("../../../testdata/policies/public-web-server.json")
	if err != nil {
		t.Fatal(err)
	}
	cases, err := fixture.LoadSimulationCases("../../../testdata/simulation/public-web-server.sim.json")
	if err != nil {
		t.Fatal(err)
	}
	norm, err := normalize.New().Normalize(authoring)
	if err != nil {
		t.Fatal(err)
	}
	resolved := resolve.New().Resolve(norm, resolve.ResolutionContext{
		EnvironmentID: authoring.EnvironmentID,
		VisibleNetworks: []model.ResolvedNetworkObject{
			{
				ID:   "obj-office",
				Name: "office-ips",
				Kind: model.NetworkObjectList,
				Values: []model.NormalizedNetworkValue{
					{Family: model.IPFamilyV4, Value: "203.0.113.0/24"},
				},
			},
		},
	})
	if len(resolved.Errors) > 0 {
		t.Fatalf("unexpected resolve errors: %v", resolved.Errors)
	}
	sim := simulatepkg.New()
	for _, c := range cases.Cases {
		res, err := sim.Simulate(resolved.Policy, simulatepkg.Request{
			PolicyVersionID: resolved.Policy.PolicyID,
			Direction:       c.Direction,
			SrcIP:           c.SrcIP,
			DstIP:           c.DstIP,
			Protocol:        c.Protocol,
			DstPort:         c.DstPort,
		})
		if err != nil {
			t.Fatalf("case %s failed: %v", c.Name, err)
		}
		if res.Verdict != c.ExpectedVerdict {
			t.Fatalf("case %s verdict mismatch: got %s want %s", c.Name, res.Verdict, c.ExpectedVerdict)
		}
		if res.MatchedRuleID != c.ExpectedRuleID {
			t.Fatalf("case %s matched rule mismatch: got %s want %s", c.Name, res.MatchedRuleID, c.ExpectedRuleID)
		}
	}
}
