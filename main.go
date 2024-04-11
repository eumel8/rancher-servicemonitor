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

	"github.com/davecgh/go-spew/spew"
)

const (
	port        = "8080"
	logTemplate = "[{{datetime}}] [{{level}}] {{caller}} {{message}} \n"
)

var registry = prometheus.NewRegistry()

var rancherClusterCpuCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "rancher_cluster_cpu_count",
	Help: "Rancher Cluster CPU count",
}, []string{"cluster", "cpu_count"})

type providerLabelCpu struct {
	Cluster  string
	CpuCount int64
}

// Client are the client kind for a Rancher v3 API
type Client struct {
	Management *managementClient.Client
}

type Counter struct {
	Clusters int
	Nodes    int
	Projects int
	Token    int
	Users    int
	Cpu      struct {
		Clusters string
		CpuCount int64
	}
	Memory struct {
		Clusters    string
		MemoryCount int
	}
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
func (c *Config) getData() (Counter, error) {
	log.Debug("Getting data")

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

	cpus, err := c.getNodeCPU()
	if err != nil {
		return Counter, err
	}

	Counter.Cpu = cpus

	// Counter.Cpu = append(Counter.Cpu, cpus)

	return Counter, nil
}

// getClusterCount gets the count of clusters in Rancher
func (c *Config) getClusterCount() (int, error) {
	log.Debug("Getting cluster count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	clusters, err := managementClient.Cluster.ListAll(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	clusterCount := len(clusters.Data)
	return clusterCount, nil
}

// getProjectCount gets the count of projects in Rancher
func (c *Config) getProjectCount() (int, error) {
	log.Debug("Getting project count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	projects, err := managementClient.Project.ListAll(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	projectCount := len(projects.Data)
	return projectCount, nil
}

// getNodeCount gets the count of nodes in Rancher
func (c *Config) getNodeCount() (int, error) {
	log.Debug("Getting node count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	nodes, err := managementClient.Node.ListAll(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	nodeCount := len(nodes.Data)
	return nodeCount, nil
}

// getNodeCPUCount gets the count of cpu in Rancher
func (c *Config) getNodeCPU() (struct {
	Clusters string
	CpuCount int64
}, error) {
	var Cpu struct {
		Clusters string
		CpuCount int64
	}

	log.Debug("Getting node cpu")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return Cpu, err
	}
	nodes, err := managementClient.Node.ListAll(clientbase.NewListOpts())
	if err != nil {
		return Cpu, err
	}

	//Cpu := struct {
	//	Clusters string
	//	CpuCount int
	//}
	//Cpu      struct {
	//	Clusters string
	//	CpuCount int
	//}
	// node cpu summary
	for _, node := range nodes.Data {

		rancherClusterCpuCount.WithLabelValues(
			node.ClusterID,
			fmt.Sprintf("%d", node.Info.CPU.Count))
		//node.Info.CPU.Count,
		//node.ClusterID, node.CPU.Count)
		Cpu.Clusters = node.ClusterID
		Cpu.CpuCount = node.Info.CPU.Count
		//Cpu = append(Cpu, node.ClusterID, node.Info.CPU.Count)
		//fmt.Println(node.ClusterID)
		//fmt.Println(node.Info.CPU.Count)
	}
	//nodeCount := nodes.Data[0].Status.NodeInfo.CPUInfo.NumCores

	//nodeCount := len(nodes.Data)
	spew.Dump(Cpu)
	return Cpu, nil
	//return nodeCount, nil
}

// getTokenCount gets the count of tokens in Rancher
func (c *Config) getTokenCount() (int, error) {
	log.Debug("Getting token count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	tokens, err := managementClient.Token.ListAll(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	tokenCount := len(tokens.Data)
	return tokenCount, nil
}

// get the count of users in Rancher
func (c *Config) getUserCount() (int, error) {
	log.Debug("Getting user count")
	managementClient, err := c.ManagementClient()
	if err != nil {
		return 0, err
	}
	users, err := managementClient.User.ListAll(clientbase.NewListOpts())
	if err != nil {
		return 0, err
	}
	userCount := len(users.Data)
	return userCount, nil
}

func main() {
	config := &Config{}
	config.Load()

	registry.MustRegister(rancherClusterCpuCount)

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

	// Create a new HTTP server
	port := config.Port
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr: ":" + port,
	}

	// Define the routes
	// Default route for probes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Rancher Prometheus Exporter")
	})

	http.Handle("/metrics2", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	// Default route for probes

	// Metrics route
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {

		log.Debug("Received request:", r.RemoteAddr, r.Method, r.RequestURI)
		dataCount, err := config.getData()
		if err != nil {
			log.Error("Failed to get the data count", err)
			http.Error(w, "Failed to get the data count", http.StatusInternalServerError)
			return
		}
		// Write the metrics to the response
		fmt.Fprintf(w, "# HELP rancher_cluster_count Current count of cluster resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_cluster_count gauge\n")
		fmt.Fprintf(w, "rancher_cluster_count %d\n", dataCount.Clusters)
		fmt.Fprintf(w, "# HELP rancher_cluster_cpu_count Current count of cluster Cpu resource in Rancher\n")
		fmt.Fprintf(w, "# rancher_cluster_cpu_count gauge\n")
		fmt.Fprintf(w, "rancher_cluster_cpu_count %v\n", dataCount.Cpu)
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
