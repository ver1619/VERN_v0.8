package internal

import "bytes"

// Comparator defines ordering over InternalKeys.
type Comparator struct{}

// Compare compares two encoded InternalKeys.
// Returns: -1 if a < b, 0 if a == b, +1 if a > b
func (Comparator) Compare(a, b []byte) int {
	// 1. Compare user keys
	ua := ExtractUserKey(a)
	ub := ExtractUserKey(b)

	if c := bytes.Compare(ua, ub); c != 0 {
		return c
	}

	// 2. Same user key → compare sequence DESCENDING
	aseq, atype, _ := ExtractTrailer(a)
	bseq, btype, _ := ExtractTrailer(b)

	if aseq > bseq {
		return -1
	}
	if aseq < bseq {
		return 1
	}

	// 3. Same seq → compare record type ASCENDING
	if atype < btype {
		return -1
	}
	if atype > btype {
		return 1
	}

	return 0
}
