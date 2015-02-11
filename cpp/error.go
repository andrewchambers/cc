package cpp

import "fmt"

type ErrorLoc struct {
	Err error
	Pos FilePos
}

func ErrWithLoc(e error, pos FilePos) error {
	return ErrorLoc{
		Err: e,
		Pos: pos,
	}
}

func (e ErrorLoc) Error() string {
	return fmt.Sprintf("%s at %s", e.Err, e.Pos)
}
