package main

import (
	"errors"
	"fmt"
	"github.com/Juniper/apstra-go-sdk/apstra"
	"io"
	"strconv"
	"strings"
)

type singleVlan struct {
	Vlan   apstra.Vlan `json:"vlan"`
	Tagged bool        `json:"tagged"`
}

func singleVlanFromVn(vn *apstra.VirtualNetwork, ctId apstra.ObjectId, skipped io.StringWriter) (*apstra.Vlan, error) {
	vlans := make(map[apstra.Vlan]struct{})
	for i, binding := range vn.Data.VnBindings {
		if binding.VlanId == nil {
			return nil, fmt.Errorf("CT %q: VN %q: binding %d has nil VLAN ID", ctId, vn.Id, i)
		}
		vlans[*binding.VlanId] = struct{}{}
	}

	switch len(vlans) {
	case 0:
		msg := fmt.Sprintf("%s: no associated VLANs ", ctId)
		if skipped == nil {
			return nil, errors.New(msg)
		} else {
			_, err := skipped.WriteString(msg)
			return nil, err
		}
	case 1:
		for k := range vlans {
			return &k, nil // return on first/only VLAN
		}
	default:
		vids := make([]string, len(vlans))
		var i int
		for vid := range vlans {
			vids[i] = strconv.Itoa(int(vid))
			i++
		}

		msg := fmt.Sprintf("%s: multiple VLANs: [%s]", ctId, strings.Join(vids, ", "))

		if skipped == nil {
			return nil, errors.New(msg)
		} else {
			_, err := skipped.WriteString(msg + "\n")
			return nil, err
		}
	}
	panic("this should never happen")
}

//func singleVlanCtVlanId(attributes *apstra.ConnectivityTemplatePrimitiveAttributesAttachSingleVlan, vn *apstra.VirtualNetwork, ctId apstra.ObjectId, skipped *bytes.Buffer) (*apstra.Vlan, error) {
//	if attributes.VnNodeId == nil {
//		return nil, errors.New("VnNodeId is nil")
//	}
//
//	vlans := make(map[apstra.Vlan]struct{})
//	for i, binding := range vn.Data.VnBindings {
//		if binding.VlanId == nil {
//			return nil, fmt.Errorf("CT %q: VN %q: binding %d has nil VLAN ID", ctId, vn.Id, i)
//		}
//		vlans[*binding.VlanId] = struct{}{}
//	}
//
//	switch len(vlans) {
//	case 0:
//		skipped.WriteString(fmt.Sprintf("%s: no associated VLANs ", ctId))
//		return nil, nil
//	case 1:
//		for k := range vlans {
//			return &k, nil // return on first/only VLAN
//		}
//	}
//
//	vids := make([]string, len(vlans))
//	var i int
//	for vid := range vlans {
//		vids[i] = strconv.Itoa(int(vid))
//	}
//	skipped.WriteString(fmt.Sprintf("%s: multiple VLANs: [%s]\n", ctId, strings.Join(vids, ", ")))
//	return nil, nil
//}

func handleAccessSingleVlanCt(ct apstra.ConnectivityTemplate, skipped io.StringWriter, vnMap map[apstra.ObjectId]apstra.VirtualNetwork) (*singleVlan, error) {
	if len(ct.Subpolicies) != 1 {
		return nil, fmt.Errorf("handleAccessSingleVlanCt: CT %q has %d subpolicies", ct.Id, len(ct.Subpolicies))
	}

	sv := ct.Subpolicies[0]
	if sv.Attributes.PolicyTypeName() != apstra.CtPrimitivePolicyTypeNameAttachSingleVlan {
		return nil, fmt.Errorf("wrong primitive type in handleAccessSingleVlanCt: %s", sv.Attributes.PolicyTypeName())
	}

	attributes := sv.Attributes.(*apstra.ConnectivityTemplatePrimitiveAttributesAttachSingleVlan)
	if attributes.VnNodeId == nil {
		return nil, fmt.Errorf("CT %q: VnNodeId is nil", ct.Id)
	}

	svAttributes := *sv.Attributes.(*apstra.ConnectivityTemplatePrimitiveAttributesAttachSingleVlan)
	if svAttributes.VnNodeId == nil {
		_, err := skipped.WriteString(fmt.Sprintf("%s: VN ID is null", ct.Id))
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	var vn apstra.VirtualNetwork
	var ok bool
	if vn, ok = vnMap[*svAttributes.VnNodeId]; !ok {
		_, err := skipped.WriteString(fmt.Sprintf("%s: virtual network %q not found", ct.Id, *svAttributes.VnNodeId))
		if err != nil {
			return nil, err
		}
	}

	vlan, err := singleVlanFromVn(&vn, *ct.Id, skipped)
	if err != nil {
		return nil, err
	}

	if vlan == nil {
		// skipped
		return nil, nil
	}

	return &singleVlan{
		Tagged: svAttributes.Tagged,
		Vlan:   *vlan,
	}, nil
}
