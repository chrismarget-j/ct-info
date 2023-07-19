package main

import (
	"fmt"
	"github.com/Juniper/apstra-go-sdk/apstra"
	"sort"
	"strings"
)

func compareMaps(a map[apstra.ObjectId]apstra.ConnectivityTemplate, b map[apstra.ObjectId]apstra.ConnectivityTemplateState) error {
	if len(a) != len(b) {
		return fmt.Errorf("%d templates and %d template states", len(a), len(b))
	}
	var i int
	aIds := make([]string, len(a))
	bIds := make([]string, len(b))

	i = 0
	for k := range a {
		aIds[i] = string(k)
		i++
	}
	sort.Strings(aIds)

	i = 0
	for k := range b {
		bIds[i] = string(k)
		i++
	}
	sort.Strings(bIds)

	for i = 0; i < len(a); i++ {
		if aIds[i] != bIds[i] {
			return fmt.Errorf("mismatch in sorted CT and CT state IDs at inded %d: %q vs. %q", i, aIds[i], bIds[i])
		}
	}

	return nil
}

func joinIds(in []apstra.ObjectId, s string) string {
	ids := make([]string, len(in))
	for i := range in {
		ids[i] = string(in[i])
	}

	return strings.Join(ids, s)
}
