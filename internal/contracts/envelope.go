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
	Warnings []string   `json:"warnings"`
	Errors   []APIError `json:"errors"`
}

type Meta struct {
	Command       string `json:"command"`
	SchemaVersion string `json:"schema_version"`
	GeneratedAt   string `json:"generated_at"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewSuccess(command string, data any) Envelope {
	return Envelope{
		OK:   true,
		Data: data,
		Meta: Meta{
			Command:       command,
			SchemaVersion: SchemaVersion,
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		Warnings: []string{},
		Errors:   []APIError{},
	}
}

func NewError(command, code, message string) Envelope {
	return Envelope{
		OK: false,
		Meta: Meta{
			Command:       command,
			SchemaVersion: SchemaVersion,
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		Warnings: []string{},
		Errors:   []APIError{{Code: code, Message: message}},
	}
}

func WriteJSON(w io.Writer, env Envelope) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
