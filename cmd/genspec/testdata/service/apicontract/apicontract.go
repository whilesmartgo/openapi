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
