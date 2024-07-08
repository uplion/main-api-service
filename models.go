package main

import "encoding/json"

type Task struct {
	RequestID string          `json:"request_id"`
	Stream    bool            `json:"stream"`
	Data      json.RawMessage `json:"data"`
	EndPoint  string          `json:"endpoint"`
}

type ClientRequest struct {
	Stream *bool  `json:"stream"`
	Model  string `json:"model"`
}

type TaskResult struct {
	RequestID string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
}

type TaskStreamResult struct {
	RequestID string           `json:"request_id"`
	End       bool             `json:"end"`
	Data      *json.RawMessage `json:"data"`
}
