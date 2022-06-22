package redis

import "errors"

type NilError struct {
	msg string
}

type Cmd interface {
	Error() error
	Result() any
}

type ResCmd struct {
	err string
	res string
	len int64
}

func (err NilError) Error() string {
	return err.msg
}

func (r ResCmd) Error() error {
	switch r.err {
	case "":
		return nil
	case "-1":
		return NilError{
			msg: "value is nil",
		}
	default:
		return errors.New(r.err)
	}
}

func (r ResCmd) Result() any {
	return r.res
}
