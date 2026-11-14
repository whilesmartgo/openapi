package gin

import "net/http"

type Context struct {
	Request *http.Request
}

type HandlerFunc func(*Context)

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (*Engine) POST(string, ...HandlerFunc) {}

func (*Engine) GET(string, ...HandlerFunc) {}

func (*Context) Query(string) string {
	return ""
}

func (*Context) ShouldBindJSON(any) error {
	return nil
}

func (*Context) JSON(int, any) {}
