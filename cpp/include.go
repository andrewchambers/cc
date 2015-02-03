package cpp

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

type IncludeSearcher interface {
	//IncludeQuote is invoked when the preprocessor
	//encounters an include of the form #include "foo.h".
	//returns the full path of the file, a reader of the contents or an error.
	IncludeQuote(requestingFile, headerPath string) (string, io.Reader, error)
	//IncludeAngled is invoked when the preprocessor
	//encounters an include of the form #include <foo.h>.
	//returns the full path of the file, a reader of the contents or an error.
	IncludeAngled(requestingFile, headerPath string) (string, io.Reader, error)
}

type StandardIncludeSearcher struct {
	//Priority order list of paths to search for headers
	systemHeadersPath []string
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (is *StandardIncludeSearcher) IncludeQuote(requestingFile, headerPath string) (string, io.Reader, error) {
	dir := path.Dir(requestingFile)
	path := path.Join(dir, headerPath)
	exists, err := fileExists(path)
	if err != nil {
		return "", nil, err
	}
	if !exists {
		return is.IncludeAngled(requestingFile, headerPath)
	}
	rdr, err := os.Open(path)
	return path, rdr, err
}

func (is *StandardIncludeSearcher) IncludeAngled(requestingFile, headerPath string) (string, io.Reader, error) {
	for idx := range is.systemHeadersPath {
		dir := path.Dir(is.systemHeadersPath[idx])
		path := path.Join(dir, headerPath)
		exists, err := fileExists(path)
		if err != nil {
			return "", nil, err
		}
		if exists {
			rdr, err := os.Open(path)
			return path, rdr, err
		}
	}
	return "", nil, fmt.Errorf("header %s not found", headerPath)
}

//A ; seperated list of paths
func NewStandardIncludeSearcher(includePaths string) IncludeSearcher {
	ret := &StandardIncludeSearcher{}
	ret.systemHeadersPath = strings.Split(includePaths, ";")
	return ret
}
