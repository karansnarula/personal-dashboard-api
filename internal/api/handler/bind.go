package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
)

// bindJSON decodes the request body into v and writes a 400 with a
// client-friendly message on failure. It returns false when the handler
// should stop.
func bindJSON(c *gin.Context, v any) bool {
	err := c.ShouldBindJSON(v)
	if err == nil {
		return true
	}

	var (
		syntaxErr *json.SyntaxError
		typeErr   *json.UnmarshalTypeError
		maxErr    *http.MaxBytesError
		fieldErrs validator.ValidationErrors
		msg       string
	)
	switch {
	case errors.Is(err, io.EOF):
		msg = "request body is empty"
	case errors.As(err, &syntaxErr):
		msg = "malformed JSON at byte " + strconv.FormatInt(syntaxErr.Offset, 10)
	case errors.As(err, &typeErr):
		msg = "field \"" + typeErr.Field + "\" has the wrong type"
	case errors.As(err, &maxErr):
		msg = "request body too large"
	case errors.As(err, &fieldErrs):
		msg = "missing or invalid fields: " + fieldNames(fieldErrs)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		msg = "unknown field " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	default:
		msg = "invalid JSON body"
	}
	response.Error(c, http.StatusBadRequest, response.CodeBadRequest, msg)
	return false
}

func fieldNames(errs validator.ValidationErrors) string {
	names := make([]string, 0, len(errs))
	for _, fe := range errs {
		names = append(names, strings.ToLower(fe.Field()))
	}
	return strings.Join(names, ", ")
}

// pathID parses a positive integer :id path parameter.
func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}
