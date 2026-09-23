package proxyprobe

import "errors"

// asErr is errors.As with a less awkward call site.
func asErr(err error, target any) bool { return errors.As(err, target) }
