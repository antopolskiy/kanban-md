// Package date provides a Date type that marshals as YYYY-MM-DD.
package date

import (
	"encoding/json"
	"fmt"
	"time"

	"go.yaml.in/yaml/v3"
)

const format = "2006-01-02"

// Date represents a calendar date without time or timezone.
type Date struct {
	time.Time
}

// New creates a Date from year, month, day.
func New(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// Today returns today's date.
func Today() Date {
	now := time.Now()
	return New(now.Year(), now.Month(), now.Day())
}

// Parse parses a YYYY-MM-DD string into a Date.
func Parse(s string) (Date, error) {
	t, err := time.Parse(format, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", s)
	}
	return Date{t}, nil
}

// String returns the date as YYYY-MM-DD.
func (d Date) String() string {
	return d.Format(format)
}

// MarshalYAML implements yaml.Marshaler.
func (d Date) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalYAML implements yaml.v3 Unmarshaler.
func (d *Date) UnmarshalYAML(value *yaml.Node) error {
	parsed, err := Parse(value.Value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
