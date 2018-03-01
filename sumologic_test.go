package sumologic

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/suite"
)

// jsonobj is an alias for type a JSON object gets unmarshalled into, because
// building nested map[string]interface{}{ ... } literals is awful.
type jsonobj = map[string]interface{}

// Test suite machinery.

type TestSuite struct {
	suite.Suite
	cleanups []func()
}

func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// AddCleanup schedules the given cleanup function to be run after the test.
// Think of it like `defer`, except it applies to the whole test rather than
// the specific function it appears in.
func (ts *TestSuite) AddCleanup(f func()) {
	ts.cleanups = append([]func(){f}, ts.cleanups...)
}

func (ts *TestSuite) TearDownTest() {
	for _, f := range ts.cleanups {
		f()
	}
}

// Setenv sets an environment variable for the duration of the test. It
// immediately fails the test if it is unable to set the envvar.
func (ts *TestSuite) Setenv(name string, value string) {
	ts.Require().NoError(os.Setenv(name, value))
	ts.AddCleanup(func() { ts.NoError(os.Unsetenv(name)) })
}

// CaptureLogs sets up log capturing machinery. It also redirects the log
// output to a buffer for the duration of the test to avoid spamming the
// console with random logs entries.
func (ts *TestSuite) CaptureLogs() (*test.Hook, *bytes.Buffer) {
	// TODO: Remove the hook after the test?
	origOut := logrus.StandardLogger().Out
	hook := test.NewGlobal()
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)
	ts.AddCleanup(func() { logrus.SetOutput(origOut) })
	return hook, &buffer
}

// WithoutError accepts a (result, error) pair, immediately fails the test if
// there is an error, and returns just the result if there is no error. It
// accepts and returns the result value as an `interface{}`, so it may need to
// be cast back to whatever type it should be afterwards.
func (ts *TestSuite) WithoutError(result interface{}, err error) interface{} {
	ts.Require().NoError(err)
	return result
}

// ReadJSON reads a JSON object from an io.Reader and returns a jsonobj,
// immediately failing the test if there is an error.
func (ts *TestSuite) ReadJSON(reader io.Reader) jsonobj {
	var obj jsonobj
	ts.Require().NoError(json.NewDecoder(reader).Decode(&obj))
	return obj
}

// Tests.

func (ts *TestSuite) Test_getopt_unset_envar_returns_default() {
	ts.EqualValues("foo", getopt("UNSET_ENV_VAR", "foo"))
}

func (ts *TestSuite) Test_getopt_set_envar_empty_returns_default() {
	ts.Setenv("SET_ENV_VAR", "")
	ts.EqualValues("foo", getopt("SET_ENV_VAR", "foo"))
}

func (ts *TestSuite) Test_getopt_set_envar_nonempty_returns_value() {
	ts.Setenv("SET_ENV_VAR", "foo")
	ts.EqualValues("foo", getopt("SET_ENV_VAR", "bar"))
}

func (ts *TestSuite) Test_getintopt_unset_envar_returns_default() {
	ts.EqualValues(1, getintopt("UNSET_ENV_VAR", 1))
}

func (ts *TestSuite) Test_getintopt_set_envar_empty_returns_default() {
	ts.Setenv("SET_ENV_VAR", "")
	ts.EqualValues(1, getintopt("SET_ENV_VAR", 1))
}

func (ts *TestSuite) Test_getintopt_set_envar_nonempty_returns_value() {
	ts.Setenv("SET_ENV_VAR", "2")
	ts.EqualValues(2, getintopt("SET_ENV_VAR", 1))
}

func (ts *TestSuite) Test_getintopt_set_envar_invalid_returns_default() {
	hook, _ := ts.CaptureLogs()

	ts.Setenv("SET_ENV_VAR", "seven")
	ts.EqualValues(1, getintopt("SET_ENV_VAR", 1))
	ts.Equal(logrus.ErrorLevel, hook.LastEntry().Level)
	ts.Equal("Failed to parse", hook.LastEntry().Message)
}

func (ts *TestSuite) Test_buildConfig_with_empty_route() {
	config := buildConfig(&router.Route{})
	ts.Equal("", config.endPoint)
}

func (ts *TestSuite) Test_buildConfig_with_env_vars() {
	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	ts.Setenv("SUMOLOGIC_ENDPOINT", expectedEndpoint)
	route := &router.Route{
		ID:      "foo",
		Address: "sumologic://",
		Adapter: "sumologic",
	}

	config := buildConfig(route)
	ts.Equal(expectedEndpoint, config.endPoint)
}

func (ts *TestSuite) Test_buildConfig_without_env_vars() {
	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	route := &router.Route{
		ID:      "foo",
		Address: "https://foo.collector.io/receiver/v1/http/Zm9vCg==",
		Adapter: "sumologic",
	}

	config := buildConfig(route)
	ts.Equal(expectedEndpoint, config.endPoint)
}

func (ts *TestSuite) Test_NewAdapter_with_env_vars() {
	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	ts.Setenv("SUMOLOGIC_ENDPOINT", expectedEndpoint)
	route := &router.Route{
		ID:      "foo",
		Address: "sumologic://",
		Adapter: "sumologic",
	}

	adapter := ts.WithoutError(NewAdapter(route)).(*Adapter)
	ts.Equal(route, adapter.route)
	ts.Equal(expectedEndpoint, adapter.config.endPoint)
	// TODO: More assertions?
}

