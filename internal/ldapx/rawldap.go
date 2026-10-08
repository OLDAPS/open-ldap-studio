package ldapx

import (
	"errors"
	"fmt"
	"io"

	ber "github.com/go-asn1-ber/asn1-ber"
)

// A minimal LDAP message layer, used for the two exchanges that must happen
// before go-ldap takes ownership of the socket: StartTLS, and the SASL
// mechanisms go-ldap does not implement.
//
// go-ldap's request plumbing is unexported — its Request interface has an
// unexported method — so a custom bind cannot be sent through its Conn. Rather
// than fork the library, those exchanges are performed here, on the raw
// socket, before the client is started.
const (
	appBindRequest      = ber.Tag(0)
	appBindResponse     = ber.Tag(1)
	appExtendedRequest  = ber.Tag(23)
	appExtendedResponse = ber.Tag(24)

	ldapVersion3 = 3

	// Context tags inside a BindRequest / BindResponse.
	tagSASLAuthentication = ber.Tag(3)
	tagServerSASLCreds    = ber.Tag(7)
	// Context tags inside an ExtendedRequest / ExtendedResponse.
	tagExtendedOID   = ber.Tag(0)
	tagExtendedValue = ber.Tag(1)
)

// message wraps a protocol operation in an LDAPMessage with its message id.
func message(id int64, op *ber.Packet) *ber.Packet {
	packet := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "LDAPMessage")
	packet.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, id, "MessageID"))
	packet.AppendChild(op)
	return packet
}

// writeSASLBindRequest sends a SASL BindRequest with the given mechanism and
// credentials. A nil credentials sends the mechanism alone, which is how a
// challenge-response exchange starts.
func writeSASLBindRequest(w io.Writer, id int64, mechanism string, credentials []byte) error {
	req := ber.Encode(ber.ClassApplication, ber.TypeConstructed, appBindRequest, nil, "Bind Request")
	req.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, ldapVersion3, "Version"))
	// The name is empty for SASL: the mechanism carries the identity.
	req.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", "Name"))

	sasl := ber.Encode(ber.ClassContext, ber.TypeConstructed, tagSASLAuthentication, nil, "SASL Credentials")
	sasl.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, mechanism, "Mechanism"))
	if credentials != nil {
		sasl.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString,
			string(credentials), "Credentials"))
	}
	req.AppendChild(sasl)

	_, err := w.Write(message(id, req).Bytes())
	return err
}

// readBindResponse reads one BindResponse, returning the verbatim result and
// any server SASL credentials it carried.
func readBindResponse(r io.Reader) (Result, []byte, error) {
	packet, err := ber.ReadPacket(r)
	if err != nil {
		return Result{}, nil, err
	}
	op, err := operation(packet, appBindResponse)
	if err != nil {
		return Result{}, nil, err
	}

	result, err := ldapResult(op)
	if err != nil {
		return Result{}, nil, err
	}

	var serverCreds []byte
	for _, child := range op.Children[3:] {
		if child.ClassType == ber.ClassContext && child.Tag == tagServerSASLCreds {
			serverCreds = child.Data.Bytes()
		}
	}
	return result, serverCreds, nil
}

// writeExtendedRequest sends an ExtendedRequest for oid.
func writeExtendedRequest(w io.Writer, id int64, oid string, value []byte) error {
	req := ber.Encode(ber.ClassApplication, ber.TypeConstructed, appExtendedRequest, nil, "Extended Request")
	req.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, tagExtendedOID, oid, "OID"))
	if value != nil {
		req.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, tagExtendedValue, string(value), "Value"))
	}
	_, err := w.Write(message(id, req).Bytes())
	return err
}

// readExtendedResponse reads one ExtendedResponse, returning its verbatim
// result and response value.
func readExtendedResponse(r io.Reader) (Result, []byte, error) {
	packet, err := ber.ReadPacket(r)
	if err != nil {
		return Result{}, nil, err
	}
	op, err := operation(packet, appExtendedResponse)
	if err != nil {
		return Result{}, nil, err
	}
	result, err := ldapResult(op)
	if err != nil {
		return Result{}, nil, err
	}

	var value []byte
	for _, child := range op.Children[3:] {
		if child.ClassType == ber.ClassContext && child.Tag == tagExtendedValue {
			value = child.Data.Bytes()
		}
	}
	return result, value, nil
}

// operation unwraps an LDAPMessage and checks the protocol operation's tag.
func operation(packet *ber.Packet, want ber.Tag) (*ber.Packet, error) {
	if packet == nil || len(packet.Children) < 2 {
		return nil, errors.New("ldapx: malformed LDAP message")
	}
	op := packet.Children[1]
	if op.ClassType != ber.ClassApplication || op.Tag != want {
		return nil, fmt.Errorf("ldapx: expected protocol operation %d, got %d", want, op.Tag)
	}
	return op, nil
}

// ldapResult reads the resultCode, matchedDN and diagnosticMessage that begin
// every LDAP response. All three are taken verbatim (SC-006).
func ldapResult(op *ber.Packet) (Result, error) {
	if len(op.Children) < 3 {
		return Result{}, errors.New("ldapx: malformed LDAP result")
	}
	code, ok := op.Children[0].Value.(int64)
	if !ok {
		return Result{}, errors.New("ldapx: malformed result code")
	}
	return NewResult(
		int(code),
		op.Children[1].Data.String(),
		op.Children[2].Data.String(),
	), nil
}
