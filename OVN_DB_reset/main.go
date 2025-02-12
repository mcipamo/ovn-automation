package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	daemonsetName      = "ovn-db-reset"
	daemonsetNamespace = "openshift-ovn-kubernetes"
)

func main() {
	// Read command-line parameters
	kubeconfig := flag.String("kubeconfig", "", "Path to the kubeconfig file")
	flag.Parse()

	// Create Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Error creating Kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
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

	log.Println("✅ OVN database reset completed successfully!")
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
					HostPID:       true,
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
	log.Println("✅ DaemonSet deployed successfully!")
	return nil
}

func deleteDaemonSet(clientset *kubernetes.Clientset) error {
	return clientset.AppsV1().DaemonSets(daemonsetNamespace).Delete(context.TODO(), daemonsetName, metav1.DeleteOptions{})
}
