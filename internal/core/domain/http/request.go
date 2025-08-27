package httpdomain

import "io"

type RequestContext struct {
	Method  string
	Path    string
	Query   map[string]string
	Headers map[string]string
	Body    io.Reader
}
