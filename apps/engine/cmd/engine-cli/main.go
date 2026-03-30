package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	platformio "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/platform/io"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/artifact"
	compilepkg "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/compile"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/fixture"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/normalize"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/resolve"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/simulate"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/validate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		must(runValidate(os.Args[2:]))
	case "compile":
		must(runCompile(os.Args[2:]))
	case "simulate":
		must(runSimulate(os.Args[2:]))
	case "artifact":
		must(runArtifact(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("cloudfirewall engine-cli")
	fmt.Println("commands: validate, compile, simulate, artifact")
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	policyPath := fs.String("policy", "", "path to policy json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, ir, result, err := loadAndValidate(*policyPath)
	if err != nil {
		return err
	}

	fmt.Printf("policy=%s valid=%v errors=%d warnings=%d\n", ir.Metadata.Name, result.Valid, len(result.Errors), len(result.Warnings))
	for _, e := range result.Errors {
		fmt.Printf("ERROR %s: %s\n", e.Code, e.Message)
	}
	for _, w := range result.Warnings {
		fmt.Printf("WARN  %s: %s\n", w.Code, w.Message)
	}
	return nil
}

func runCompile(args []string) error {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	policyPath := fs.String("policy", "", "path to policy json")
	outPath := fs.String("out", "", "output path for compiled nftables")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, ir, result, err := loadAndValidate(*policyPath)
	if err != nil {
		return err
	}
	if !result.Valid {
		return errors.New("policy is invalid")
	}

	compiled, err := compilepkg.New().Compile(ir)
	if err != nil {
		return err
	}
	if *outPath == "" {
		fmt.Print(compiled.Content)
		return nil
	}
	return os.WriteFile(*outPath, []byte(compiled.Content), 0o644)
}

func runSimulate(args []string) error {
	fs := flag.NewFlagSet("simulate", flag.ContinueOnError)
	policyPath := fs.String("policy", "", "path to policy json")
	casePath := fs.String("case", "", "path to simulation case json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, ir, result, err := loadAndValidate(*policyPath)
	if err != nil {
		return err
	}
	if !result.Valid {
		return errors.New("policy is invalid")
	}

	cases, err := fixture.LoadSimulationCases(*casePath)
	if err != nil {
		return err
	}

	sim := simulate.New()
	for _, c := range cases.Cases {
		res, err := sim.Simulate(ir, simulate.Request{
			PolicyVersionID: ir.PolicyID,
			Direction:       c.Direction,
			SrcIP:           c.SrcIP,
			DstIP:           c.DstIP,
			Protocol:        c.Protocol,
			DstPort:         c.DstPort,
		})
		if err != nil {
			return err
		}
		fmt.Printf("case=%s verdict=%s matchedRule=%s usedDefault=%v\n", c.Name, res.Verdict, res.MatchedRuleID, res.UsedDefault)
	}
	return nil
}

func runArtifact(args []string) error {
	fs := flag.NewFlagSet("artifact", flag.ContinueOnError)
	policyPath := fs.String("policy", "", "path to policy json")
	outPath := fs.String("out", "", "output path for artifact json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	authoring, ir, result, err := loadAndValidate(*policyPath)
	if err != nil {
		return err
	}
	if !result.Valid {
		return errors.New("policy is invalid")
	}

	compiled, err := compilepkg.New().Compile(ir)
	if err != nil {
		return err
	}
	art := artifact.NewFromCompilation(authoring, ir, compiled, "0.1.0")

	b, err := platformio.MarshalJSON(art)
	if err != nil {
		return err
	}
	if *outPath == "" {
		fmt.Print(string(b))
		return nil
	}
	return os.WriteFile(*outPath, b, 0o644)
}

func loadAndValidate(policyPath string) (model.AuthoringPolicyModel, model.PolicyVersionIR, validate.Result, error) {
	if policyPath == "" {
		return model.AuthoringPolicyModel{}, model.PolicyVersionIR{}, validate.Result{}, errors.New("--policy is required")
	}

	authoring, err := fixture.LoadAuthoringPolicy(policyPath)
	if err != nil {
		return model.AuthoringPolicyModel{}, model.PolicyVersionIR{}, validate.Result{}, err
	}
	norm, err := normalize.New().Normalize(authoring)
	if err != nil {
		return model.AuthoringPolicyModel{}, model.PolicyVersionIR{}, validate.Result{}, err
	}

	ctx := resolve.ResolutionContext{
		EnvironmentID: norm.EnvironmentID,
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
	}

	resolved := resolve.New().Resolve(norm, ctx)
	if len(resolved.Errors) > 0 {
		vr := validate.NewResult()
		vr.Valid = false
		for _, msg := range resolved.Errors {
			vr.Errors = append(vr.Errors, validate.Message{Code: validate.CodeUnknownObjectReference, Message: msg})
		}
		for _, msg := range resolved.Warnings {
			vr.Warnings = append(vr.Warnings, validate.Message{Code: "RESOLUTION_WARNING", Message: msg})
		}
		return authoring, resolved.Policy, vr, nil
	}

	structural := validate.NewStructural().Validate(resolved.Policy)
	semantic := validate.NewSemantic().Validate(resolved.Policy)
	return authoring, resolved.Policy, validate.Merge(structural, semantic), nil
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
