package outtune

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
)

// Loosely ported from Mozilla Firefox source code, licensed under Mozilla public license
//
// The format of the key is the base64 encoding of the following:
// 4 bytes: {0, 0, 0, 0} (this was intended to be the module ID, but it was
//                        never implemented)
// 4 bytes: {0, 0, 0, 0} (this was intended to be the slot ID, but it was
//                        never implemented)
// 4 bytes: <serial number length in big-endian order>
// 4 bytes: <DER-encoded issuer distinguished name length in big-endian order>
// n bytes: <bytes of serial number>
// m bytes: <DER-encoded issuer distinguished name>
type mozkey struct {
	moduleID     uint32 //unused
	slotID       uint32 //unused
	serialLength uint32
	dnLength     uint32
	serial       []byte
	dn           []byte
}

func GenerateDBKey(cert *x509.Certificate) (string, error) {
	buffer := &bytes.Buffer{}
	ew := &errorWriter{w: buffer}

	sn := cert.SerialNumber.Bytes()
	dn := cert.RawIssuer

	mk := &mozkey{
		serialLength: uint32(len(sn)),
		serial:       sn,
		dnLength:     uint32(len(dn)),
		dn:           dn,
	}

	binary.Write(ew, binary.BigEndian, mk.moduleID)
	binary.Write(ew, binary.BigEndian, mk.slotID)
	binary.Write(ew, binary.BigEndian, mk.serialLength)
	binary.Write(ew, binary.BigEndian, mk.dnLength)
	binary.Write(ew, binary.BigEndian, mk.serial)
	binary.Write(ew, binary.BigEndian, mk.dn)

	if ew.Error() != nil {
		return "", ew.Error()
	}

	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
