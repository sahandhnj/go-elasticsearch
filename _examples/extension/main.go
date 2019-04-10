// +build ignore

package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/estransport"
)

const port = "9209"

// ExtendedClient allows to call custom APIs via the elasticsearch client.
//
type ExtendedClient struct {
	*elasticsearch.Client
	Custom *ExtendedAPI
}

type ExtendedAPI struct {
	*elasticsearch.Client
}

// CatExample calls the custom REST API, "/_cat/example".
//
func (c *ExtendedAPI) Example() (*esapi.Response, error) {
	req, _ := http.NewRequest("GET", "/_cat/example", nil) // errcheck exclude

	res, err := c.Perform(req)
	if err != nil {
		return nil, err
	}

	return &esapi.Response{StatusCode: res.StatusCode, Body: res.Body, Header: res.Header}, nil
}

func main() {
	log.SetFlags(0)

	var (
		signal = make(chan bool)
	)

	// --> Launch proxy server
	//
	go launchServer(signal)

	esclient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:" + port},
		Logger:    &estransport.ColorLogger{Output: os.Stdout, EnableRequestBody: true, EnableResponseBody: true},
	})
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	es := ExtendedClient{Client: esclient, Custom: &ExtendedAPI{esclient}}
	<-signal

	// --> Call a regular Elasticsearch API
	//
	es.Cat.Health()

	// --> Call a custom API
	//
	es.Custom.Example()
}

func launchServer(signal chan<- bool) {
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "localhost:9200"})

	// Respond with custom content on "GET /_cat/example", proxy to Elasticsearch for other requests
	//
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/_cat/example" {
			io.WriteString(w, "Hello from Cat Example action")
			return
		}
		proxy.ServeHTTP(w, r)
	})

	ln, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		log.Fatalf("Unable to start server: %s", err)
	}

	signal <- true
	go http.Serve(ln, nil)
}
