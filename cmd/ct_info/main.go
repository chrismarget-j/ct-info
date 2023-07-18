package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Juniper/apstra-go-sdk/apstra"
	"github.com/chrismarget-j/ctinfo"
	"os"
	"strings"
)

var mainCtMap, accessCtMap map[apstra.ObjectId]apstra.ConnectivityTemplate

var accessSingleTagged, accessSingleUntagged, mainSingleTagged, mainSingleUntagged map[apstra.Vlan][]apstra.ObjectId

var accessVnMap, mainVnMap map[apstra.ObjectId]apstra.VirtualNetwork

func main() {
	ctx := context.Background()

	err := config(ctx)
	if err != nil {
		die(err)
	}

	mainCtMap, err = ctinfo.GetCtMap(ctx, mainBpClient)
	if err != nil {
		die(err)
	}

	mainVnMap, err = mainBpClient.GetAllVirtualNetworks(ctx)
	if err != nil {
		die(err)
	}

	accessCtMap, err = ctinfo.GetCtMap(ctx, accessBpClient)
	if err != nil {
		die(err)
	}

	accessCtStateMap, err := ctinfo.GetCtStateMap(ctx, accessBpClient)
	if err != nil {
		die(err)
	}

	accessVnMap, err = accessBpClient.GetAllVirtualNetworks(ctx)
	if err != nil {
		die(err)
	}

	err = compareMaps(accessCtMap, accessCtStateMap)
	if err != nil {
		die(err)
	}

	skipped := new(bytes.Buffer)
	for id, ct := range accessCtMap {
		// look for reasons to skip this CT
		switch {
		case accessCtStateMap[id].Status != "assigned":
			skipped.WriteString(fmt.Sprintf("%s: status %s\n", id, accessCtStateMap[id].Status))
			continue
		case len(ct.Subpolicies) != 1:
			skipped.WriteString(fmt.Sprintf("%s: root primitive count: %d\n", id, len(ct.Subpolicies)))
			continue
		}

		// type-specific CT handling
		var b []byte
		switch ct.Subpolicies[0].Attributes.PolicyTypeName() {
		case apstra.CtPrimitivePolicyTypeNameAttachSingleVlan:
			var sv *singleVlan
			sv, err = handleAccessSingleVlanCt(ct, skipped, accessVnMap)
			if err != nil {
				die(err)
			}

			switch sv.Tagged {
			case true:
				accessSingleTagged[sv.Vlan] = append(accessSingleTagged[sv.Vlan], id)
			case false:
				accessSingleUntagged[sv.Vlan] = append(accessSingleTagged[sv.Vlan], id)
			}

			b, err = json.MarshalIndent(&sv, "", "  ")
			if err != nil {
				die(fmt.Errorf("failed marshaling a singleVlan - %w", err))
			}
		default:
			skipped.WriteString(fmt.Sprintf("%s: unhandled type %s\n", id, ct.Subpolicies[0].Attributes.PolicyTypeName()))
			continue
		}

		err = fw.writeFile(string(*ct.Id), b)
		if err != nil {
			die(err)
		}
	}

	anomalies := new(bytes.Buffer)
	for id, ct := range mainCtMap {
		// look for reasons to skip this CT
		switch {
		case len(ct.Subpolicies) != 1:
			continue
		}

		// type-specific CT handling
		switch ct.Subpolicies[0].Attributes.PolicyTypeName() {
		case apstra.CtPrimitivePolicyTypeNameAttachSingleVlan:
			attributes := ct.Subpolicies[0].Attributes.(*apstra.ConnectivityTemplatePrimitiveAttributesAttachSingleVlan)
			if attributes.VnNodeId == nil {
				anomalies.WriteString(fmt.Sprintf("%s subpolicy 0 has null VN node ID\n", *ct.Id))
				continue
			}

			var vn apstra.VirtualNetwork
			var ok bool
			if vn, ok = mainVnMap[*attributes.VnNodeId]; !ok {
				die(fmt.Errorf("%s subpolicy 0 has unknown VN node ID %q", *ct.Id, *attributes.VnNodeId))
			}

			vlan, err := singleVlanFromVn(&vn, *ct.Id, nil)
			if err != nil {
				switch {
				case strings.Contains(err.Error(), "multiple VLANs"):
					anomalies.WriteString(err.Error() + "\n")
					continue
				default:
					die(err)
				}
			}
			if vlan == nil {
				panic("this should never happen")
			}

			switch attributes.Tagged {
			case true:
				mainSingleTagged[*vlan] = append(mainSingleTagged[*vlan], id)
			case false:
				mainSingleUntagged[*vlan] = append(mainSingleTagged[*vlan], id)
			}
		default:
			continue
		}
	}

	err = fw.writeFile("_access_skipped", skipped.Bytes())
	if err != nil {
		die(err)
	}

	err = fw.writeFile("_main_anomalies", anomalies.Bytes())
	if err != nil {
		die(err)
	}
}

func die(err error) {
	_, _ = os.Stderr.WriteString(err.Error() + "\n")
	os.Exit(1)
}
