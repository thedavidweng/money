package contracts

import (
	"encoding/json"
	"io"
)

const SchemaVersion = "2026-07-25"

type Envelope struct {
	OK    bool      `json:"ok"`
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
	Meta  Meta      `json:"meta"`
}

type Meta struct {
	Command       string      `json:"command"`
	Profile       string      `json:"profile"`
	DurationMs    int64       `json:"duration_ms"`
	SchemaVersion string      `json:"schema_version"`
	RequestID     string      `json:"request_id"`
	Demo          bool        `json:"demo,omitempty"`
	Pagination    *Pagination `json:"pagination,omitempty"`
	Warnings      []Warning   `json:"warnings,omitempty"`
}

type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   *int `json:"total,omitempty"`
	HasMore bool `json:"has_more"`
}

type APIError struct {
	Code      string     `json:"code"`
	Message   string     `json:"message"`
	Category  Category   `json:"category"`
	Retryable bool       `json:"retryable"`
	Details   []APIError `json:"details,omitempty"`
}

type Warning struct {
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Category Category `json:"category"`
}

type Category string

const (
	CategoryAuth       Category = "auth"
	CategoryNetwork    Category = "network"
	CategoryAPI        Category = "api"
	CategoryValidation Category = "validation"
	CategorySafety     Category = "safety"
	CategoryConfig     Category = "config"
	CategoryInternal   Category = "internal"
)

func NewSuccess(command string, data any) Envelope {
	return Envelope{
		OK:   true,
		Data: data,
		Meta: Meta{
			Command:       command,
			SchemaVersion: SchemaVersion,
		},
	}
}

func NewError(command, code, message string, category Category, retryable bool) Envelope {
	return Envelope{
		OK:    false,
		Error: &APIError{Code: code, Message: message, Category: category, Retryable: retryable},
		Meta: Meta{
			Command:       command,
			SchemaVersion: SchemaVersion,
		},
	}
}

func WriteJSON(w io.Writer, env *Envelope, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(env)
}
