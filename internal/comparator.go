package internal

import "bytes"

type Comparator struct{}

// Comparator sorts keys in ascending order and sequences in descending order.
func (Comparator) Compare(a, b []byte) int {
	// Compare user keys.
	ua := ExtractUserKey(a)
	ub := ExtractUserKey(b)

	if c := bytes.Compare(ua, ub); c != 0 {
		return c
	}

	// Same user key: compare sequence numbers (descending).
	aseq, atype, _ := ExtractTrailer(a)
	bseq, btype, _ := ExtractTrailer(b)

	if aseq > bseq {
		return -1
	}
	if aseq < bseq {
		return 1
	}

	// Same sequence: compare types (ascending).
	if atype < btype {
		return -1
	}
	if atype > btype {
		return 1
	}

	return 0
}
