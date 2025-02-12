package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/openshift/osdctl/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	// Parameters to be passed via command line
	clusterID := flag.String("cluster-id", "", "OpenShift ClusterID")
	reason := flag.String("reason", "OVN DB Reset", "Reason to perform the action")
	flag.Parse()

	// Validate entry
	if *clusterID == "" {
		log.Fatal("Error: --cluster-id is required")
	}

	if *reason == "" {
		log.Fatal("Error: reason is required")
	}

	// Create KubeClient as backplane-cluster-admin
	kubeClient, err := createKubeClient(*clusterID, *reason)
	if err != nil {
		log.Fatalf("Error getting Kubernetes client: %v", err)
	}

	// Get nodes running OVN
	nodes, err := getNodesRunningOVN(kubeClient)
	if err != nil {
		log.Fatalf("Error getting OVN nodes: %v", err)
	}

	// Iterate each node
	for _, node := range nodes {
		log.Printf("Processing node: %s", node.Name)

		if err := cleanOVNDBAndRestartServices(node); err != nil {
			log.Printf("Error cleaning OVN DB in %s: %v", node.Name, err)
			continue
		}

		if err := deleteOVNKubeNodePod(node); err != nil {
			log.Printf("Error deleting ovnkube-node pod in %s: %v", node.Name, err)
			continue
		}

		if err := waitForPodRecreation(node); err != nil {
			log.Printf("Waiting for pod recreation in %s: %v", node.Name, err)
			continue
		}
	}
	log.Println("OVN database reset successfully completed!")
}

// createKubeClient crea un cliente Kubernetes con privilegios de administrador
func createKubeClient(clusterID, reason string) (client.Client, error) {
	kubeClient, err := k8s.NewAsBackplaneClusterAdmin(clusterID, client.Options{}, reason)
	if err != nil {
		return nil, fmt.Errorf("backplane-cluster-admin authentication failed: %w", err)
	}
	return kubeClient, nil
}

// getNodesRunningOVN get nodes with ovnkube-node
func getNodesRunningOVN(kubeClient client.Client) ([]corev1.Node, error) {
	var nodeList corev1.NodeList
	err := kubeClient.List(context.TODO(), &nodeList)
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %w", err)
	}
	return nodeList.Items, nil
}

// cleanOVNDBAndRestartServices removes OVN DB and restart Open vSwitch
func cleanOVNDBAndRestartServices(node corev1.Node) error {
	log.Printf("Cleaning OVN DB in node %s...", node.Name)
	// Next implementation
	time.Sleep(2 * time.Second)
	return nil
}

// deleteOVNKubeNodePod removes ovnkube-node pod on specific node
func deleteOVNKubeNodePod(node corev1.Node) error {
	log.Printf("Removing ovnkube-node pod in node %s...", node.Name)
	// Next implementation
	time.Sleep(2 * time.Second)
	return nil
}

// waitForPodRecreation wait for ovnkube-node recreation
func waitForPodRecreation(node corev1.Node) error {
	log.Printf(" Waiting for pod recreatin in node %s...", node.Name)
	// Next implementation
	time.Sleep(5 * time.Second)
	return nil
}
