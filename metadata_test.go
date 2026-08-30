package openapi_test

import (
	"testing"

	"github.com/whilesmartgo/openapi"
)

func TestDescribeReturnsHandler(t *testing.T) {
	handler := func() string { return "handled" }
	described := openapi.Describe(handler, openapi.Operation{Summary: "Handle request"})
	if described() != "handled" {
		t.Fatal("described handler changed behavior")
	}
}
