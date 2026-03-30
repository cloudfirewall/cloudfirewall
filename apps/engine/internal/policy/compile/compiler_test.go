package compile_test

import (
	"os"
	"strings"
	"testing"

	compilepkg "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/compile"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/fixture"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/normalize"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/resolve"
)

func TestCompilePublicWebServer(t *testing.T) {
	testCompilePolicy(t, "public-web-server", "../../../testdata/policies/public-web-server.json", "../../../testdata/compiled/public-web-server.nft.golden")
}

func TestCompileRiskyPublicSSH(t *testing.T) {
	testCompilePolicy(t, "risky-public-ssh", "../../../testdata/policies/risky-public-ssh.json", "../../../testdata/compiled/risky-public-ssh.nft.golden")
}

func testCompilePolicy(t *testing.T, name, policyPath, goldenPath string) {
	t.Helper()

	authoring, err := fixture.LoadAuthoringPolicy(policyPath)
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
	compiled, err := compilepkg.New().Compile(resolved.Policy)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(compiled.Content) != strings.TrimSpace(string(golden)) {
		t.Fatalf("%s compiled output mismatch\nexpected:\n%s\nactual:\n%s", name, string(golden), compiled.Content)
	}
}
