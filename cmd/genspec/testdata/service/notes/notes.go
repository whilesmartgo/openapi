package notes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type noteInput struct {
	Body string `json:"body" binding:"required"`
}

type Handler struct{}

func (*Handler) Create(context *gin.Context) {
	var input noteInput
	_ = context.ShouldBindJSON(&input)
	context.JSON(http.StatusCreated, input)
}

func (h *Handler) Routes(api *gin.Engine) {
	api.POST("/notes", h.Create)
}
