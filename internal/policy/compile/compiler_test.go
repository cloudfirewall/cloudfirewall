package compile_test

import (
	"os"
	"strings"
	"testing"

	compilepkg "github.com/cloudfirewall/cloudfirewall/internal/policy/compile"
	"github.com/cloudfirewall/cloudfirewall/internal/policy/fixture"
	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
	"github.com/cloudfirewall/cloudfirewall/internal/policy/normalize"
	"github.com/cloudfirewall/cloudfirewall/internal/policy/resolve"
)

func TestCompilePublicWebServer(t *testing.T) {
	authoring, err := fixture.LoadAuthoringPolicy("../../../testdata/policies/public-web-server.json")
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
	golden, err := os.ReadFile("../../../testdata/compiled/public-web-server.nft.golden")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(compiled.Content) != strings.TrimSpace(string(golden)) {
		t.Fatalf("compiled output mismatch\nexpected:\n%s\nactual:\n%s", string(golden), compiled.Content)
	}
}
