package adminapi

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
		// todo: separate queryRequest from the JSON request in the end
		queryRequest: queryRequest{
			Filters:    map[string]any{},
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

func (q *Query) AddFilter(attribute string, filter any) {
	q.queryRequest.Filters[attribute] = filter
}

// Count matching SA objects
func (q *Query) Count() (int, error) {
	err := q.load()
	if err != nil {
		return 0, err
	}

	return len(q.serverObjects), nil
}

// All returns all matching SA objects
func (q *Query) All() ([]*ServerObject, error) {
	err := q.load()
	if err != nil {
		return nil, err
	}

	return q.serverObjects, nil
}

// One returns exactly one matching SA object. If there is none or more than one, an error is returned.
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

	// always add "object_id" as attribute as we need it to modify the object
	if !containsString(q.queryRequest.Restricted, "object_id") {
		q.queryRequest.Restricted = append(q.queryRequest.Restricted, "object_id")
	}

	resp, err := sendRequest(apiEndpointQuery, q.queryRequest)
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

// NewServer creates a new server object (fetches default attributes from SA)
func NewServer(serverType string) (*ServerObject, error) {
	// todo urlencode
	resp, err := sendRequest(apiEndpointNewObject+"?servertype="+serverType, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	server := &ServerObject{}
	err = json.NewDecoder(resp.Body).Decode(&server.attributes)

	return server, err
}

// like {"Filters": {"hostname": {"Regexp": "foo.local.*"}}, "restrict": ["hostname", "object_id"]}
type queryRequest struct {
	Filters    map[string]any `json:"filters"`
	Restricted []string       `json:"restrict"`
	OrderBy    string         `json:"order_by,omitempty"`
}

// like {"status": "success", "result": [{"object_id": 483903, "hostname": "foo.local"}]}
type queryResponse struct {
	Status string           `json:"status"`
	Result []map[string]any `json:"result"`
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
