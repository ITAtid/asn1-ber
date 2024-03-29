package ber

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

type Packet struct {
	ClassType   uint8
	TagType     uint8
	Tag         uint8
	Value       interface{}
	Data        *bytes.Buffer
	Children    []*Packet
	Description string
}

const (
	TagEOC              = 0x00
	TagBoolean          = 0x01
	TagInteger          = 0x02
	TagBitString        = 0x03
	TagOctetString      = 0x04
	TagNULL             = 0x05
	TagObjectIdentifier = 0x06
	TagObjectDescriptor = 0x07
	TagExternal         = 0x08
	TagRealFloat        = 0x09
	TagEnumerated       = 0x0a
	TagEmbeddedPDV      = 0x0b
	TagUTF8String       = 0x0c
	TagRelativeOID      = 0x0d
	TagSequence         = 0x10
	TagSet              = 0x11
	TagNumericString    = 0x12
	TagPrintableString  = 0x13
	TagT61String        = 0x14
	TagVideotexString   = 0x15
	TagIA5String        = 0x16
	TagUTCTime          = 0x17
	TagGeneralizedTime  = 0x18
	TagGraphicString    = 0x19
	TagVisibleString    = 0x1a
	TagGeneralString    = 0x1b
	TagUniversalString  = 0x1c
	TagCharacterString  = 0x1d
	TagBMPString        = 0x1e
	TagBitmask          = 0x1f // xxx11111b
)

var TagMap = map[uint8]string{
	TagEOC:              "EOC (End-of-Content)",
	TagBoolean:          "Boolean",
	TagInteger:          "Integer",
	TagBitString:        "Bit String",
	TagOctetString:      "Octet String",
	TagNULL:             "NULL",
	TagObjectIdentifier: "Object Identifier",
	TagObjectDescriptor: "Object Descriptor",
	TagExternal:         "External",
	TagRealFloat:        "Real (float)",
	TagEnumerated:       "Enumerated",
	TagEmbeddedPDV:      "Embedded PDV",
	TagUTF8String:       "UTF8 String",
	TagRelativeOID:      "Relative-OID",
	TagSequence:         "Sequence and Sequence of",
	TagSet:              "Set and Set OF",
	TagNumericString:    "Numeric String",
	TagPrintableString:  "Printable String",
	TagT61String:        "T61 String",
	TagVideotexString:   "Videotex String",
	TagIA5String:        "IA5 String",
	TagUTCTime:          "UTC Time",
	TagGeneralizedTime:  "Generalized Time",
	TagGraphicString:    "Graphic String",
	TagVisibleString:    "Visible String",
	TagGeneralString:    "General String",
	TagUniversalString:  "Universal String",
	TagCharacterString:  "Character String",
	TagBMPString:        "BMP String",
}

const (
	ClassUniversal   = 0   // 00xxxxxxb
	ClassApplication = 64  // 01xxxxxxb
	ClassContext     = 128 // 10xxxxxxb
	ClassPrivate     = 192 // 11xxxxxxb
	ClassBitmask     = 192 // 11xxxxxxb
)

var ClassMap = map[uint8]string{
	ClassUniversal:   "Universal",
	ClassApplication: "Application",
	ClassContext:     "Context",
	ClassPrivate:     "Private",
}

const (
	TypePrimitive   = 0  // xx0xxxxxb
	TypeConstructed = 32 // xx1xxxxxb
	TypeBitmask     = 32 // xx1xxxxxb
)

var TypeMap = map[uint8]string{
	TypePrimitive:   "Primitive",
	TypeConstructed: "Constructed",
}

var Debug = false

func PrintBytes(buf []byte, indent string) {
	dataLines := make([]string, (len(buf)/30)+1)
	numLines := make([]string, (len(buf)/30)+1)

	for i, b := range buf {
		dataLines[i/30] += fmt.Sprintf("%02x ", b)
		numLines[i/30] += fmt.Sprintf("%02d ", (i+1)%100)
	}

	for i := 0; i < len(dataLines); i++ {
		fmt.Print(indent + dataLines[i] + "\n")
		fmt.Print(indent + numLines[i] + "\n\n")
	}
}

