// Copyright 2021 Contributors to the Veraison project.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"encoding/json"
	"fmt"

	"github.com/ohler55/ojg/jp"
)

// QueryArgs is a map of key-value pairs of arguments for a query
type QueryArgs map[string]interface{}

// QueryResult is a slice of all matching results
type QueryResult []interface{}

// Query defines the query function signature. A query functions takes
// QueryArgs as a parameter and returns a QueryResult and/or an error. The
// QueryResult contain the query's matches.
type Query func(QueryArgs) (QueryResult, error)

// QueryConstraint defines a constraint on the number of expected matches in
// the QueryResult.
type QueryConstraint int

const (

	// QcNone indicates no constraint on the number of matches.
	QcNone = QueryConstraint(iota)

	// QcZero indicates that there must not be any matches for the query.
	QcZero

	// QcOne indicates that the query must result in exactly one match.
	QcOne

	// QcOneOrMore indicates the query must result in at least one match.
	QcOneOrMore

	// QcMultiple indicates the query must must result in multiple matches.
	QcMultiple
)

// QueryDescriptor describes the query to be executed.
type QueryDescriptor struct {

	// Name indicates which query is to be executed. This is used to match
	// the Query function from the ones registered with the store.
	Name string // Name of the query to run

	// Args contains the query parameters that will be used when executing
	// the query. What parameters are valid depends on the query name.
	Args QueryArgs //map param

	// Constraint defines the constraint on the result generated by query
	// (see QueryConstraint above).
	Constraint QueryConstraint
}

// PopulateQueryDescriptor populates the provided QueryDescriptor with
// specified name and the parameters extracted from the claims based on the
// JSONpath's in the provided params map.
func PopulateQueryDescriptor(
	claims map[string]interface{},
	name string, // name of the query to run
	params map[string]string, // parameter names mapped onto the locations of their values in the claims
	qd *QueryDescriptor,
) error {
	qd.Name = name
	qd.Args = make(map[string]interface{})

	for pName, pPath := range params {
		expr, err := jp.ParseString(pPath)
		if err != nil {
			return fmt.Errorf("could not parse query param path: %v", err)
		}

		qd.Args[pName] = expr.Get(claims)
	}

	return nil
}

// ParseQueryDescriptors generates QueryDescriptor's based on the query spec
// provided by the Policy and the evidence claims. The Spec specifies the query
// names and JSONpath's used to extract parameter values from the claims.
func ParseQueryDescriptors(claims map[string]interface{}, data []byte) ([]*QueryDescriptor, error) {
	var qds []*QueryDescriptor
	var unmarshaledData interface{}

	err := json.Unmarshal(data, &unmarshaledData)
	if err != nil {
		return nil, err
	}

	var querySpecs map[string]interface{}

	switch v := unmarshaledData.(type) {
	case map[string]interface{}:
		querySpecs = v
	default:
		return nil, fmt.Errorf("unexpected type for unmashaled query specs; must be a JSON object")
	}

	for queryName, unmarshaledArgsSpec := range querySpecs {
		qd := new(QueryDescriptor)
		qd.Constraint = QcNone
		argsSpec := make(map[string]string)

		switch v := unmarshaledArgsSpec.(type) {
		case map[string]string:
			argsSpec = v
		case map[string]interface{}:
			for key, val := range v {
				switch v1 := val.(type) {
				case string:
					argsSpec[key] = v1
				default:
					return nil, fmt.Errorf("query arg spec value must be a string")
				}
			}
		default:
			return nil, fmt.Errorf("unexpected type for unmashaled query arg specs; must be a JSON object with string values")
		}

		if err := PopulateQueryDescriptor(claims, queryName, argsSpec, qd); err != nil {
			return nil, err
		}
		qds = append(qds, qd)
	}

	return qds, nil
}

// QueryDescriptorsByName implements sort.Interface for sorting query descriptors by name.
type QueryDescriptorsByName []*QueryDescriptor

// Len returns the length of QueryDescriptor slice to be sorted.
func (q QueryDescriptorsByName) Len() int {
	return len(q)
}

// Less returns true iff the QueryDescriptor at index i compares as "less" than the
// one at index j, based on their names.
func (q QueryDescriptorsByName) Less(i, j int) bool {
	return q[i].Name < q[j].Name
}

// Swap swaps QueryDescriptors at the specified indexes.
func (q QueryDescriptorsByName) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}
