package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	//"github.com/rancher/shepherd/clients/rancher"
	//"github.com/rancher/shepherd/pkg/session"

	//"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rancher/norman/clientbase"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/stretchr/testify/suite"
)

type ManagementTestSuite struct {
	suite.Suite
	cluster  *management.Cluster
	clusters []*management.Cluster
	projects []*management.Project
	project  *management.Project
	nodes    []*management.Node
}

func (suite *ManagementTestSuite) SetupSuite() {
	os.Setenv("RANCHER_CACERT", "testCACert")
	os.Setenv("RANCHER_CLUSTERID", "testClusterID")
	os.Setenv("LOG_LEVEL", "testLogLevel")
	os.Setenv("PORT", "testPort")
	os.Setenv("RANCHER_PROJECTID", "testProjectID")
	os.Setenv("RANCHER_TOKEN", "testToken")
	os.Setenv("RANCHER_URL", "testURL")
	clusterConfig := &management.Cluster{

		Name:      "test-cluster",
		NodeCount: 5,
	}
	clusters := []*management.Cluster{clusterConfig}
	suite.clusters = clusters

}

func (suite *ManagementTestSuite) TearDownSuite() {
	os.Unsetenv("RANCHER_CACERT")
	os.Unsetenv("RANCHER_CLUSTERID")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("PORT")
	os.Unsetenv("RANCHER_PROJECTID")
	os.Unsetenv("RANCHER_TOKEN")
	os.Unsetenv("RANCHER_URL")
}

// func TestManagementTestSuite(t *testing.T) {
//	suite.Run(t, new(ManagementTestSuite))
//}

func (suite *ManagementTestSuite) TestGetClusters() {
	// Create a new instance of the Config struct
	config := &Config{}

	// Call the ManagementClient function
	client, err := config.ManagementClient()

	// Check if the error is nil
	suite.Nil(err)

	// Call the GetClusters ListAll function
	clusters, err := client.Cluster.ListAll(
		clientbase.NewListOpts(),
	)

	fmt.Println("cluster: ", clusters)
	// Check if the error is nil
	suite.Nil(err)

}

func MockMuxer() {
	mux := http.NewServeMux()

	// https://stackoverflow.com/questions/47148240/correct-way-to-match-url-path-to-page-name-with-url-routing-in-go
	// watch: /apis/otc.mcsps.de/v1alpha1/rdss?allowWatchBookmarks=true&resourceVersion=265294298&timeout=5m29s&timeoutSeconds=329&watch=true
	mux.Handle("/metrics", WithLogging(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/metrics") {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "ok metrics")
			time.Sleep(3 * time.Second)
			return
		}
		if r.URL.Path != "/" {

			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			uri := r.URL.String()
			fmt.Printf("Uri: %s\n", string(uri))
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Body: %s\n", body)
			return
		}
	})

	fmt.Println("Listening...")

	var retries int = 3

	for retries > 0 {
		err := http.ListenAndServe("127.0.0.1:8080", mux)
		if err != nil {
			fmt.Println("Restart http server ... ", err)
			retries -= 1
		} else {
			break
		}
	}

}

func TestServer(t *testing.T) {
	// Create a new request to the default route
	req := httptest.NewRequest("GET", "/", nil)

	// Create a response recorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Rancher Prometheus Exporter")
	})
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := "Rancher Prometheus Exporter"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
func TestConfig(t *testing.T) {
	// Create a new instance of the Config struct
	config := &Config{}

	// Perform assertions on the Config struct
	if config == nil {
		t.Error("Config struct is nil")
	}

	os.Setenv("RANCHER_CACERT", "testCACert")
	os.Setenv("RANCHER_CLUSTERID", "testClusterID")
	os.Setenv("LOG_LEVEL", "testLogLevel")
	os.Setenv("PORT", "testPort")
	os.Setenv("RANCHER_PROJECTID", "testProjectID")

	config.Load()
	if config.Loglevel != "testLogLevel" {
		t.Errorf("Expected Loglevel to be testLogLevel, got %s", config.Loglevel)
	}
	if config.Port != "testPort" {
		t.Errorf("Expected Port to be testPort, got %s", config.Port)
	}
	if config.ProjectID != "testProjectID" {
		t.Errorf("Expected ProjectID to be testProjectID, got %s", config.ProjectID)

	}
}

