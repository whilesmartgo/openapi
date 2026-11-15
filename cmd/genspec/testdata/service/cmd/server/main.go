package main

import (
	"net/http"
	"time"

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

type place struct {
	Name string `json:"name" required:"true"`
}

type eventPatch struct {
	Title     *string          `json:"title" maxLength:"200"`
	Day       apicontract.Date `json:"day"`
	Attendees []place          `json:"attendees" maxItems:"10"`
	Metadata  map[string]any   `json:"metadata"`
}

func (*eventsHandler) Update(context *gin.Context) {
	var patch eventPatch
	_, _ = apicontract.IdempotencyKey(context.Request)
	_ = apicontract.BindJSON(context.Request, &patch)
	context.JSON(http.StatusOK, eventOutput{})
}

type stamp = time.Time

type countPointer *int

type taskInput struct {
	Due   stamp        `json:"due_at"`
	Count countPointer `json:"count" maximum:"5"`
	Level int8         `json:"level"`
}

type internalInput struct {
	Secret string `json:"secret"`
}

type taskOutput struct {
	ID string `json:"id"`
}

type tasksHandler struct{}

func (*tasksHandler) Create(context *gin.Context) {
	unused := func() {
		_ = apicontract.BindJSON(context.Request, &internalInput{})
	}
	_ = unused
	var input taskInput
	if err := apicontract.BindJSONBytes(nil, &input); err != nil {
		return
	}
	_ = apicontract.WriteError(context.Writer, http.StatusConflict, "task_exists", "The task exists.")
	_ = apicontract.WriteJSON(context.Writer, http.StatusCreated, taskOutput{})
}

func (*tasksHandler) List(context *gin.Context) {
	_ = apicontract.WriteList[taskOutput](context.Writer, []taskOutput{}, "")
}

type archiveInput struct {
	Reason string `json:"reason"`
}

type restoreInput struct {
	At int `json:"at"`
}

type archived struct {
	Archived bool `json:"archived"`
}

func (*tasksHandler) Archive(context *gin.Context) {
	if context.Request.URL.Query().Get("restore") != "" {
		_ = apicontract.BindJSON(context.Request, &restoreInput{})
		_ = apicontract.WriteJSON(context.Writer, http.StatusOK, taskOutput{})
		return
	}
	_ = apicontract.BindJSON(context.Request, &archiveInput{})
	_ = apicontract.WriteJSON(context.Writer, http.StatusOK, archived{})
	_ = apicontract.WriteJSON(context.Writer, http.StatusAccepted, archived{})
}

func (*tasksHandler) Delete(context *gin.Context) {
	_ = apicontract.WriteJSON(context.Writer, http.StatusNoContent, nil)
}

func (*tasksHandler) Reset(context *gin.Context) {
	_ = apicontract.WriteJSON(context.Writer, http.StatusResetContent, struct{}{})
}

type renameInput struct {
	Name string `json:"name"`
}

func (*tasksHandler) Rename(context *gin.Context) {
	(func() {
		_ = apicontract.BindJSON(context.Request, &renameInput{})
	})()
	_ = apicontract.WriteProblem(context.Writer, apicontract.NewProblem(http.StatusConflict, "task_exists", "The task exists."))
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
	router.PATCH("/events/:id", events.Update)
	tasks := &tasksHandler{}
	router.POST("/tasks", tasks.Create)
	router.GET("/tasks", tasks.List)
	router.DELETE("/tasks/:id", tasks.Delete)
	router.POST("/tasks/:id/archive", tasks.Archive)
	router.POST("/tasks/reset", tasks.Reset)
	router.PUT("/tasks/:id", tasks.Rename)
	(&notes.Handler{}).Routes(router)
}
