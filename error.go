package main

import "github.com/gin-gonic/gin"

type RequestError struct {
	Message  string
	Response gin.H
}

func (e *RequestError) Error() string {
	return e.Message
}

func NewRequestError(message string, errType string) *RequestError {
	return &RequestError{
		Message: message,
		Response: gin.H{
			"error": gin.H{
				"message": message,
				"type":    errType,
				"param":   nil,
				"code":    nil,
			},
		},
	}
}

var InvalidRequestError = NewRequestError("We could not parse the JSON body of your request.", "invalid_request_error")
var InternalServerError = NewRequestError("Internal server error", "internal_server_error")
var RequestTimeoutError = NewRequestError("Request timeout", "request_timeout_error")
