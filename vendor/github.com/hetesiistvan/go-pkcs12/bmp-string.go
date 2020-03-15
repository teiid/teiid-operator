// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs12

import (
	"encoding/binary"
	"errors"
	"math/bits"
	"unicode/utf16"
)

// bmpString returns s encoded in UCS-2 with a zero terminator.
func bmpString(s string) ([]byte, error) {
	// References:
	// https://tools.ietf.org/html/rfc7292#appendix-B.1
	// https://en.wikipedia.org/wiki/Plane_(Unicode)#Basic_Multilingual_Plane
	//  - non-BMP characters are encoded in UTF 16 by using a surrogate pair of 16-bit codes
	//	  EncodeRune returns 0xfffd if the rune does not need special encoding
	//  - the above RFC provides the info that BMPStrings are NULL terminated.

	ret := make([]byte, 0, 2*len(s)+2)

	for _, r := range s {
		if t, _ := utf16.EncodeRune(r); t != 0xfffd {
			return nil, errors.New("pkcs12: string contains characters that cannot be encoded in UCS-2")
		}
		ret = append(ret, byte(r/256), byte(r%256))
	}

	return append(ret, 0, 0), nil
}

func decodeBMPString(bmpString []byte) (string, error) {
	if len(bmpString)%2 != 0 {
		return "", errors.New("pkcs12: odd-length BMP string")
	}

	// strip terminator if present
	if l := len(bmpString); l >= 2 && bmpString[l-1] == 0 && bmpString[l-2] == 0 {
		bmpString = bmpString[:l-2]
	}

	s := make([]uint16, 0, len(bmpString)/2)
	for len(bmpString) > 0 {
		s = append(s, uint16(bmpString[0])<<8+uint16(bmpString[1]))
		bmpString = bmpString[2:]
	}

	return string(utf16.Decode(s)), nil
}

// marshalBmpString marshals a string into a ASN1 type 30 (BMP) string. DER encoding is used by marshalling.
// See https://en.wikipedia.org/wiki/X.690#DER_encoding
func marshalBmpString(s string) ([]byte, error) {
	lenOctets, lenOctetsSize := computeBmpStringSizeBytes(s)
	// Slice is computed from the following elements:
	// - one byte for the type
	// - len octet(s)
	// - string
	ret := make([]byte, 0, 2*len(s)+1+int(lenOctetsSize))

	ret = append(ret, 30)
	ret = append(ret, lenOctets...)

	for _, r := range s {
		if t, _ := utf16.EncodeRune(r); t != 0xfffd {
			return nil, errors.New("pkcs12: string contains characters that cannot be encoded in UCS-2")
		}
		ret = append(ret, byte(r/256), byte(r%256))
	}

	return ret, nil
}

// unmarshalBmpString unmarshals a ASN1 type 30 (BMP) slice (DER encoded) into a string.
// See https://en.wikipedia.org/wiki/X.690#DER_encoding
func unmarshalBmpString(derString []byte) (string, error) {
	// Do basic verification and compute the string size
	stringSize, err := computeBmpStringSize(derString)
	if err != nil {
		return "", err
	}
	startIndex := len(derString) - stringSize
	stringSlice := derString[startIndex:]

	uintSlice := make([]uint16, 0, len(stringSlice)/2)
	for len(stringSlice) > 0 {
		uintSlice = append(uintSlice, uint16(stringSlice[0])<<8+uint16(stringSlice[1]))
		stringSlice = stringSlice[2:]
	}

	resultString := string(utf16.Decode(uintSlice))
	return resultString, nil
}

// computeBmpStringSizeBytes calculates the lentgh field size of the BMP string according the DER encoding rules.
// See https://en.wikipedia.org/wiki/X.690#Length_octets
func computeBmpStringSizeBytes(s string) (lengthBytes []byte, lengthBytesSize byte) {
	var stringSize uint = uint(len(s)) * 2

	// Short form
	if stringSize <= 126 {
		lengthBytes = []byte{byte(stringSize)}
		lengthBytesSize = 1
		return
	}

	// Long form
	// Calculating first how many bytes are needed to represent the length
	stringSizeBits := bits.Len(stringSize)
	stringSizeBytes := byte(stringSizeBits / 8)
	if stringSizeBits%8 != 0 {
		stringSizeBytes++
	}

	// Allocating the slice needed to represent the length
	// We need one more byte as a header
	lengthBytesSize = stringSizeBytes + 1
	lengthBytes = make([]byte, 1, lengthBytesSize)
	// The header byte is 128 (7. bit set) + the number of bytes representing
	// the length of the string (lower bits of the byte)
	lengthBytes[0] = 128 + stringSizeBytes

	// Storing the bytes representing the length
	lengthBytesNoHeader := make([]byte, 8)
	binary.BigEndian.PutUint64(lengthBytesNoHeader, uint64(stringSize))
	lengthBytes = append(lengthBytes, lengthBytesNoHeader[8-stringSizeBytes:]...)

	return
}

// computeBmpStringSize decodes the len octets from the derBmpString argument.
// derBmpString contains the full string, including the element type and len
// octets.
// See https://en.wikipedia.org/wiki/X.690#Length_octets
func computeBmpStringSize(derBmpString []byte) (stringSize int, err error) {
	bpmSliceLen := len(derBmpString)

	switch bpmSliceLen {
	case 0:
		return -1, errors.New("pkcs12: empty der string provided")
	case 1:
		return -1, errors.New("pkcs12: invalid length, slice must be at least 2 byte")
	}

	// Checking the type
	if derBmpString[0] != 30 {
		return -1, errors.New("pkcs12: invalid DER bmp string type, it must be 30")
	}

	// Checking the first size octet
	octetFirst := derBmpString[1]
	if octetFirst < 128 {
		// Small form, size is defied in one byte

		// Size must be even
		if octetFirst%2 != 0 {
			return -1, errors.New("pkcs12: invalid size specified, it must be an even number")
		}

		// Cross checking the length octet to the size of the slice
		if int(octetFirst+2) != bpmSliceLen {
			return -1, errors.New("pkcs12: invalid size specified, it is not matching the length of the provided slice")
		}

		return int(octetFirst), nil
	}

	// Long form (size is defined in multiple bytes)
	lenOctetsSize := octetFirst - 128
	lenOctets := make([]byte, 8-lenOctetsSize, 8)
	lenOctets = append(lenOctets, derBmpString[2:2+lenOctetsSize]...)

	lenUint := binary.BigEndian.Uint64(lenOctets)

	// Size must be even
	if lenUint%2 != 0 {
		return -1, errors.New("pkcs12: invalid size specified, it must be an even number")
	}

	// Cross checking the length of the string slice with the calculated string length
	if int(uint64(lenOctetsSize)+2+lenUint) != len(derBmpString) {
		return -1, errors.New("pkcs12: invalid size specified, it is not matching the length of the provided slice")
	}

	return int(lenUint), nil
}
