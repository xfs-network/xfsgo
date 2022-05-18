package api

import "xfsgo"

func errorcasefn(errfn func() error) error {
	if err := errfn(); err != nil {
		return xfsgo.NewRPCErrorCause(-1006, err)
	}
	return nil
}

func errorcase(err error) error {
	if err != nil {
		return xfsgo.NewRPCErrorCause(-1006, err)
	}
	return nil
}
