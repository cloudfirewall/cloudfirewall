package normalize

import (
	"sort"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/internal/policy/model"
)

type Normalizer interface {
	Normalize(in model.AuthoringPolicyModel) (model.AuthoringPolicyModel, error)
}

type DefaultNormalizer struct{}

func New() Normalizer {
	return DefaultNormalizer{}
}

func (n DefaultNormalizer) Normalize(in model.AuthoringPolicyModel) (model.AuthoringPolicyModel, error) {
	out := in
	for i := range out.Rules {
		r := &out.Rules[i]
		r.Direction = model.Direction(strings.ToUpper(string(r.Direction)))
		r.Action = model.Verdict(strings.ToUpper(string(r.Action)))
		r.Service.Protocol = model.Protocol(strings.ToUpper(string(r.Service.Protocol)))
		sort.Ints(r.Service.Ports)
	}
	sort.SliceStable(out.Rules, func(i, j int) bool {
		if out.Rules[i].OrderIndex == out.Rules[j].OrderIndex {
			return out.Rules[i].ID < out.Rules[j].ID
		}
		return out.Rules[i].OrderIndex < out.Rules[j].OrderIndex
	})
	return out, nil
}
