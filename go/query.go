package main

import (
	"encoding/json"
	"fmt"
)

type Query struct {
	queryRequest  queryRequest
	serverObjects []*ServerObject
	loaded        bool
}

// NewQuery initialize a new query which loads data from SA if needed
func NewQuery() Query {
	return Query{
		queryRequest: queryRequest{
			Filters:    map[string]map[string]string{},
			Restricted: []string{"hostname"},
		},
		serverObjects: []*ServerObject{},
	}
}

func (q *Query) SetAttributes(attributes []string) {
	q.queryRequest.Restricted = attributes
}

func (q *Query) OrderBy(attribute string) {
	q.queryRequest.OrderBy = attribute
}

func (q *Query) AddFilter(attribute string, filterType string, value string) {
	// todo real filters, like Regexp(string) or Not(filter)
	q.queryRequest.Filters[attribute] = map[string]string{}
	q.queryRequest.Filters[attribute][filterType] = value
}

func (q *Query) All() ([]*ServerObject, error) {
	err := q.load()
	if err != nil {
		return nil, err
	}

	return q.serverObjects, nil
}

func (q *Query) One() (*ServerObject, error) {
	err := q.load()
	if err != nil {
		return nil, err
	}

	if len(q.serverObjects) != 1 {
		return nil, fmt.Errorf("expected exactly one server object, got %d", len(q.serverObjects))
	}

	return q.serverObjects[0], nil
}

func (q *Query) load() error {
	if q.loaded {
		return nil
	}

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	// todo: always add object_id as attribute
	resp, err := sendRequest(apiEndpointQuery, cfg, q.queryRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respServer := queryResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respServer)

	// map attribute map into ServerObject objects
	q.serverObjects = make([]*ServerObject, len(respServer.Result))
	for idx, object := range respServer.Result {
		q.serverObjects[idx] = &ServerObject{
			attributes: object,
		}
	}
	q.loaded = true

	return err
}

// like {"Filters": {"hostname": {"Regexp": "de1w1.foe.*"}}, "restrict": ["hostname", "object_id"]}
type queryRequest struct {
	Filters    map[string]map[string]string `json:"filters"`
	Restricted []string                     `json:"restrict"`
	OrderBy    string                       `json:"order_by,omitempty"`
}

// like {"status": "success", "result": [{"object_id": 483903, "hostname": "de1w1.foe.ig.local"}]}
type queryResponse struct {
	Status string              `json:"status"`
	Result []map[string]string `json:"result"`
}
