package antigravity

import (
	"encoding/binary"
	"errors"
)

var errMalformedProto = errors.New("malformed protobuf")

// protoField is one top-level field of a protobuf message.
type protoField struct {
	num  int
	wire int
	u    uint64 // varint, fixed32 and fixed64 values
	b    []byte // length-delimited payload
}

// parseProto splits a protobuf message into its top-level fields. Antigravity
// ships no schema, so only the fields codetok needs are read by number.
func parseProto(b []byte) ([]protoField, error) {
	var out []protoField
	for len(b) > 0 {
		key, n := binary.Uvarint(b)
		if n <= 0 {
			return nil, errMalformedProto
		}
		b = b[n:]
		f := protoField{num: int(key >> 3), wire: int(key & 7)}
		if f.num <= 0 {
			return nil, errMalformedProto
		}
		switch f.wire {
		case 0:
			v, n := binary.Uvarint(b)
			if n <= 0 {
				return nil, errMalformedProto
			}
			f.u, b = v, b[n:]
		case 1:
			if len(b) < 8 {
				return nil, errMalformedProto
			}
			f.u, b = binary.LittleEndian.Uint64(b), b[8:]
		case 2:
			l, n := binary.Uvarint(b)
			if n <= 0 || l > uint64(len(b)-n) {
				return nil, errMalformedProto
			}
			f.b, b = b[n:n+int(l)], b[n+int(l):]
		case 5:
			if len(b) < 4 {
				return nil, errMalformedProto
			}
			f.u, b = uint64(binary.LittleEndian.Uint32(b)), b[4:]
		default:
			return nil, errMalformedProto
		}
		out = append(out, f)
	}
	return out, nil
}

// message returns the first length-delimited field num.
func message(fields []protoField, num int) ([]byte, bool) {
	for _, f := range fields {
		if f.num == num && f.wire == 2 {
			return f.b, true
		}
	}
	return nil, false
}

// uintField returns the first varint field num, or 0.
func uintField(fields []protoField, num int) uint64 {
	for _, f := range fields {
		if f.num == num && f.wire == 0 {
			return f.u
		}
	}
	return 0
}

// repeatedUint returns all values of a repeated varint field, packed or not.
func repeatedUint(fields []protoField, num int) ([]uint64, error) {
	var out []uint64
	for _, f := range fields {
		if f.num != num {
			continue
		}
		switch f.wire {
		case 0:
			out = append(out, f.u)
		case 2:
			for b := f.b; len(b) > 0; {
				v, n := binary.Uvarint(b)
				if n <= 0 {
					return nil, errMalformedProto
				}
				out = append(out, v)
				b = b[n:]
			}
		}
	}
	return out, nil
}
