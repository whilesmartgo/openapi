module example.com/service

go 1.24.0

require (
	github.com/gin-gonic/gin v0.0.0
	github.com/whilesmartgo/openapi v0.0.0
)

replace github.com/gin-gonic/gin => ./gin

replace github.com/whilesmartgo/openapi => ../../../..
