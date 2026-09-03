package controls

import (
	"errors"
	"fmt"

	ber "github.com/go-asn1-ber/asn1-ber"
	"github.com/go-ldap/ldap/v3"
)

// The virtual list view control, implemented here because go-ldap ships
// neither the request nor the response (FR-002, plan.md § Complexity Tracking).
//
// VLV is what makes a scrollbar over a 100,000-entry container honest: paging
// walks forward, VLV jumps. The draft it comes from (draft-ietf-ldapext-ldapv3-vlv)
// never became an RFC, which is why no maintained Go library carries it.
//
//	VirtualListViewRequest ::= SEQUENCE {
//	     beforeCount    INTEGER (0..maxInt),
//	     afterCount     INTEGER (0..maxInt),
//	     target         CHOICE {
//	                      byOffset [0] SEQUENCE {
//	                           offset          INTEGER (1..maxInt),
//	                           contentCount    INTEGER (0..maxInt) },
//	                      greaterThanOrEqual [1] AssertionValue },
//	     contextID      OCTET STRING OPTIONAL }
type VLVRequest struct {
	// Before and After are how many entries either side of the target to send.
	Before, After int
	// Offset and ContentCount select the by-offset form. ContentCount is the
	// client's belief about the list size, echoed from a previous response.
	Offset, ContentCount int
	// GreaterThanOrEqual selects the by-value form. When set, it wins over the
	// offset form.
	GreaterThanOrEqual string
	// ContextID is echoed from the server's previous response.
	ContextID []byte
	Critical  bool
}

// GetControlType implements ldap.Control.
func (r *VLVRequest) GetControlType() string { return OIDVLVRequest }

// String implements ldap.Control.
func (r *VLVRequest) String() string {
	if r.GreaterThanOrEqual != "" {
		return fmt.Sprintf("VLV(before=%d after=%d >=%q)", r.Before, r.After, r.GreaterThanOrEqual)
	}
	return fmt.Sprintf("VLV(before=%d after=%d offset=%d/%d)", r.Before, r.After, r.Offset, r.ContentCount)
}

// Encode implements ldap.Control.
func (r *VLVRequest) Encode() *ber.Packet {
	packet := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Control")
	packet.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString,
		OIDVLVRequest, "Control Type ("+OIDVLVRequest+")"))
	if r.Critical {
		packet.AppendChild(ber.NewBoolean(ber.ClassUniversal, ber.TypePrimitive, ber.TagBoolean, true, "Criticality"))
	}

	value := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "VirtualListViewRequest")
	value.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(r.Before), "beforeCount"))
	value.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(r.After), "afterCount"))

	if r.GreaterThanOrEqual != "" {
		value.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, ber.Tag(1),
			r.GreaterThanOrEqual, "greaterThanOrEqual"))
	} else {
		byOffset := ber.Encode(ber.ClassContext, ber.TypeConstructed, ber.Tag(0), nil, "byOffset")
		byOffset.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(r.Offset), "offset"))
		byOffset.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(r.ContentCount), "contentCount"))
		value.AppendChild(byOffset)
	}
	if len(r.ContextID) > 0 {
		value.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString,
			string(r.ContextID), "contextID"))
	}

	// The control value is the DER encoding of the request, carried as an
	// octet string.
	wrapper := ber.Encode(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, nil, "Control Value (VLV)")
	wrapper.Value = string(value.Bytes())
	wrapper.Data.Write(value.Bytes())
	packet.AppendChild(wrapper)

	return packet
}

// VLVResponse is the server's answer.
//
//	VirtualListViewResponse ::= SEQUENCE {
//	     targetPosition    INTEGER (0..maxInt),
//	     contentCount      INTEGER (0..maxInt),
//	     virtualListViewResult ENUMERATED { ... },
//	     contextID         OCTET STRING OPTIONAL }
type VLVResponse struct {
	TargetPosition int
	ContentCount   int
	// ResultCode is the VLV result, which is distinct from the search's own
	// result code and is reported alongside it, never in place of it.
	ResultCode int
	ContextID  []byte
}

func (r *VLVResponse) GetControlType() string { return OIDVLVResponse }
func (r *VLVResponse) String() string {
	return fmt.Sprintf("VLVResponse(position=%d/%d result=%d)", r.TargetPosition, r.ContentCount, r.ResultCode)
}

// Encode implements ldap.Control. A response control is never sent, so this
// returns nil rather than pretending to build one.
func (r *VLVResponse) Encode() *ber.Packet { return nil }

// DecodeVLVResponse parses a VLV response control's value.
func DecodeVLVResponse(value []byte) (*VLVResponse, error) {
	packet, err := ber.DecodePacketErr(value)
	if err != nil {
		return nil, fmt.Errorf("vlv: malformed response control: %w", err)
	}
	if len(packet.Children) < 3 {
		return nil, errors.New("vlv: response control has too few fields")
	}

	target, ok1 := packet.Children[0].Value.(int64)
	count, ok2 := packet.Children[1].Value.(int64)
	code, ok3 := packet.Children[2].Value.(int64)
	if !ok1 || !ok2 || !ok3 {
		return nil, errors.New("vlv: response control fields are not integers")
	}

	resp := &VLVResponse{
		TargetPosition: int(target),
		ContentCount:   int(count),
		ResultCode:     int(code),
	}
	if len(packet.Children) > 3 {
		resp.ContextID = packet.Children[3].Data.Bytes()
	}
	return resp, nil
}

// FindVLVResponse extracts the VLV response from a search's response controls.
func FindVLVResponse(response []ldap.Control) (*VLVResponse, bool) {
	for _, c := range response {
		if c.GetControlType() != OIDVLVResponse {
			continue
		}
		if parsed, ok := c.(*VLVResponse); ok {
			return parsed, true
		}
		if raw, ok := c.(*ldap.ControlString); ok {
			if decoded, err := DecodeVLVResponse([]byte(raw.ControlValue)); err == nil {
				return decoded, true
			}
		}
	}
	return nil, false
}

// VLV result codes worth naming. The rest are surfaced numerically.
const (
	VLVSuccess                  = 0
	VLVOperationsError          = 1
	VLVUnwillingToPerform       = 53
	VLVInsufficientAccessRights = 50
	VLVOffsetRangeError         = 61
	VLVControlError             = 76
)
