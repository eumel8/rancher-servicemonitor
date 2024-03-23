package main

import (
	"context"
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

type Counter struct {
	Clusters int
	Nodes    int
	Projects int
	Token    int
	Users    int
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

// load config from environment variables
func (c *Config) Load() {
	c.URL = os.Getenv("RANCHER_URL")
	c.TokenKey = os.Getenv("RANCHER_TOKEN")
	c.CACerts = os.Getenv("RANCHER_CACERT")
	c.ClusterID = os.Getenv("RANCHER_CLUSTERID")
	c.ProjectID = os.Getenv("RANCHER_PROJECTID")
}

// ManagementClient creates a Rancher client scoped to the management API
func (c *Config) ManagementClient() (*managementClient.Client, error) {
	// Load the configuration
	c.Load()
	options := &clientbase.ClientOpts{
		URL:      c.URL,
		TokenKey: c.TokenKey,
		CACerts:  c.CACerts,
		Insecure: c.Insecure,
	}

	mClient, err := managementClient.NewClient(options)
	if err != nil {
		return nil, err
	}
	c.Client.Management = mClient
	return c.Client.Management, nil
}

func (c *Config) getData() (Counter, error) {
	fmt.Println("Getting data ...")

	clusters, err := c.getClusterCount()
	if err != nil {
		return Counter{}, err
	}
	Counter := Counter{
		Clusters: clusters,
	}

	projects, err := c.getProjectCount()
	if err != nil {
		return Counter, err
	}
	Counter.Projects = projects

	nodes, err := c.getNodeCount()
	if err != nil {
		return Counter, err
	}
	Counter.Nodes = nodes

	tokens, err := c.getTokenCount()
	if err != nil {
		return Counter, err
	}
	Counter.Token = tokens

	users, err := c.getUserCount()
	if err != nil {
		return Counter, err
	}
	Counter.Users = users

	return Counter, nil
}

func (c *Config) getClusterCount() (int, error) {
	fmt.Println("Getting cluster count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	clusters, err := managementClient.Cluster.List(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	clusterCount := len(clusters.Data)
	return clusterCount, nil
}

func (c *Config) getProjectCount() (int, error) {
	fmt.Println("Getting project count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	projects, err := managementClient.Project.List(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	projectCount := len(projects.Data)
	return projectCount, nil
}

func (c *Config) getNodeCount() (int, error) {
	fmt.Println("Getting node count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	nodes, err := managementClient.Node.List(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	nodeCount := len(nodes.Data)
	return nodeCount, nil
}

func (c *Config) getTokenCount() (int, error) {
	fmt.Println("Getting token count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	tokens, err := managementClient.Token.List(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	tokenCount := len(tokens.Data)
	return tokenCount, nil
}

func (c *Config) getUserCount() (int, error) {
	fmt.Println("Getting user count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	users, err := managementClient.User.List(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	userCount := len(users.Data)
	return userCount, nil
}

func main() {
	fmt.Println("Starting Rancher Prometheus Exporter")
	// Create a new HTTP server
	server := &http.Server{
		Addr: ":8080",
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		config := &Config{}
		dataCount, err := config.getData()
		if err != nil {
			http.Error(w, "Failed to get the data count", http.StatusInternalServerError)
			return
		}
		// Write the metrics to the response
		fmt.Fprintf(w, "# HELP rancher_cluster_count Current count of cluster resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_cluster_count gauge\n")
		fmt.Fprintf(w, "rancher_cluster_count %d\n", dataCount.Clusters)
		fmt.Fprintf(w, "# HELP rancher_project_count Current count of project resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_project_count gauge\n")
		fmt.Fprintf(w, "rancher_project_count %d\n", dataCount.Projects)
		fmt.Fprintf(w, "# HELP rancher_node_count Current count of node resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_node_count gauge\n")
		fmt.Fprintf(w, "rancher_node_count %d\n", dataCount.Nodes)
		fmt.Fprintf(w, "# HELP rancher_token_count Current count of token resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_token_count gauge\n")
		fmt.Fprintf(w, "rancher_token_count %d\n", dataCount.Token)
		fmt.Fprintf(w, "# HELP rancher_user_count Current count of user resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_user_count gauge\n")
		fmt.Fprintf(w, "rancher_user_count %d\n", dataCount.Users)
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
	server.Shutdown(context.TODO())
}

//func (c *Config) getProjectCount(clusterID string) (int, error) {
