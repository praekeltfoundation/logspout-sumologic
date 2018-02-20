package sumologic

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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

	timeoutInMillis := 10 * time.Second
	httpClient := heimdall.NewHTTPClient(timeoutInMillis)
	httpClient.SetRetryCount(2)
	httpClient.SetRetrier(heimdall.NewRetrier(heimdall.NewConstantBackoff(10)))

	return &SumoLogicAdapter{
		route:  route,
		client: httpClient,
	}, nil
}

func (s *SumoLogicAdapter) Stream(logstream chan *router.Message) {
	for msg := range logstream {

		go s.SendLog(msg)

	}
}

func (s *SumoLogicAdapter) SendLog(msg *router.Message) {
	headers := http.Header{}
	headers.Set("X-Sumo-Name", "foo")
	headers.Set("X-Sumo-Host", "bar")
	headers.Set("X-Sumo-Category", "baz")
	fmt.Println("SEND LOG {0}", msg.Data)
	var r, err = s.client.Post("https://httpbin.org/post", strings.NewReader(msg.Data), headers)
	if err != nil {
		errorf("Borked {0}", err)
	}

	fmt.Println(r.Status)
}

func errorf(format string, a ...interface{}) (err error) {
	err = fmt.Errorf(format, a...)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(err.Error())
	}
	return
}
