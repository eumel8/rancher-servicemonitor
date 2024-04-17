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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/gookit/slog"

	"github.com/rancher/norman/clientbase"
	managementClient "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

const (
	port        = "8080"
	logTemplate = "[{{datetime}}] [{{level}}] {{caller}} {{message}} \n"
)

var registry = prometheus.NewRegistry()

var rancherClusterCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "rancher_cluster_count",
	Help: "Rancher Cluster count",
})

var rancherNodeCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "rancher_node_count",
	Help: "Rancher Node count",
})

var rancherProjectCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "rancher_project_count",
	Help: "Rancher Project count",
})

var rancherTokenCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "rancher_token_count",
	Help: "Rancher Token count",
})

var rancherUserCount = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "rancher_user_count",
	Help: "Rancher User count",
})

var rancherClusterCpuCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "rancher_cluster_cpu_count",
	Help: "Rancher Cluster CPU count",
}, []string{"cluster", "node", "type"})

var rancherClusterMemoryCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "rancher_cluster_memory_count",
	Help: "Rancher Cluster Memory count",
}, []string{"cluster", "node", "type"})

// Client are the client kind for a Rancher v3 API
type Client struct {
	Management *managementClient.Client
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
	Loglevel             string
	Port                 string
}

// Options for the client
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
	c.CACerts = os.Getenv("RANCHER_CACERT")
	c.ClusterID = os.Getenv("RANCHER_CLUSTERID")
	c.Loglevel = os.Getenv("LOG_LEVEL")
	c.Port = os.Getenv("PORT")
	c.ProjectID = os.Getenv("RANCHER_PROJECTID")
	c.TokenKey = os.Getenv("RANCHER_TOKEN")
	c.URL = os.Getenv("RANCHER_URL")
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

// getData gets the data from the Rancher API for the Counter struct
func (c *Config) getData() error {
	log.Debug("Getting data")

	managementClient, err := c.ManagementClient()
	if err != nil {
		return err
	}

	err = c.getClusterCount(managementClient)
	if err != nil {
		return err
	}

	err = c.getProjectCount(managementClient)
	if err != nil {
		return err
	}

	err = c.getNodeCount(managementClient)
	if err != nil {
		return err
	}

	err = c.getTokenCount(managementClient)
	if err != nil {
		return err
	}

	err = c.getUserCount(managementClient)
	if err != nil {
		return err
	}

	err = c.getNodeMetrics(managementClient)
	if err != nil {
		return err
	}

	return nil
}

// getClusterCount gets the count of clusters in Rancher
func (c *Config) getClusterCount(managementClient *managementClient.Client) error {
	log.Debug("Getting cluster count")

	clusters, err := managementClient.Cluster.ListAll(clientbase.NewListOpts())
	if err != nil {
		return err
	}
	clusterCount := len(clusters.Data)
	rancherClusterCount.Add(
		float64(clusterCount),
	)
	return nil
}

// getProjectCount gets the count of projects in Rancher
func (c *Config) getProjectCount(managementClient *managementClient.Client) error {
	log.Debug("Getting project count")

	projects, err := managementClient.Project.ListAll(clientbase.NewListOpts())
	if err != nil {
		return err
	}
	projectCount := len(projects.Data)
	rancherProjectCount.Add(
		float64(projectCount),
	)
	return nil
}

// getNodeCount gets the count of nodes in Rancher
func (c *Config) getNodeCount(managementClient *managementClient.Client) error {
	log.Debug("Getting node count")

	nodes, err := managementClient.Node.ListAll(clientbase.NewListOpts())
	if err != nil {
		return err
	}
	nodeCount := len(nodes.Data)
	rancherNodeCount.Add(
		float64(nodeCount),
	)
	return nil
}

// getNodeCPUCount gets the count of cpu in Rancher
func (c *Config) getNodeMetrics(managementClient *managementClient.Client) error {
	log.Debug("Getting node cpu")

	nodes, err := managementClient.Node.ListAll(clientbase.NewListOpts())
	if err != nil {
		return err
	}

	var nodeType string

	// loop around all nodes data
	for _, node := range nodes.Data {
		if node.Worker {
			nodeType = "worker"
		} else {
			nodeType = "master"
		}

		rancherClusterCpuCount.
			WithLabelValues(node.ClusterID, node.Name, nodeType).
			Set(float64(node.Info.CPU.Count))

		fmt.Println("Node Name: ", node.Name, "Node Cluster ID: ", node.ClusterID, "Node Type: ", nodeType, "Node CPU Count: ", node.Info.CPU.Count)
		rancherClusterMemoryCount.
			WithLabelValues(node.ClusterID, node.Name, nodeType).
			Set(float64(node.Info.Memory.MemTotalKiB))

	}
	return nil
}

// getTokenCount gets the count of tokens in Rancher
func (c *Config) getTokenCount(managementClient *managementClient.Client) error {
	log.Debug("Getting token count")

	tokens, err := managementClient.Token.ListAll(clientbase.NewListOpts())
	if err != nil {
		return err
	}
	tokenCount := len(tokens.Data)
	rancherTokenCount.Add(
		float64(tokenCount),
	)
	return nil
}

// get the count of users in Rancher
func (c *Config) getUserCount(managementClient *managementClient.Client) error {
	log.Debug("Getting user count")

	users, err := managementClient.User.ListAll(clientbase.NewListOpts())
	if err != nil {
		return err
	}
	userCount := len(users.Data)
	rancherUserCount.Add(
		float64(userCount),
	)
	return nil
}

func main() {
	config := &Config{}
	config.Load()

	logLevel := &config.Loglevel
	switch *logLevel {
	case "fatal":
		log.SetLogLevel(log.FatalLevel)
	case "trace":
		log.SetLogLevel(log.TraceLevel)
	case "debug":
		log.SetLogLevel(log.DebugLevel)
	case "error":
		log.SetLogLevel(log.ErrorLevel)
	case "warn":
		log.SetLogLevel(log.WarnLevel)
	case "info":
		log.SetLogLevel(log.InfoLevel)
	default:
		log.SetLogLevel(log.InfoLevel)
	}

	log.GetFormatter().(*log.TextFormatter).SetTemplate(logTemplate)

	log.Info("Starting Rancher Prometheus Exporter")

	// Create a new Prometheus registry
	registry.MustRegister(rancherClusterCount, rancherClusterCpuCount, rancherClusterMemoryCount, rancherNodeCount, rancherProjectCount, rancherTokenCount, rancherUserCount)

	// Create a new HTTP server
	port := config.Port
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr: ":" + port,
	}

	// call data
	err := config.getData()
	if err != nil {
		log.Error("Failed to get the data count", err)
		//http.Error(w, "Failed to get the data count", http.StatusInternalServerError)
		return
	}

	// Define the routes
	// Default route for probes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Rancher Prometheus Exporter")
	})

	// Metrics route
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	//log.Debug("Received request:", http.

	//log.Debug("Received request:", r.RemoteAddr, r.Method, r.RequestURI)

	// Start the server in a separate goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Error("Failed to start the server", err)
		}
	}()

	// Wait for termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Shutdown the server gracefully
	server.Shutdown(context.TODO())
}
