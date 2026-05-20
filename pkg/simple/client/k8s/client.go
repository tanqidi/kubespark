package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Interface defines the interface for Kubernetes client operations
type Interface interface {
	Kubernetes() kubernetes.Interface
	Dynamic() dynamic.Interface
	RESTConfig() *rest.Config
}

// Client wraps Kubernetes clients
type Client struct {
	kubernetes kubernetes.Interface
	dynamic    dynamic.Interface
	config     *rest.Config
}

// Kubernetes returns the standard Kubernetes clientset
func (c *Client) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

// Dynamic returns the dynamic client for handling all resources
func (c *Client) Dynamic() dynamic.Interface {
	return c.dynamic
}

// RESTConfig returns Kubernetes REST config for advanced subresources.
func (c *Client) RESTConfig() *rest.Config {
	return c.config
}

// NewClient creates a new Kubernetes client
// It will try to use in-cluster config first, then fallback to kubeconfig
func NewClient() (Interface, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err == nil {
		fmt.Println("[kubeconfig] using in-cluster config")
	} else {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		if _, statErr := os.Stat(kubeconfig); statErr == nil {
			fmt.Printf("[kubeconfig] found at: %s\n", kubeconfig)
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				fmt.Printf("[kubeconfig] failed to load: %v\n", err)
				return nil, err
			}
		} else {
			fmt.Printf("[kubeconfig] not found at: %s\n", kubeconfig)
			return nil, statErr
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		kubernetes: clientset,
		dynamic:    dynamicClient,
		config:     config,
	}, nil
}
