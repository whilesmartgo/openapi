package gin

type Context struct{}

type HandlerFunc func(*Context)

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (*Engine) POST(string, ...HandlerFunc) {}

func (*Context) ShouldBindJSON(any) error {
	return nil
}

func (*Context) JSON(int, any) {}
