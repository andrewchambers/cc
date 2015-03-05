package cpp

// The hideset of a token is the set of identifiers whose expansion resulted in the token.
//
// Hidesets prevent infinite macro expansion.
// It is implemented as an immutable singly linked list for code clarity.
// Performance should be ok for most real world code (hidesets are small in practice).

type hideset struct {
	r   *hideset
	val string
}

var emptyHS *hideset = nil

func (hs *hideset) rest() *hideset {
	if hs == emptyHS {
		return emptyHS
	}
	return hs.r
}

func (hs *hideset) len() int {
	l := 0
	for hs != emptyHS {
		l += 1
		hs = hs.rest()
	}
	return l
}

func (hs *hideset) contains(s string) bool {
	for hs != emptyHS {
		if hs.val == s {
			return true
		}
		hs = hs.rest()
	}
	return false
}

func (hs *hideset) add(s string) *hideset {
	if hs.contains(s) {
		return hs
	}
	return &hideset{
		r:   hs,
		val: s,
	}
}

func (hs *hideset) union(b *hideset) *hideset {
	for hs != emptyHS {
		b = b.add(hs.val)
		hs = hs.rest()
	}
	return b
}

func (hs *hideset) intersection(b *hideset) *hideset {
	ret := emptyHS
	for hs != emptyHS {
		if b.contains(hs.val) {
			ret = ret.add(hs.val)
		}
		hs = hs.rest()
	}
	return ret
}
