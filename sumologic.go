package sumologic

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gliderlabs/logspout/router"
	"github.com/gojektech/heimdall"
)

func init() {
	router.AdapterFactories.Register(NewSumoLogicAdapter, "sumologic")
}

type SumoLogicAdapter struct {
	route  *router.Route
	client heimdall.Client
}

func NewSumoLogicAdapter(route *router.Route) (router.LogAdapter, error) {

	timeoutInMillis := 1000
	httpClient := heimdall.NewHTTPClient(timeoutInMillis)
	httpClient.SetRetryCount(2)
	httpClient.SetRetrier(heimdall.NewRetrier(heimdall.NewConstantBackoff(10)))

	return &SumoLogicAdapter{
		route:  route,
		client: httpClient,
	}, nil
}

func (s *SumoLogicAdapter) Stream(logstream chan *router.Message) {
	for raw_message := range logstream {
		headers := http.Header{}
		headers.Set("X-Sumo-Name", "foo")
		headers.Set("X-Sumo-Host", "bar")
		headers.Set("X-Sumo-Category", "baz")
		var r, err = s.client.Post("https://httpbin.org/post", strings.NewReader(raw_message.Data), headers)
		if err != nil {
			errorf("Borked {0}", err)
		}
		fmt.Println(r.Status)

	}
}

func errorf(format string, a ...interface{}) (err error) {
	err = fmt.Errorf(format, a...)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(err.Error())
	}
	return
}
