package cpp

// The hideset of a token is the set of identifiers whose expansion resulted in the token.
//
// Hidesets prevent infinite macro expansion.
// It is implemented as an immutable singly linked list for code clarity.
// Performance should be ok for most real world code (hidesets are small in practice).

type hideset struct {
	next *hideset
	val  string
}

var emptyHS *hideset

func (hs *hideset) contains(s string) bool {
	for hs != nil {
		if s == hs.val {
			return true
		}
		hs = hs.next
	}
	return false
}

func (hs *hideset) add(s string) *hideset {
	if hs.contains(s) {
		return hs
	}
	return &hideset{
		next: hs,
		val:  s,
	}
}

func (hs *hideset) intersection(b *hideset) *hideset {
	for hs != nil {
		b = b.add(hs.val)
		hs = hs.next
	}
	return b
}

func (hs *hideset) union(b *hideset) *hideset {
	ret := emptyHS
	for hs.next != nil {
		if b.contains(hs.val) {
			ret = ret.add(hs.val)
		}
		hs = hs.next
	}
	return ret
}
