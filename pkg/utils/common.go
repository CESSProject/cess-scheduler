package utils

import (
	"bytes"
	"errors"
	"fmt"
	"runtime/debug"
)

func RecoverError(err interface{}) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%v\n", "--------------------panic--------------------")
	fmt.Fprintf(buf, "%v\n", err)
	fmt.Fprintf(buf, "%v\n", string(debug.Stack()))
	return errors.New(buf.String())
}
