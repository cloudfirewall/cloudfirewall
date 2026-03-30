package fixture

import (
	"os"

	platformio "github.com/cloudfirewall/cloudfirewall/apps/engine/internal/platform/io"
	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

type SimulationCase struct {
	Name            string          `json:"name"`
	Direction       model.Direction `json:"direction"`
	SrcIP           string          `json:"srcIp"`
	DstIP           string          `json:"dstIp"`
	Protocol        model.Protocol  `json:"protocol"`
	DstPort         int             `json:"dstPort"`
	ExpectedVerdict model.Verdict   `json:"expectedVerdict"`
	ExpectedRuleID  string          `json:"expectedRuleId"`
}

type SimulationCases struct {
	Cases []SimulationCase `json:"cases"`
}

func LoadAuthoringPolicy(path string) (model.AuthoringPolicyModel, error) {
	var out model.AuthoringPolicyModel
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	return out, platformio.UnmarshalByExt(path, b, &out)
}

func LoadSimulationCases(path string) (SimulationCases, error) {
	var out SimulationCases
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	return out, platformio.UnmarshalByExt(path, b, &out)
}