func (ts *TestSuite) Test_NewAdapter_without_env_vars() {
	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	route := &router.Route{
		ID:      "foo",
		Address: "https://foo.collector.io/receiver/v1/http/Zm9vCg==",
		Adapter: "sumologic",
	}

	adapter := ts.WithoutError(NewAdapter(route)).(*Adapter)
	ts.Equal(route, adapter.route)
	ts.Equal(expectedEndpoint, adapter.config.endPoint)
	// TODO: More assertions?
}

func (ts *TestSuite) Test_renderTemplate_with_empty_string() {
	msg := &router.Message{}
	value := ts.WithoutError(renderTemplate(msg, ""))
	ts.Equal("", value)
}

func (ts *TestSuite) Test_renderTemplate_with_non_empty_string() {
	msg := &router.Message{}
	value := ts.WithoutError(renderTemplate(msg, "foo"))
	ts.Equal("foo", value)
}

func (ts *TestSuite) Test_renderTemplate_with_template_string() {
	msg := &router.Message{
		Container: &docker.Container{Name: "foo"},
	}
	value := ts.WithoutError(renderTemplate(msg, "{{.Container.Name}}"))
	ts.Equal("foo", value)
}

func (ts *TestSuite) Test_renderTemplate_with_bad_template_string() {
	msg := &router.Message{}
	value, err := renderTemplate(msg, "{{")
	ts.EqualError(err,
		"Couldn't parse sumologic source template. template: "+
			"info:1: unexpected unclosed action in command")
	ts.Equal("", value)
}

func (ts *TestSuite) Test_renderTemplate_with_render_error() {
	msg := &router.Message{}
	value, err := renderTemplate(msg, "{{.Container.Name}}")
	ts.EqualError(err,
		"template: info:1:12: executing \"info\" at <.Container.Name>: "+
			"can't evaluate field Name in type *docker.Container")
	ts.Equal("", value)
}

func (ts *TestSuite) Test_buildData_with_empty_message() {
	msg := &router.Message{}
	ts.Panics(func() { buildData(msg) })
}

func (ts *TestSuite) Test_buildData_with_empty_container() {
	msg := &router.Message{
		Container: &docker.Container{},
	}
	ts.Panics(func() { buildData(msg) })
}

func (ts *TestSuite) Test_buildData_with_simple_message() {
	msg := &router.Message{
		Data: "Some data.",
		Container: &docker.Container{
			Name:   "foo",
			Config: &docker.Config{},
		},
	}
	data := buildData(msg)
	ts.Equal("foo", data.Container.Name)
	ts.Equal("Some data.", data.Message)
}

func (ts *TestSuite) Test_buildHeaders_with_empty_message() {
	msg := &router.Message{}
	config := buildConfig(&router.Route{})
	headers := buildHeaders(msg, config)
	ts.Equal(http.Header{}, headers)
}

func (ts *TestSuite) Test_buildHeaders_with_empty_container() {
	expectedHeaders := http.Header{}
	expectedHeaders.Add("X-Sumo-Name", "")

	msg := &router.Message{
		Container: &docker.Container{},
	}
	config := buildConfig(&router.Route{})
	headers := buildHeaders(msg, config)
	ts.Equal(expectedHeaders, headers)
}

func (ts *TestSuite) Test_buildHeaders_with_all_fields() {
	expectedHeaders := http.Header{}
	expectedHeaders.Add("X-Sumo-Name", "foo")
	expectedHeaders.Add("X-Sumo-Host", "example.com")
	expectedHeaders.Add("X-Sumo-Category", "feline")

	ts.Setenv("SUMOLOGIC_SOURCE_CATEGORY", "feline")
	msg := &router.Message{
		Container: &docker.Container{
			Name: "foo",
			Config: &docker.Config{
				Hostname: "example.com",
			},
		},
	}
	config := buildConfig(&router.Route{})
	headers := buildHeaders(msg, config)
	ts.Equal(expectedHeaders, headers)
}

func (ts *TestSuite) Test_sendLog_simple_message() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.Equal([]string{"box"}, r.Header["X-Sumo-Name"])
		ts.Equal([]string{"example.com"}, r.Header["X-Sumo-Host"])
		body := ts.ReadJSON(r.Body)
		ts.Equal(jsonobj{
			"message": "Some data.",
			// The timestamp is different every time, so just get it from the
			// body JSON.
			"timestamp": body["timestamp"],
			"container": jsonobj{
				"docker_name":     "box",
				"docker_hostname": "example.com",
				"docker_id":       "",
				"docker_image":    "",
				"time":            "0001-01-01T00:00:00Z",
				"source":          "",
			},
		}, body)
		// No response body.
	}))
	defer server.Close()

	adapter := ts.WithoutError(NewAdapter(&router.Route{
		ID:      "foo",
		Address: server.URL,
		Adapter: "sumologic",
	})).(*Adapter)

	msg := &router.Message{
		Data: "Some data.",
		Container: &docker.Container{
			Name: "box",
			Config: &docker.Config{
				Hostname: "example.com",
			},
		},
	}

	adapter.sendLog(msg)
}
