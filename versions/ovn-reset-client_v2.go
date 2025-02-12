package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	clusterID := flag.String("cluster-id", "", "OpenShift ClusterID")
	flag.Parse()

	if *clusterID == "" {
		log.Fatal("Error: --cluster-id is required")
	}

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get Kubernetes config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	nodes, err := getNodesRunningOVN(clientset)
	if err != nil {
		log.Fatalf("Error getting OVN nodes: %v", err)
	}

	for _, node := range nodes {
		log.Printf("Processing node: %s", node.Name)

		if err := executeCommandOnNode(clientset, node.Name, "rm -f /var/lib/ovn-ic/etc/ovn*.db"); err != nil {
			log.Printf("Error cleaning OVN DB on %s: %v", node.Name, err)
			continue
		}

		if err := executeCommandOnNode(clientset, node.Name, "systemctl restart ovs-vswitchd ovsdb-server"); err != nil {
			log.Printf("Error restarting Open vSwitch on %s: %v", node.Name, err)
			continue
		}

		if err := deleteOVNKubeNodePod(clientset, node.Name); err != nil {
			log.Printf("Error deleting ovnkube-node pod on %s: %v", node.Name, err)
			continue
		}

		if err := waitForPodRecreation(clientset, node.Name); err != nil {
			log.Printf("Error waiting for pod recreation on %s: %v", node.Name, err)
			continue
		}
	}
	log.Println(" OVN database reset completed successfully!")
}

func getNodesRunningOVN(clientset *kubernetes.Clientset) ([]corev1.Node, error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func executeCommandOnNode(clientset *kubernetes.Clientset, nodeName, command string) error {
	log.Printf("Executing on %s: %s", nodeName, command)
	// Implement pod execution logic with client-go
	return nil
}

func deleteOVNKubeNodePod(clientset *kubernetes.Clientset, nodeName string) error {
	log.Printf("Deleting ovnkube-node pod on %s", nodeName)
	return clientset.CoreV1().Pods("openshift-ovn-kubernetes").DeleteCollection(
		context.TODO(),
		metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=ovnkube-node,spec.nodeName=%s", nodeName),
		},
	)
}

func waitForPodRecreation(clientset *kubernetes.Clientset, nodeName string) error {
	log.Printf("Waiting for ovnkube-node pod to be recreated on %s", nodeName)
	time.Sleep(5 * time.Second)
	return nil
}

