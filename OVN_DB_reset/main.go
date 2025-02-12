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

	commands := []string{
		"rm -f /host/var/lib/ovn-ic/etc/ovn*.db",
		"chroot /host /bin/bash -c 'systemctl restart ovs-vswitchd ovsdb-server'",
	}

	for _, node := range nodes {
		log.Printf("Processing node: %s", node.Name)

		if err := executeCommandOnNode(kubeClient, node.Name, commands); err != nil {
			log.Printf("Error executing command on node: %v", err)
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
	log.Println("OVN database reset successfully completed!")
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

func deleteOVNKubeNodePod(kubeClient client.Client, nodeName string) error {
	log.Printf("Deleting ovnkube-node pod on node %s...", nodeName)

	var podList corev1.PodList
	err := kubeClient.List(context.TODO(), &podList, client.InNamespace("openshift-ovn-kubernetes"), client.MatchingLabels{"app": "ovnkube-node"})
	if err != nil {
		return fmt.Errorf("error listing ovnkube-node pods: %w", err)
	}

	for _, pod := range podList.Items {
		if pod.Spec.NodeName == nodeName {
			if err := kubeClient.Delete(context.TODO(), &pod); err != nil {
				return fmt.Errorf("failed to delete pod %s on node %s: %w", pod.Name, nodeName, err)
			}
			log.Printf("Deleted pod %s on node %s", pod.Name, nodeName)
		}
	}
	return nil
}

func waitForPodRecreation(kubeClient client.Client, nodeName string) error {
	log.Printf("Waiting for ovnkube-node pod recreation on %s...", nodeName)
	time.Sleep(5 * time.Second)
	return nil
}

func executeCommandOnNode(kubeClient client.Client, nodeName string, commands []string) error {
	log.Printf("Executing commands on node %s", nodeName)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ovn-reset-",
			Namespace:    "default",
		},
		Spec: corev1.PodSpec{
			HostPID:       true,
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "ovn-reset",
					Image: "registry.redhat.io/ubi9/ubi:latest",
					SecurityContext: &corev1.SecurityContext{
						Privileged: func(b bool) *bool { return &b }(true),
					},
					Command: []string{"/bin/bash", "-c", commands[0] + " && " + commands[1]},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "host-root",
							MountPath: "/host",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "host-root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
						},
					},
				},
			},
		},
	}

	if err := kubeClient.Create(context.TODO(), pod); err != nil {
		return fmt.Errorf("failed to create pod on node %s: %w", nodeName, err)
	}

	log.Printf("Pod created on node %s, waiting for execution...", nodeName)

	time.Sleep(30 * time.Second)

	if err := kubeClient.Delete(context.TODO(), pod); err != nil {
		log.Printf("Error deleting pod on node %s: %v", nodeName, err)
	}

	log.Printf("Pod deleted on node %s", nodeName)
	return nil
}
