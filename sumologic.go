package sumologic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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

type SumologicData struct {
	Message   string         `json:"message"`
	Container *ContainerData `json:"container"`
	Timestamp string         `json:"timestamp"`
}

type ContainerData struct {
	Time     string `json:"time"`
	Source   string `json:"source"`
	Name     string `json:"docker_name"`
	ID       string `json:"docker_id"`
	Image    string `json:"docker_image"`
	Hostname string `json:"docker_hostname"`
}

// NewSumoLogicAdapter provides a SumoLogicAdapter
// to the logspout adapter factory.
func NewSumoLogicAdapter(route *router.Route) (router.LogAdapter, error) {

	config := buildConfig(route)

	timeoutInMillis := time.Duration(config.timeout) * time.Millisecond
	httpClient := heimdall.NewHTTPClient(timeoutInMillis)
	httpClient.SetRetrier(
		heimdall.NewRetrier(heimdall.NewConstantBackoff(config.backoff)))
	httpClient.SetRetryCount(int(config.retries))

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
		timeout: getintopt("SUMOLOGIC_TIMEOUT_MS", 10000),
	}
	return config
}

// getopt retrieves an environment variable if it's set to
// a non-emty string.
// The supplied default is returned otherwise.
func getopt(name string, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

// getoptint retrieves an environment variable as an int if it's set
// to a non-empty string.
// The supplied default int is returned otherwise.
func getintopt(name string, dfault int64) int64 {
	value := os.Getenv(name)
	if value == "" {
		return dfault
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.WithError(err).WithField(name, value).Error("Failed to parse")
		return dfault
	}
	return intValue
}

// Stream is a logspout adapter implementation method.
func (s *SumoLogicAdapter) Stream(logstream chan *router.Message) {
	for msg := range logstream {
		go s.sendLog(msg)
	}
}

// sendLog post a log to Sumologic
func (s *SumoLogicAdapter) sendLog(msg *router.Message) {

	headers := s.buildHeaders(msg)
	data := s.buildData(msg)

	strData, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).WithField("message_source", msg.Source).Errorf(
			"Unable to build json data, skipping send")
		return
	}

	req, reqErr := s.client.Post(
		s.config.endPoint, strings.NewReader(string(strData)), headers)
	if reqErr != nil {
		log.WithError(reqErr).Error("Failed to send log to Sumologic")
		return
	}

	_, err = ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Error("Unable to read response body.")
	}
	req.Body.Close()
	if req.StatusCode != http.StatusOK {
		log.WithField(
			"StatusCode", req.StatusCode).Error("Failed to send log to Sumologic")
		return
	}

}

// buildHeaders creates a set of Sumologic classification headers,
// these header values are derived from env vars and/or container properties,
// then renderTemplate is called to compile for e.g {{.Container.Name}}
func (s *SumoLogicAdapter) buildHeaders(msg *router.Message) http.Header {
	headers := http.Header{}

	sourceName, nameErr := renderTemplate(msg, s.config.sourceName)
	if nameErr == nil {
		headers.Add("X-Sumo-Name", sourceName)
	}

	sourceHost, hostErr := renderTemplate(msg, s.config.sourceHost)
	if hostErr == nil {
		headers.Add("X-Sumo-Host", sourceHost)
	}

	if s.config.sourceCategory != "" {
		sourceCategory, catErr := renderTemplate(msg, s.config.sourceCategory)
		if catErr == nil {
			headers.Add("X-Sumo-Category", sourceCategory)
		}
	}
	return headers
}

// buildData builds the message to send to sumologic.
func (s *SumoLogicAdapter) buildData(msg *router.Message) *SumologicData {
	container := &ContainerData{
		Source:   msg.Source,
		Time:     msg.Time.Format(time.RFC3339),
		Name:     msg.Container.Name,
		ID:       msg.Container.ID,
		Image:    msg.Container.Config.Image,
		Hostname: msg.Container.Config.Hostname,
	}
	return &SumologicData{
		Container: container,
		Message:   msg.Data,
		// Sumologic supports 13 digit/UnixMilli in json messages.
		Timestamp: strconv.FormatInt(msg.Time.UTC().UnixNano()/1000000, 10),
	}
}

// renderTemplate compiles a template string, e.g {{.Container.Name}} using
// a router.Message as the context.
func renderTemplate(msg *router.Message, text string) (string, error) {
	tmpl, err := template.New("info").Parse(text)
	if err != nil {
		return "", fmt.Errorf("Couldn't parse sumologic source template. %v", err)
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, msg)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
