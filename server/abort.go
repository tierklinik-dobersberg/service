package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/logger"
)

func isErrType(err error, ref error) bool {
	refType := reflect.Indirect(reflect.ValueOf(ref)).Type()
	unwrapCause := func(s error) error {
		e := errors.Unwrap(s)
		if e == nil {
			if causer, ok := s.(interface{ Cause() error }); ok {
				e = causer.Cause()
			}
		}

		return e
	}

	e := err
	for {
		t := reflect.Indirect(reflect.ValueOf(e)).Type()
		if refType == t {
			return true
		}

		e = unwrapCause(e)
		if e == nil {
			return false
		}
	}
}

// AbortRequest aborts the current request with status and err.
// If status is 0 it tries to automatically determine the appropriate
// status code for err.
func AbortRequest(ctx *gin.Context, status int, err error) {
	if status == 0 {
		status = http.StatusInternalServerError

		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}

		if isErrType(err, &json.SyntaxError{}) || isErrType(err, &strconv.NumError{}) {
			status = http.StatusBadRequest
		}

		if isErrType(err, &ValidationError{}) {
			status = http.StatusUnprocessableEntity
		}

		if e, ok := err.(interface{ StatusCode() int }); ok {
			status = e.StatusCode()
		}

		if e, ok := err.(interface{ Code() int }); ok {
			code := e.Code()
			// below is a simple plausability check to ensure
			// Code() was meant to be used for HTTP
			if code >= 400 && code < 600 {
				status = code
			}
		}
	}

	ctx.AbortWithStatus(status)

	fields := logger.Fields{
		"error": err.Error(),
		"url":   ctx.Request.URL.Path,
	}

	for _, v := range ctx.Params {
		fields[v.Key] = v.Value
	}

	logger.From(ctx.Request.Context()).WithFields(fields).Errorf("failed to handle request")
}
