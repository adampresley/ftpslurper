package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type HttpHelpersConfig struct {
}

type HttpHelpers struct {
}

func NewHttpHelpers(config HttpHelpersConfig) HttpHelpers {
	return HttpHelpers{}
}

func (s HttpHelpers) GetIntFromRequest(r *http.Request, name string) int {
	var (
		err         error
		valueString string
		value       int
	)

	valueString = r.FormValue(name)

	if value, err = strconv.Atoi(valueString); err != nil {
		vars := mux.Vars(r)
		valueString = vars[name]

		if value, err = strconv.Atoi(valueString); err != nil {
			value = 0
		}
	}

	return value
}

func (s HttpHelpers) GetIntSliceFromRequest(r *http.Request, name string) []int {
	var (
		err         error
		valueString []string
		value       []int = []int{}
		temp        int
	)

	valueString = r.Form[name]

	if len(valueString) > 0 {
		for _, v := range valueString {
			if temp, err = strconv.Atoi(v); err == nil {
				value = append(value, temp)
			}
		}
	}

	return value
}

func (h HttpHelpers) GetUIntFromRequest(r *http.Request, name string) uint {
	var (
		err         error
		valueString string
		value       uint
		temp        int
	)

	valueString = r.FormValue(name)

	if temp, err = strconv.Atoi(valueString); err != nil {
		vars := mux.Vars(r)
		valueString = vars[name]

		if temp, err = strconv.Atoi(valueString); err != nil {
			value = 0
		} else {
			value = uint(temp)
		}
	} else {
		value = uint(temp)
	}

	return value
}

func (h HttpHelpers) GetUIntSliceFromRequest(r *http.Request, name string) []uint {
	var (
		err         error
		valueString []string
		value       []uint = []uint{}
		temp        int
	)

	valueString = r.Form[name]

	if len(valueString) > 0 {
		for _, v := range valueString {
			if temp, err = strconv.Atoi(v); err == nil {
				value = append(value, uint(temp))
			}
		}
	}

	return value
}

func (h HttpHelpers) GetFloatFromRequest(r *http.Request, name string) float64 {
	var (
		err         error
		valueString string
		value       float64
	)

	valueString = r.FormValue(name)

	if value, err = strconv.ParseFloat(valueString, 64); err != nil {
		vars := mux.Vars(r)
		valueString = vars[name]

		if value, err = strconv.ParseFloat(valueString, 64); err != nil {
			value = 0
		}
	}

	return value
}

func (h HttpHelpers) GetStringFromRequest(r *http.Request, name string) string {
	value := r.FormValue(name)

	if value == "" {
		vars := mux.Vars(r)
		value = vars[name]
	}

	return value
}

func (h HttpHelpers) GetStringSliceFromRequest(r *http.Request, name string) []string {
	var (
		value []string
	)

	value = r.Form[name]
	return value
}

func (h HttpHelpers) GetStringListFromRequest(r *http.Request, name string) []string {
	var (
		value []string
	)

	values := r.Form.Get(name)
	value = strings.Split(values, ",")
	return value
}

/*
ReadJSONBody reads the body content from an http.Request as JSON data into
dest.
*/
func (h HttpHelpers) ReadJSONBody(r *http.Request, dest interface{}) error {
	var (
		err error
		b   []byte
	)

	if b, err = io.ReadAll(r.Body); err != nil {
		return fmt.Errorf("error reading request body: %w", err)
	}

	if err = json.Unmarshal(b, &dest); err != nil {
		return fmt.Errorf("error unmarshaling body to destination: %w", err)
	}

	return nil
}

/*
WriteJSON writes JSON content to the response writer.
*/
func (h HttpHelpers) WriteJSON(w http.ResponseWriter, status int, value interface{}) {
	var (
		err error
		b   []byte
	)

	w.Header().Set("Content-Type", "application/json")

	if b, err = json.Marshal(value); err != nil {
		b, _ = json.Marshal(struct {
			Message    string `json:"message"`
			Suggestion string `json:"suggestion"`
		}{
			Message:    "Error marshaling value for writing",
			Suggestion: "See error log for more information",
		})

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "%s", string(b))
		return
	}

	if status > 299 {
		w.WriteHeader(status)
	}

	_, _ = fmt.Fprintf(w, "%s", string(b))
}

func (h HttpHelpers) JsonOK(w http.ResponseWriter, value interface{}) {
	h.WriteJSON(w, http.StatusOK, value)
}

func (h HttpHelpers) JsonBadRequest(w http.ResponseWriter, value interface{}) {
	h.WriteJSON(w, http.StatusBadRequest, value)
}

func (h HttpHelpers) JsonInternalServerError(w http.ResponseWriter, value interface{}) {
	h.WriteJSON(w, http.StatusInternalServerError, value)
}

func (h HttpHelpers) JsonError(w http.ResponseWriter, message string) {
	result := make(map[string]string)
	result["message"] = message

	h.JsonInternalServerError(w, result)
}
