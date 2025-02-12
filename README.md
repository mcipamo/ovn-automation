# OVN Automation Script

This Go program automates the process of rebuilding OVN databases and restarting OVN-related services across multiple nodes in an OpenShift cluster. It performs the following tasks for each node running `ovnkube-node`:

1. **Removes the OVN database files**
2. **Restarts Open vSwitch services**
3. **Deletes the `ovnkube-node` pod**
4. **Watches for pod recreation**

If you need to rebuild the OVN deployments of more than one node, the script will process all nodes automatically.

## Prerequisites

Ensure you have the following installed and configured:

- OpenShift CLI (`oc`)
- Go (version 1.22 or newer recommended)
- Sufficient cluster permissions to execute administrative commands

## Installation

1. Clone this repository:
   ```sh
   git clone https://github.com/mcipamo/ovn-automation.git
   cd ovn-automation
   ```
2. Build the Go binary:
   ```sh
   go build -o ovn-rebuild
   ```

## Usage

Run the script to automate OVN database rebuilding across all relevant nodes:

```sh
./ovn-rebuild
```

This will:
- Identify nodes running `ovnkube-node`
- Execute all necessary cleanup and restart operations
- Monitor pod recreation for 120 seconds

## Troubleshooting

If you encounter errors, ensure:
- Your user has `backplane-cluster-admin` privileges.
- OpenShift CLI (`oc`) is authenticated with the correct cluster.

## Contributions

Contributions are welcome! Feel free to submit issues or pull requests to improve this automation.