func PrintPacket(p *Packet) {
	printPacket(p, 0, false)
}

func printPacket(p *Packet, indent int, printBytes bool) {
	indentStr := ""
	for len(indentStr) != indent {
		indentStr += " "
	}

	classStr := ClassMap[p.ClassType]
	tagtypeStr := TypeMap[p.TagType]
	tagStr := fmt.Sprintf("0x%02X", p.Tag)

	if p.ClassType == ClassUniversal {
		tagStr = TagMap[p.Tag]
	}

	value := fmt.Sprint(p.Value)
	description := ""
	if p.Description != "" {
		description = p.Description + ": "
	}

	fmt.Printf("%s%s(%s, %s, %s) Len=%d %q\n", indentStr, description, classStr, tagtypeStr, tagStr, p.Data.Len(), value)

	if printBytes {
		PrintBytes(p.Bytes(), indentStr)
	}

	for _, child := range p.Children {
		printPacket(child, indent+1, printBytes)
	}
}

func resizeBuffer(in []byte, newSize uint64) (out []byte) {
	out = make([]byte, newSize)
	copy(out, in)
	return
}

func ReadPacket(reader io.Reader) (*Packet, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}
	idx := uint64(2)
	datalen := uint64(buf[1])
	if Debug {
		fmt.Printf("Read: datalen = %d len(buf) = %d ", datalen, len(buf))
		for _, b := range buf {
			fmt.Printf("%02X ", b)
		}
		fmt.Printf("\n")
	}
	if datalen&128 != 0 {
		a := datalen - 128
		idx += a
		buf = resizeBuffer(buf, 2+a)
		if _, err := io.ReadFull(reader, buf[2:]); err != nil {
			return nil, err
		}
		datalen = DecodeInteger(buf[2 : 2+a])
		if Debug {
			fmt.Printf("Read: a = %d  idx = %d  datalen = %d  len(buf) = %d", a, idx, datalen, len(buf))
			for _, b := range buf {
				fmt.Printf("%02X ", b)
			}
			fmt.Printf("\n")
		}
	}

	buf = resizeBuffer(buf, idx+datalen)
	if _, err := io.ReadFull(reader, buf[idx:]); err != nil {
		return nil, err
	}

	if Debug {
		fmt.Printf("Read: len( buf ) = %d  idx=%d datalen=%d idx+datalen=%d\n", len(buf), idx, datalen, idx+datalen)
		for _, b := range buf {
			fmt.Printf("%02X ", b)
		}
	}

	return DecodePacket(buf), nil
}

// DecodeString returns a string version of data treating it as
// ASCII rather than UTF-8.
func DecodeString(data []byte) string {
	runes := make([]rune, len(data))
	for i, c := range data {
		runes[i] = rune(c)
	}
	return string(runes)
}

func DecodeInteger(data []byte) uint64 {
	var ret uint64
	for _, i := range data {
		ret <<= 8
		ret += uint64(i)
	}
	return ret
}

func EncodeInteger(val uint64) []byte {
	var out bytes.Buffer
	found := false
	shift := uint(56)
	mask := uint64(0xFF00000000000000)
	for mask > 0 {
		if !found && (val&mask != 0) {
			found = true
		}
		if found || (shift == 0) {
			out.Write([]byte{byte((val & mask) >> shift)})
		}
		shift -= 8
		mask = mask >> 8
	}
	return out.Bytes()
}

func DecodePacket(data []byte) *Packet {
	p, _ := decodePacket(data)
	return p
}

