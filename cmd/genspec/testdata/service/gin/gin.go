package gin

import "net/http"

type Context struct {
	Request *http.Request
	Writer  http.ResponseWriter
}

type HandlerFunc func(*Context)

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (*Engine) POST(string, ...HandlerFunc) {}

func (*Engine) GET(string, ...HandlerFunc) {}

func (*Engine) PATCH(string, ...HandlerFunc) {}

func (*Context) Query(string) string {
	return ""
}

func (*Context) ShouldBindJSON(any) error {
	return nil
}

func (*Context) JSON(int, any) {}

func (*Engine) PUT(string, ...HandlerFunc) {}

func (*Engine) DELETE(string, ...HandlerFunc) {}
