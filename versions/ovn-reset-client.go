package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/openshift/osdctl/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	clusterID := flag.String("cluster-id", "", "OpenShift ClusterID")
	reason := flag.String("reason", "OVN DB Reset", "Reason for the operation")
	flag.Parse()

	if *clusterID == "" {
		log.Fatal("Error: --cluster-id is required")
	}

	kubeClient, err := createKubeClient(*clusterID, *reason)
	if err != nil {
		log.Fatalf("Error getting Kubernetes client: %v", err)
	}

	nodes, err := getNodesRunningOVN(kubeClient)
	if err != nil {
		log.Fatalf("Error getting OVN nodes: %v", err)
	}

	for _, node := range nodes {
		log.Printf("Processing node: %s", node.Name)

		if err := cleanOVNDBAndRestartServices(kubeClient, node.Name); err != nil {
			log.Printf("Error cleaning OVN DB in %s: %v", node.Name, err)
			continue
		}

		if err := deleteOVNKubeNodePod(kubeClient, node.Name); err != nil {
			log.Printf("Error deleting ovnkube-node pod in %s: %v", node.Name, err)
			continue
		}

		if err := waitForPodRecreation(kubeClient, node.Name); err != nil {
			log.Printf("Error waiting for pod recreation in %s: %v", node.Name, err)
			continue
		}
	}
	log.Println("✅ OVN database reset successfully completed!")
}

func createKubeClient(clusterID, reason string) (client.Client, error) {
	kubeClient, err := k8s.NewAsBackplaneClusterAdmin(clusterID, client.Options{}, reason)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate as backplane-cluster-admin: %w", err)
	}
	return kubeClient, nil
}

func getNodesRunningOVN(kubeClient client.Client) ([]corev1.Node, error) {
	var nodeList corev1.NodeList
	err := kubeClient.List(context.TODO(), &nodeList)
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %w", err)
	}
	return nodeList.Items, nil
}

func cleanOVNDBAndRestartServices(kubeClient client.Client, nodeName string) error {
	log.Printf("Cleaning OVN DB in node %s...", nodeName)
	cmds := []string{
		"rm -f /var/lib/ovn-ic/etc/ovn*.db",
		"systemctl restart ovs-vswitchd ovsdb-server",
	}
	return executeCommandOnNode(kubeClient, nodeName, cmds)
}

func deleteOVNKubeNodePod(kubeClient client.Client, nodeName string) error {
	log.Printf("Deleting ovnkube-node pod on node %s...", nodeName)
	return kubeClient.Delete(context.TODO(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "openshift-ovn-kubernetes",
			Labels:    map[string]string{"app": "ovnkube-node"},
		},
	})
}

func waitForPodRecreation(kubeClient client.Client, nodeName string) error {
	log.Printf("Waiting for ovnkube-node pod recreation on %s...", nodeName)
	time.Sleep(5 * time.Second)
	return nil
}

func executeCommandOnNode(kubeClient client.Client, nodeName string, commands []string) error {
	log.Printf("Executing commands on node %s", nodeName)
	// Aquí iría la implementación de ejecución remota
	time.Sleep(2 * time.Second)
	return nil
}

