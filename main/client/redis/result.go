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

type ArrayCmd struct {
	err string
	res []string
	len int
}

type NumberCmd struct {
	err string
	res int
}

func (cmd NumberCmd) Result() any {
	return cmd.res
}

func (cmd NumberCmd) Error() error {
	return errors.New(cmd.err)
}

func (cmd ArrayCmd) Result() any {
	return cmd.res
}

func (cmd ArrayCmd) Error() error {
	switch cmd.err {
	case "":
		return nil
	case "-1":
		return NilError{
			msg: "value is nil",
		}
	default:
		return errors.New(cmd.err)
	}
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
