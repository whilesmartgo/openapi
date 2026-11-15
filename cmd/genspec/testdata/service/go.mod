module example.com/service

go 1.24.0

require (
	github.com/gin-gonic/gin v0.0.0
	github.com/whilesmartgo/apicontract v0.1.1-0.20261115031627-62c9b44ab3d9
	github.com/whilesmartgo/openapi v0.0.0
)

replace github.com/gin-gonic/gin => ./gin

replace github.com/whilesmartgo/openapi => ../../../..

replace github.com/whilesmartgo/apicontract => ./apicontract
