package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/validator"
)

type ResponseWrapper map[string]interface{}

func (app *application) writeJSON(w http.ResponseWriter, status int, data ResponseWrapper, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		// badly formatted JSON - syntax error
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly formed JSON (at character %d)", syntaxError.Offset)

		// badly formatted JSON - unexpected EOF
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly formed JSON")

		// badly formatted JSON - invalid character 'x' looking for beginning of value
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains invalid JSON value for the %q field (at character %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			}
			return fmt.Errorf("body contains invalid JSON value (at character %d)", unmarshalTypeError.Offset)

		// empty request body
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// json: unknown field "name"
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown field %s", fieldName)

		// request body too large
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body is too large")

		// Faulty decode destination
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		}
	}

	// check for multiple json values in the request body
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	return s
}
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

func (app *application) readDate(qs url.Values, key string, defaultValue string) time.Time {
	s := qs.Get(key)
	if s == "" {
		s = defaultValue
	}

	s = s + "T00:00:00Z"
	t, err := time.Parse("02-01-2006T15:04:05Z07:00", s)
	if err != nil {
		return time.Time{}
	}

	return t
}
