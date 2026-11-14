package main

import (
	"net/http"

	"example.com/service/notes"
	"github.com/gin-gonic/gin"
	"github.com/whilesmartgo/apicontract"
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

type eventsQuery struct {
	apicontract.Page
	Updated apicontract.Range     `query:"updated"`
	Sort    []apicontract.SortKey `query:"sort" enum:"starts_at,title"`
	Start   string                `query:"start" deprecated:"-"`
}

func (*eventsHandler) List(context *gin.Context) {
	var query eventsQuery
	_ = apicontract.BindQuery(context.Request, &query)
	_ = context.Query("calendar_id")
	_ = context.Query("limit")
	context.JSON(http.StatusOK, eventOutput{})
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
	router.GET("/events", events.List)
	(&notes.Handler{}).Routes(router)
}
