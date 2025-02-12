package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/openshift/osdctl/pkg/k8s"
)

const (
	daemonsetName      = "ovn-db-reset"
	daemonsetNamespace = "openshift-ovn-kubernetes"
)

func main() {
	clusterID := flag.String("cluster-id", "", "OpenShift ClusterID")
	reason := flag.String("reason", "OVN DB Reset", "Reason for the operation")
	flag.Parse()

	if *clusterID == "" {
		log.Fatal("Error: --cluster-id is required")
	}

	clientset, err := createKubeClient(*clusterID, *reason)
	if err != nil {
		log.Fatalf("Error getting Kubernetes client: %v", err)
	}

	// Deploy DaemonSet to clean OVN
	if err := deployDaemonSet(clientset); err != nil {
		log.Fatalf("Error deploying DaemonSet: %v", err)
	}

	// Wait for the DaemonSet to complete execution
	log.Println("Waiting for DaemonSet to complete execution...")
	time.Sleep(30 * time.Second) // Adjust the time as needed

	// Delete the DaemonSet
	if err := deleteDaemonSet(clientset); err != nil {
		log.Fatalf("Error deleting DaemonSet: %v", err)
	}

	log.Println(" OVN database reset completed successfully!")
}

func createKubeClient(clusterID, reason string) (*kubernetes.Clientset, error) {
	config, err := k8s.GetKubeConfig(clusterID, reason)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate as backplane-cluster-admin: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientset, nil
}

func deployDaemonSet(clientset *kubernetes.Clientset) error {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonsetName,
			Namespace: daemonsetNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": daemonsetName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": daemonsetName},
				},
				Spec: corev1.PodSpec{
					HostPID:  true,
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "ovn-reset",
							Image: "registry.redhat.io/ubi9/ubi:latest",
							SecurityContext: &corev1.SecurityContext{
								Privileged: func(b bool) *bool { return &b }(true),
							},
							Command: []string{"/bin/bash", "-c", `
								rm -f /var/lib/ovn-ic/etc/ovn*.db &&
								systemctl restart ovs-vswitchd ovsdb-server
							`},
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().DaemonSets(daemonsetNamespace).Create(context.TODO(), daemonSet, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create DaemonSet: %w", err)
	}
	log.Println(" DaemonSet deployed successfully!")
	return nil
}

func deleteDaemonSet(clientset *kubernetes.Clientset) error {
	return clientset.AppsV1().DaemonSets(daemonsetNamespace).Delete(context.TODO(), daemonsetName, metav1.DeleteOptions{})
}

