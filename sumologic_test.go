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

func (ts *TestSuite) AddCleanup(f func()) {
	ts.cleanups = append([]func(){f}, ts.cleanups...)
}

func (ts *TestSuite) TearDownTest() {
	for _, f := range ts.cleanups {
		f()
	}
}

func (ts *TestSuite) Setenv(name string, value string) {
	ts.NoError(os.Setenv(name, value))
	ts.AddCleanup(func() { ts.NoError(os.Unsetenv(name)) })
}

func (ts *TestSuite) CaptureLogs() (*test.Hook, *bytes.Buffer) {
	origOut := logrus.StandardLogger().Out
	hook := test.NewGlobal()
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)
	ts.AddCleanup(func() { logrus.SetOutput(origOut) })
	return hook, &buffer
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

func (ts *TestSuite) Test_render_template_with_empty_string() {
	msg := &router.Message{}
	value, err := renderTemplate(msg, "")
	if ts.NoError(err) {
		ts.Equal("", value)
	}
}

func (ts *TestSuite) Test_render_template_with_non_empty_string() {
	msg := &router.Message{}
	value, err := renderTemplate(msg, "foo")
	if ts.NoError(err) {
		ts.Equal("foo", value)
	}
}

func (ts *TestSuite) Test_render_template_with_template_string() {
	msg := &router.Message{
		Container: &docker.Container{Name: "foo"},
	}
	value, err := renderTemplate(msg, "{{.Container.Name}}")
	if ts.NoError(err) {
		ts.Equal("foo", value)
	}
}
