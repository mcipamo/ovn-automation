package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// runCommand executes a shell command and returns its output.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// removeOVNDatabase removes the OVN database files on a given node.
func removeOVNDatabase(node string) error {
	fmt.Printf("Removing OVN DB on node: %s\n", node)
	output, err := runCommand("oc", "debug", "node/"+node,
		"--as", "backplane-cluster-admin",
		"--", "chroot", "/host", "/bin/bash", "-c",
		"rm -f /var/lib/ovn-ic/etc/ovn*.db",
	)
	if err != nil {
		log.Printf("Failed to remove OVN DB on node %s: %v\nOutput: %s", node, err, output)
		return err
	}
	fmt.Printf("\033[032mSuccessfully removed OVN DB on node: %s\033[0m\n", node)
	return nil
}

// restartOVS restarts Open vSwitch services on a given node.
func restartOVS(node string) error {
	fmt.Printf("Restarting Open vSwitch services on node: %s\n", node)
	output, err := runCommand("oc", "debug", "node/"+node,
		"--as", "backplane-cluster-admin",
		"--", "chroot", "/host", "/bin/bash", "-c",
		"systemctl restart ovs-vswitchd ovsdb-server",
	)
	if err != nil {
		log.Printf("Failed to restart services on node %s: %v\nOutput: %s", node, err, output)
		return err
	}
	fmt.Printf("\033[032mSuccessfully restarted Open vSwitch services on node: %s\033[0m\n", node)
	return nil
}

// deleteOVNKubePod deletes the ovnkube-node pod running on a given node.
func deleteOVNKubePod(node string) error {
	fmt.Printf("Deleting ovnkube-node pod on node: %s\n", node)
	output, err := runCommand("oc", "-n", "openshift-ovn-kubernetes",
		"--as", "backplane-cluster-admin", "delete", "pod",
		"-l", "app=ovnkube-node", "--field-selector=spec.nodeName="+node)
	if err != nil {
		log.Printf("Failed to delete pod on node %s: %v\nOutput: %s", node, err, output)
		return err
	}
	fmt.Printf("\033[032mSuccessfully deleted ovnkube-node pod on node: %s\033[0m\n", node)
	return nil
}

// watchPodRecreation watches for pod recreation across all nodes.
func watchPodRecreation(duration time.Duration) {
	fmt.Println("Watching for pod recreation across all nodes...")

	watchCmd := exec.Command("oc", "-n", "openshift-ovn-kubernetes",
		"--as", "backplane-cluster-admin", "get", "pod",
		"-l", "app=ovnkube-node", "-w")

	stdout, err := watchCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating stdout pipe: %v", err)
	}

	if err := watchCmd.Start(); err != nil {
		log.Fatalf("Error starting watch command: %v", err)
	}

	// Read pod recreation logs in real-time
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Error reading watch output: %v", err)
		}
	}()

	// Wait for the specified duration before stopping the watch process
	time.Sleep(duration)

	// Gracefully stop the watch command
	if err := watchCmd.Process.Kill(); err != nil {
		log.Printf("Error stopping watch command: %v", err)
	} else {
		fmt.Println("Stopped watching after", duration)
	}
}

func main() {
	// Step 1: Get the list of nodes running ovnkube-node
	output, err := runCommand("oc", "get", "pod", "-n", "openshift-ovn-kubernetes",
		"-l=app=ovnkube-node", "-o", "custom-columns=NODE_NAME:.spec.nodeName", "--no-headers")
	if err != nil {
		log.Fatalf("Error retrieving nodes: %v\nOutput: %s", err, output)
	}

	nodes := strings.Split(strings.TrimSpace(output), "\n")
	if len(nodes) == 0 || (len(nodes) == 1 && nodes[0] == "") {
		log.Println("No nodes found with ovnkube-node.")
		return
	}

	// Step 2: Perform tasks for each node
	for _, node := range nodes {
		if err := removeOVNDatabase(node); err != nil {
			continue
		}
		if err := restartOVS(node); err != nil {
			continue
		}
		if err := deleteOVNKubePod(node); err != nil {
			continue
		}
	}

	// Step 3: Watch for pod recreation for 120 seconds
	watchPodRecreation(120 * time.Second)
}
