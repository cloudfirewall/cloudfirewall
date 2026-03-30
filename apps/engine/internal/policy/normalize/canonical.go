package normalize

import (
	"net"
	"sort"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/apps/engine/internal/policy/model"
)

func NormalizeNetworkLiteral(value string) (model.NormalizedNetworkValue, bool) {
	value = strings.TrimSpace(value)
	if ip := net.ParseIP(value); ip != nil {
		if ip.To4() != nil {
			return model.NormalizedNetworkValue{Family: model.IPFamilyV4, Value: ip.String()}, true
		}
		return model.NormalizedNetworkValue{Family: model.IPFamilyV6, Value: ip.String()}, true
	}
	if _, ipNet, err := net.ParseCIDR(value); err == nil {
		if ipNet.IP.To4() != nil {
			return model.NormalizedNetworkValue{Family: model.IPFamilyV4, Value: ipNet.String()}, true
		}
		return model.NormalizedNetworkValue{Family: model.IPFamilyV6, Value: ipNet.String()}, true
	}
	return model.NormalizedNetworkValue{}, false
}

func SortServiceEntries(entries []model.ServiceEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Protocol == entries[j].Protocol {
			return entries[i].Port < entries[j].Port
		}
		return entries[i].Protocol < entries[j].Protocol
	})
}