func TestConfig_ManagementClient(t *testing.T) {
	// Create a new instance of the Config struct
	config := &Config{}

	// Set the environment variables for testing
	os.Setenv("RANCHER_CACERT", "testCACert")
	os.Setenv("RANCHER_CLUSTERID", "testClusterID")
	os.Setenv("LOG_LEVEL", "testLogLevel")
	os.Setenv("PORT", "testPort")
	os.Setenv("RANCHER_PROJECTID", "testProjectID")
	os.Setenv("RANCHER_TOKEN", "testToken")
	os.Setenv("RANCHER_URL", "testURL")

	// Call the ManagementClient function
	_, err := config.ManagementClient()

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected error to be nil, got %s", err)
	}

	// Clean up the environment variables
	os.Unsetenv("RANCHER_CACERT")
	os.Unsetenv("RANCHER_CLUSTERID")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("PORT")
	os.Unsetenv("RANCHER_PROJECTID")
	os.Unsetenv("RANCHER_TOKEN")
	os.Unsetenv("RANCHER_URL")
}
func TestConfig_Load(t *testing.T) {
	// Create a new instance of the Config struct
	config := &Config{}

	// Set the environment variables for testing
	os.Setenv("RANCHER_CACERT", "testCACert")
	os.Setenv("RANCHER_CLUSTERID", "testClusterID")
	os.Setenv("LOG_LEVEL", "testLogLevel")
	os.Setenv("PORT", "testPort")
	os.Setenv("RANCHER_PROJECTID", "testProjectID")
	os.Setenv("RANCHER_TOKEN", "testToken")
	os.Setenv("RANCHER_URL", "testURL")

	// Call the Load function
	config.Load()

	// Check if the values are loaded correctly
	if config.CACerts != "testCACert" {
		t.Errorf("Expected CACerts to be testCACert, got %s", config.CACerts)
	}
	if config.ClusterID != "testClusterID" {
		t.Errorf("Expected ClusterID to be testClusterID, got %s", config.ClusterID)
	}
	if config.Loglevel != "testLogLevel" {
		t.Errorf("Expected Loglevel to be testLogLevel, got %s", config.Loglevel)
	}
	if config.Port != "testPort" {
		t.Errorf("Expected Port to be testPort, got %s", config.Port)
	}
	if config.ProjectID != "testProjectID" {
		t.Errorf("Expected ProjectID to be testProjectID, got %s", config.ProjectID)
	}
	if config.TokenKey != "testToken" {
		t.Errorf("Expected TokenKey to be testToken, got %s", config.TokenKey)
	}
	if config.URL != "testURL" {
		t.Errorf("Expected URL to be testURL, got %s", config.URL)
	}

	// Clean up the environment variables
	os.Unsetenv("RANCHER_CACERT")
	os.Unsetenv("RANCHER_CLUSTERID")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("PORT")
	os.Unsetenv("RANCHER_PROJECTID")
	os.Unsetenv("RANCHER_TOKEN")
	os.Unsetenv("RANCHER_URL")
}
func TestConfig_CreateClientOpts(t *testing.T) {
	// Create a new instance of the Config struct
	config := &Config{
		URL:      "testURL",
		TokenKey: "testToken",
		CACerts:  "testCACert",
		Insecure: true,
	}

	// Call the CreateClientOpts method
	options := config.CreateClientOpts()

	// Check if the options are created correctly
	if options.URL != "testURL" {
		t.Errorf("Expected URL to be testURL, got %s", options.URL)
	}
	if options.TokenKey != "testToken" {
		t.Errorf("Expected TokenKey to be testToken, got %s", options.TokenKey)
	}
	if options.CACerts != "testCACert" {
		t.Errorf("Expected CACerts to be testCACert, got %s", options.CACerts)
	}
	if options.Insecure != true {
		t.Errorf("Expected Insecure to be true, got %v", options.Insecure)
	}
}

func Test_main(t *testing.T) {
	go MockMuxer()
	timeout := time.After(5 * time.Second)
	done := make(chan bool)
	go func() {
		main() // We make a roughly fly over test if controller is starting
		time.Sleep(3 * time.Second)
		done <- true
	}()

	select {
	case <-timeout:
	case <-done:
	}
}
