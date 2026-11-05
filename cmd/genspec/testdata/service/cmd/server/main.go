package main

import (
	"net/http"

	"example.com/service/notes"
	"github.com/gin-gonic/gin"
	"github.com/whilesmartgo/openapi"
)

type eventInput struct {
	Title string `json:"title" binding:"required" doc:"Event title" example:"Planning session"`
}

type eventOutput struct {
	ID string `json:"id"`
}

type eventsHandler struct{}

func (*eventsHandler) Create(context *gin.Context) {
	var input eventInput
	_ = context.ShouldBindJSON(&input)
	context.JSON(http.StatusCreated, eventOutput{})
}

func main() {
	router := gin.New()
	events := &eventsHandler{}
	router.POST("/events", openapi.Describe(events.Create, openapi.Operation{
		Summary:     "Create an event",
		Description: "Creates an event for the selected owner.",
		Tags:        []string{"calendar"},
		Deprecated:  true,
	}))
	(&notes.Handler{}).Routes(router)
}
