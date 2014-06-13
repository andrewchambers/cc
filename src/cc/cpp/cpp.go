package cpp

import "io"

type IncludeSearcher interface {
	//IncludeQuote is invoked when the preprocessor
	//encounters an include of the form #include "foo.h".
	IncludeQuote(path string) (io.Reader, error)
	//IncludeAngled is invoked when the preprocessor
	//encounters an include of the form #include <foo.h>.
	IncludeAngled(path string) (io.Reader, error)
}

type Preprocessor struct {
	is IncludeSearcher
}

func New(is IncludeSearcher) *Preprocessor {
	return nil
}

func (cpp *Preprocessor) PreprocessFile() {

}

//Define can be used to predefine values in the preprocessor.
//This is what is used to perform -D defines from the command line.
func (cpp *Preprocessor) Define() {

}
