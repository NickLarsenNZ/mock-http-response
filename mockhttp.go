package mockhttp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/nicklarsennz/mockhttp/responders"
	"github.com/pkg/errors"
)

func NewClient(definitionsFilePath string) (*http.Client, error) {
	config, err := responders.ParseConfig(definitionsFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "parser error")
	}

	client := makeClient(config)

	return client, nil
}

func makeClient(config *responders.ResponderConfig) *http.Client {
	return &http.Client{
		Transport: &mockTransport{
			ResponderConfig: config,
		},
	}
}

type mockTransport struct {
	*responders.ResponderConfig
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return MatchResponse(req, t.ResponderConfig), nil
}

// Todo: move this to the responder package
// I think we need more clever matching, best match, not just first match
func MatchResponse(req *http.Request, config *responders.ResponderConfig) *http.Response {
	trimmedRequestBody := strings.Trim(bodyString(req), " \r\n")

	var fullPath = req.URL.Path
	if req.URL.RawQuery != "" {
		fullPath = fullPath + "?" + req.URL.RawQuery
	}
	fmt.Println(req.Method + " " + fullPath)

	// Loop through responders which match the method and path
	for _, responder := range config.Responders {

		// Skip if no match
		if responder.When.Http.Method != req.Method {
			continue
		}
		if responder.When.Http.Path != fullPath {
			continue
		}
		if !responder.When.Headers.AppearIn(req.Header) {
			continue
		}

		trimmedResponderBody := strings.Trim(responder.When.Body, " \r\n")
		if trimmedResponderBody != trimmedRequestBody {
			continue
		}

		var headers = make(http.Header)
		for k, v := range responder.Then.Headers {
			headers.Add(k, v)
		}
		return &http.Response{
			Status:     fmt.Sprintf("%d %s", responder.Then.Http.Status, responder.Then.Http.Message),
			StatusCode: responder.Then.Http.Status,
			Header:     headers,
			Body:       ioutil.NopCloser(bytes.NewBufferString(responder.Then.Body)),
		}
	}

	// Otherwise 404
	return &http.Response{
		Status: "404 Not Found",
		Header: http.Header{"X-NOOP": []string{"mockhttp failed to match"}},
		Body:   ioutil.NopCloser(bytes.NewBufferString("Error: mockhttp could not find the responder for the given conditions")),
	}
}

func bodyString(r *http.Request) string {
	if r.Body == nil {
		return ""
	}

	if body, err := ioutil.ReadAll(r.Body); err == nil {
		return string(body)
	}

	return ""

}