func decodePacket(data []byte) (*Packet, []byte) {
	if Debug {
		fmt.Printf("decodePacket: enter %d\n", len(data))
	}
	p := &Packet{
		ClassType: data[0] & ClassBitmask,
		TagType:   data[0] & TypeBitmask,
		Tag:       data[0] & TagBitmask,
		Data:      new(bytes.Buffer),
	}

	datalen := DecodeInteger(data[1:2])
	datapos := uint64(2)
	if datalen&128 != 0 {
		datalen -= 128
		datapos += datalen
		datalen = DecodeInteger(data[2 : 2+datalen])
	}

	valueData := data[datapos : datapos+datalen]

	if p.TagType == TypeConstructed {
		for len(valueData) != 0 {
			var child *Packet
			child, valueData = decodePacket(valueData)
			p.AppendChild(child)
		}
	} else if p.ClassType == ClassUniversal {
		p.Data.Write(data[datapos : datapos+datalen])
		switch p.Tag {
		case TagEOC:
		case TagBoolean:
			val := DecodeInteger(valueData)
			p.Value = val != 0
		case TagInteger:
			p.Value = DecodeInteger(valueData)
		case TagBitString:
		case TagOctetString:
			// should not be interpreted as Unicode code point
			// p.Value = DecodeString(valueData)
			p.Value = string(valueData)
		case TagNULL:
		case TagObjectIdentifier:
		case TagObjectDescriptor:
		case TagExternal:
		case TagRealFloat:
		case TagEnumerated:
			p.Value = DecodeInteger(valueData)
		case TagEmbeddedPDV:
		case TagUTF8String:
		case TagRelativeOID:
		case TagSequence:
		case TagSet:
		case TagNumericString:
		case TagPrintableString:
			p.Value = DecodeString(valueData)
		case TagT61String:
		case TagVideotexString:
		case TagIA5String:
		case TagUTCTime:
		case TagGeneralizedTime:
		case TagGraphicString:
		case TagVisibleString:
		case TagGeneralString:
		case TagUniversalString:
		case TagCharacterString:
		case TagBMPString:
		}
	} else {
		p.Data.Write(data[datapos : datapos+datalen])
	}

	return p, data[datapos+datalen:]
}

func (p *Packet) DataLength() uint64 {
	return uint64(p.Data.Len())
}

func (p *Packet) Bytes() []byte {
	n := p.DataLength()
	packetLength := EncodeInteger(n)
	size := 1 + len(packetLength) + int(n)
	isBig := n > 127 || len(packetLength) > 1
	if isBig {
		size++
	}

	out := make([]byte, size)
	out[0] = p.ClassType | p.TagType | p.Tag
	offset := 2
	if isBig {
		out[1] = byte(len(packetLength) | 128)
		offset += copy(out[2:], packetLength)
	} else {
		out[1] = packetLength[0]
	}
	copy(out[offset:], p.Data.Bytes())
	return out
}

func (p *Packet) AppendChild(child *Packet) {
	p.Data.Write(child.Bytes())
	p.Children = append(p.Children, child)
}

func Encode(classType, tagType, tag uint8, value interface{}, description string) *Packet {
	p := &Packet{
		ClassType:   classType,
		TagType:     tagType,
		Tag:         tag,
		Data:        new(bytes.Buffer),
		Value:       value,
		Description: description,
	}

	if value != nil {
		v := reflect.ValueOf(value)

		if classType == ClassUniversal {
			switch tag {
			case TagOctetString:
				sv, ok := v.Interface().(string)
				if ok {
					p.Data.Write([]byte(sv))
				}
			}
		}
	}

	return p
}

func NewSequence(description string) *Packet {
	return Encode(ClassUniversal, TypePrimitive, TagSequence, nil, description)
}

func NewBoolean(classType, tagType, tag uint8, value bool, description string) *Packet {
	intValue := 0
	if value {
		intValue = 1
	}

	p := Encode(classType, tagType, tag, nil, description)
	p.Value = value
	p.Data.Write(EncodeInteger(uint64(intValue)))
	return p
}

func NewInteger(classType, tagType, tag uint8, value uint64, description string) *Packet {
	p := Encode(classType, tagType, tag, nil, description)
	p.Value = value
	p.Data.Write(EncodeInteger(value))
	return p
}

func NewString(classType, tagType, tag uint8, value, description string) *Packet {
	p := Encode(classType, tagType, tag, nil, description)
	p.Value = value
	p.Data.Write([]byte(value))
	return p
}
