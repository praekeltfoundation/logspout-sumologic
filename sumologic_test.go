package sumologic

import (
	"bytes"
	"os"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/suite"
)

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

func (ts *TestSuite) Test_build_configs_with_env_vars() {
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

func (ts *TestSuite) Test_build_configs_without_env_vars() {
	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	route := &router.Route{
		ID:      "foo",
		Address: "https://foo.collector.io/receiver/v1/http/Zm9vCg==",
		Adapter: "sumologic",
	}

	config := buildConfig(route)
	ts.Equal(expectedEndpoint, config.endPoint)
}

func (ts *TestSuite) Test_newAdapter_with_env_vars() {
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

func (ts *TestSuite) Test_render_template_with_empty_string() {
	msg := &router.Message{}
	value := ts.WithoutError(renderTemplate(msg, ""))
	ts.Equal("", value)
}

func (ts *TestSuite) Test_render_template_with_non_empty_string() {
	msg := &router.Message{}
	value := ts.WithoutError(renderTemplate(msg, "foo"))
	ts.Equal("foo", value)
}

func (ts *TestSuite) Test_render_template_with_template_string() {
	msg := &router.Message{
		Container: &docker.Container{Name: "foo"},
	}
	value := ts.WithoutError(renderTemplate(msg, "{{.Container.Name}}"))
	ts.Equal("foo", value)
}
