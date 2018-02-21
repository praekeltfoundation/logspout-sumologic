package sumologic

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gliderlabs/logspout/router"
	"github.com/gojektech/heimdall"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	router.AdapterFactories.Register(NewSumoLogicAdapter, "sumologic")
}

type SumoLogicAdapter struct {
	route  *router.Route
	client heimdall.Client
	config *SumoLogicConfig
}

type SumoLogicConfig struct {
	endPoint       string
	sourceName     string
	sourceCategory string
	sourceHost     string
	retries        int64
	timeout        int64
	backoff        int64
}

func NewSumoLogicAdapter(route *router.Route) (router.LogAdapter, error) {

	config := buildConfig(route)
	timeoutInMillis := int(config.timeout)
	httpClient := heimdall.NewHTTPClient(timeoutInMillis)
	httpClient.SetRetryCount(int(config.retries))
	httpClient.SetRetrier(
		heimdall.NewRetrier(heimdall.NewConstantBackoff(config.backoff)))
	return &SumoLogicAdapter{
		route:  route,
		client: httpClient,
		config: config,
	}, nil
}

func buildConfig(route *router.Route) *SumoLogicConfig {
	config := &SumoLogicConfig{
		endPoint:       getopt("SUMOLOGIC_ENDPOINT", route.Address),
		sourceName:     getopt("SUMOLOGIC_SOURCE_NAME", "{{.Container.Name}}"),
		sourceCategory: getopt("SUMOLOGIC_SOURCE_CATEGORY", ""),
		sourceHost: getopt(
			"SUMOLOGIC_SOURCE_HOST", "{{.Container.Config.Hostname}}"),
		retries: getintopt("SUMOLOGIC_RETRIES", 2),
		backoff: getintopt("SUMOLOGIC_BACKOFF", 10),
		timeout: getintopt("SUMOLOGIC_TIMEOUT", 10),
	}
	log.Info("ENDPOINT: ", config.endPoint)
	return config
}

func getopt(name string, dfault string) string {
	value := os.Getenv(name)
	log.Info(name, ": ", value)
	if value == "" {
		value = dfault
	}
	return value
}

func getintopt(name string, dfault int64) int64 {
	value := os.Getenv(name)
	if value == "" {
		return dfault
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return intValue
}

func (s *SumoLogicAdapter) Stream(logstream chan *router.Message) {
	for msg := range logstream {
		go s.sendLog(msg)
	}
}

func (s *SumoLogicAdapter) sendLog(msg *router.Message) {

	headers := s.buildHeaders(msg)
	if strings.Contains(msg.Container.Name, "logspout") {
		return
	}
	if strings.Contains(msg.Container.Image, "logspout") {
		return
	}

	log.Debug("Sending LOG FROM", msg.Container.Name)
	fmt.Println("SEND LOG DATA", msg.Data)

	var r, err = s.client.Post(s.config.endPoint, strings.NewReader(msg.Data), headers)
	if err != nil {
		log.Error("Borked {0}", err)
		return
	}

	fmt.Println(r.Status)
	if r.StatusCode != http.StatusOK {
		errorf("Borked {0}", err)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorf("Borked {0}", err)
	}
	fmt.Println(string(body))
}

func (s *SumoLogicAdapter) buildHeaders(msg *router.Message) http.Header {
	headers := http.Header{}

	sourceName, nameErr := renderTemplate(msg, s.config.sourceName)
	if nameErr != nil {
		headers.Set("X-Sumo-Name", sourceName)
	}

	sourceHost, hostErr := renderTemplate(msg, s.config.sourceHost)
	if hostErr != nil {
		headers.Set("X-Sumo-Host", sourceHost)
	}

	if s.config.sourceCategory != "" {
		sourceCategory, catErr := renderTemplate(msg, s.config.sourceCategory)
		if catErr != nil {
			headers.Set("X-Sumo-Category", sourceCategory)
		}
	}
	return headers
}

func renderTemplate(msg *router.Message, text string) (string, error) {
	tmpl, err := template.New("info").Parse(text)
	if err != nil {
		return "", errorf("Couldn't parse sumologic source template. %v", err)
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, msg)
	if err != nil {
		return "", err
	}

	return buf.String(), nil

}

func errorf(format string, a ...interface{}) (err error) {
	err = fmt.Errorf(format, a...)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(err.Error())
	}
	return
}
