package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type outPutter interface {
	init() error
	writeFile(string, []byte) error
}

func newFileWriter(dir string) outPutter {
	return &fileWriter{
		dirname: dir,
	}
}

type fileWriter struct {
	dirname string
	created bool
}

func (o *fileWriter) init() error {
	o.created = true
	return os.MkdirAll(o.dirname, 0755)
}

func (o fileWriter) writeFile(s string, bytes []byte) error {
	if !o.created {
		return fmt.Errorf("writeFile() called before mkdir()")
	}
	filepath := path.Join(o.dirname, s)

	if strings.Contains(s, string(os.PathSeparator)) {
		err := os.MkdirAll(path.Dir(filepath), 0755)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(filepath, bytes, 0644)
}
