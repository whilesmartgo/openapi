package apicontract

import (
	"net/http"
	"time"
)

type Page struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"200" clamp:"true"`
	Cursor string `query:"cursor" maxLength:"2048"`
}

type Range struct {
	From time.Time `query:"from"`
	To   time.Time `query:"to" bound:"end"`
}

type SortKey struct {
	Field string
	Desc  bool
}

func BindQuery(*http.Request, any) error {
	return nil
}

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func BindJSON(*http.Request, any, ...func()) error {
	return nil
}

func IdempotencyKey(*http.Request) (string, error) {
	return "", nil
}

func BindJSONBytes([]byte, any, ...func()) error {
	return nil
}

func WriteJSON(http.ResponseWriter, int, any) error {
	return nil
}

func WriteList[T any](http.ResponseWriter, []T, string) error {
	return nil
}

func WriteError(http.ResponseWriter, int, string, string, ...any) error {
	return nil
}

type Problem struct{}

func (*Problem) Error() string {
	return ""
}

func NewProblem(int, string, string, ...any) *Problem {
	return &Problem{}
}

func WriteProblem(http.ResponseWriter, error) error {
	return nil
}
