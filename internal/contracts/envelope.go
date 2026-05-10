package contracts

import (
	"encoding/json"
	"io"
	"time"
)

const SchemaVersion = "0.1"

type Envelope struct {
	OK       bool       `json:"ok"`
	Data     any        `json:"data,omitempty"`
	Meta     Meta       `json:"meta"`
	Warnings []Warning  `json:"warnings"`
	Errors   []APIError `json:"errors"`
}

type Meta struct {
	Command       string      `json:"command"`
	SchemaVersion string      `json:"schema_version"`
	GeneratedAt   string      `json:"generated_at"`
	Demo          bool        `json:"demo,omitempty"`
	Pagination    *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   *int `json:"total,omitempty"`
	HasMore bool `json:"has_more"`
}

type APIError struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Category  Category `json:"category"`
	Retryable bool     `json:"retryable"`
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
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		Warnings: []Warning{},
		Errors:   []APIError{},
	}
}

func NewError(command, code, message string, category Category, retryable bool) Envelope {
	return Envelope{
		OK: false,
		Meta: Meta{
			Command:       command,
			SchemaVersion: SchemaVersion,
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		Warnings: []Warning{},
		Errors:   []APIError{{Code: code, Message: message, Category: category, Retryable: retryable}},
	}
}

func WriteJSON(w io.Writer, env Envelope) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
