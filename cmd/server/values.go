// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	moovhttp "github.com/moov-io/base/http"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
)

// accumulator is a case-insensitve collector for string values.
//
// getValues() will return an orderd distinct array of accumulated strings
// where each string is the first seen instance.
type accumulator struct {
	limit  int
	values map[string]string
}

func newAccumulator(limit int) accumulator {
	return accumulator{
		limit:  limit,
		values: make(map[string]string),
	}
}

func (acc accumulator) add(value string) {
	if len(acc.values) >= acc.limit {
		return
	}

	norm := strings.ToLower(strings.TrimSpace(value))
	if norm == "" {
		return
	}
	if _, exists := acc.values[norm]; !exists {
		acc.values[norm] = value
	}
}

func (acc accumulator) getValues() []string {
	var out []string
	for _, v := range acc.values {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func addValuesRoutes(logger log.Logger, r *mux.Router, searcher *searcher) {
	r.Methods("GET").Path("/ui/values/{key}").HandlerFunc(getValues(logger, searcher))
}

func getKey(r *http.Request) string {
	v, _ := mux.Vars(r)["key"]
	return strings.ToLower(v)
}

func getValues(logger log.Logger, searcher *searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w = wrapResponseWriter(logger, w, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		acc := newAccumulator(extractSearchLimit(r))
		for i := range searcher.SDNs {
			// If we add support for other filters (CallSign, Tonnage)
			// then we should add those keys here.
			switch k := getKey(r); k {
			case "sdntype":
				acc.add(searcher.SDNs[i].SDNType)
			case "program":
				acc.add(searcher.SDNs[i].Program)
			default:
				moovhttp.Problem(w, fmt.Errorf("unknown key: %s", k))
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(acc.getValues())
	}
}
