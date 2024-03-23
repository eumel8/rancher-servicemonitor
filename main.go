package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rancher/norman/clientbase"
	//clusterClient "github.com/rancher/rancher/pkg/client/generated/cluster/v3"
	managementClient "github.com/rancher/rancher/pkg/client/generated/management/v3"
	//projectClient "github.com/rancher/rancher/pkg/client/generated/project/v3"
)

// Client are the client kind for a Rancher v3 API
type Client struct {
	Management *managementClient.Client
	//CatalogV2  map[string]*clientbase.APIBaseClient
	//Cluster    map[string]*clusterClient.Client
	//Project    map[string]*projectClient.Client
}

// Config is the configuration parameters for a Rancher v3 API
type Config struct {
	TokenKey             string `json:"tokenKey"`
	URL                  string `json:"url"`
	CACerts              string `json:"cacert"`
	Insecure             bool   `json:"insecure"`
	Bootstrap            bool   `json:"bootstrap"`
	ClusterID            string `json:"clusterId"`
	ProjectID            string `json:"projectId"`
	Timeout              time.Duration
	RancherVersion       string
	K8SDefaultVersion    string
	K8SSupportedVersions []string
	Sync                 sync.Mutex
	Client               Client
}

func (c *Config) CreateClientOpts() *clientbase.ClientOpts {
	options := &clientbase.ClientOpts{
		URL:      c.URL,
		TokenKey: c.TokenKey,
		CACerts:  c.CACerts,
		Insecure: c.Insecure,
	}
	return options
}

// ManagementClient creates a Rancher client scoped to the management API
func (c *Config) ManagementClient() (*managementClient.Client, error) {

	// Setup the management client
	options := c.CreateClientOpts()
	//options.URL = options.URL + rancher2ClientAPIVersion
	mClient, err := managementClient.NewClient(options)
	if err != nil {
		return nil, err
	}
	c.Client.Management = mClient

	return c.Client.Management, nil
}

func (c *Config) getData() (int, error) {
	return c.getProjectCount()
}

func main() {
	// Create a new HTTP server
	server := &http.Server{
		Addr: ":8080",
	}

	// Handle the /pods endpoint
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Get the number of Pods in the cluster
		config := &Config{}

		projectCount, err := config.getData()
		if err != nil {
			http.Error(w, "Failed to get the number of Pods", http.StatusInternalServerError)
			return
		}

		// Write the number of Pods to the response in Prometheus format
		fmt.Fprintf(w, "pod_count %d\n", projectCount)

	})

	// Start the server in a separate goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	// Wait for termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Shutdown the server gracefully
	server.Shutdown(nil)
}

//func (c *Config) getProjectCount(clusterID string) (int, error) {

func (c *Config) getProjectCount() (int, error) {

	projects, err := c.Client.Management.Project.List(nil)
	if err != nil {
		return 0, err
	}

	projectCount := len(projects.Data)
	return projectCount, nil
}
