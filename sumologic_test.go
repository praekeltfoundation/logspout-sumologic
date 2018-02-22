package sumologic

import (
	"os"
	"strings"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
)

func Test_getopt_unset_envar_returns_default(t *testing.T) {

	expectedValue := "foo"

	value := getopt("UNSET_ENV_VAR", "foo")
	if !strings.EqualFold(expectedValue, value) {
		t.Fatal("expected equal values")
	}

}

func Test_getopt_set_envar_empty_returns_default(t *testing.T) {

	os.Setenv("SET_ENV_VAR", "")
	expectedValue := "foo"

	value := getopt("SET_ENV_VAR", "foo")
	if !strings.EqualFold(expectedValue, value) {
		t.Fatal("expected equal values")
	}

}

func Test_getopt_set_envar_nonempty_returns_value(t *testing.T) {

	os.Setenv("SET_ENV_VAR", "foo")
	expectedValue := "foo"

	value := getopt("SET_ENV_VAR", "bar")
	if !strings.EqualFold(expectedValue, value) {
		t.Fatal("expected equal values")
	}

}

func Test_build_configs_with_env_vars(t *testing.T) {

	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	os.Setenv("SUMOLOGIC_ENDPOINT", expectedEndpoint)
	defer os.Unsetenv("SUMOLOGIC_ENDPOINT")
	route := &router.Route{
		ID:      "foo",
		Address: "sumologic://",
		Adapter: "sumologic",
	}

	config := buildConfig(route)
	if !strings.EqualFold(expectedEndpoint, config.endPoint) {
		t.Fatal("expected equal endpoint addrs")
	}

}

func Test_build_configs_without_env_vars(t *testing.T) {

	expectedEndpoint := "https://foo.collector.io/receiver/v1/http/Zm9vCg=="
	route := &router.Route{
		ID:      "foo",
		Address: "https://foo.collector.io/receiver/v1/http/Zm9vCg==",
		Adapter: "sumologic",
	}

	config := buildConfig(route)
	if !strings.EqualFold(expectedEndpoint, config.endPoint) {
		t.Fatal("expected equal endpoint addrs")
	}

}

func Test_render_template_with_empty_string(t *testing.T) {

	expectedValue := ""
	msg := &router.Message{}
	value, err := renderTemplate(msg, "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.EqualFold(expectedValue, value) {
		t.Fatal("expected equal values")
	}
}

func Test_render_template_with_non_empty_string(t *testing.T) {

	expectedValue := "foo"
	msg := &router.Message{}
	value, err := renderTemplate(msg, "foo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.EqualFold(expectedValue, value) {
		t.Fatal("expected equal values")
	}
}

func Test_render_template_with_template_string(t *testing.T) {

	expectedValue := "foo"
	msg := &router.Message{
		Container: &docker.Container{Name: "foo"},
	}
	value, err := renderTemplate(msg, "{{.Container.Name}}")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.EqualFold(expectedValue, value) {
		t.Fatal("expected equal values")
	}
}
