# Kubernetes Guestbook Operator

This is a Kubernetes operator for managing `GuestbookEntry` custom resources. For each `GuestbookEntry`, the operator creates and manages a corresponding `ConfigMap` containing the guestbook entry's name and message.

## Overview

The Guestbook Operator demonstrates a basic Kubernetes operator pattern built with [Kubebuilder](https://book.kubebuilder.io/). It defines a Custom Resource Definition (CRD) called `GuestbookEntry` and a controller that reconciles these resources.

When a `GuestbookEntry` CR is created, the controller will:
1. Create a `ConfigMap` named `<guestbookentry-name>-entry`.
2. Populate the `ConfigMap` with the `name` and `message` from the `GuestbookEntry`'s spec.
3. Set an owner reference on the `ConfigMap` so that it is garbage collected when the `GuestbookEntry` is deleted.
4. Update the `GuestbookEntry`'s status to reflect the state of the `ConfigMap`.

## Prerequisites

*   [Go](https://golang.org/dl/) (version as specified in `go.mod`)
*   [Docker](https://docs.docker.com/get-docker/) (if building and pushing container images)
*   [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
*   Access to a Kubernetes cluster (e.g., [Kind](https://kind.sigs.k8s.io/), [Minikube](https://minikube.sigs.k8s.io/docs/start/), or a cloud provider's Kubernetes service)
*   [Kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) (usually bundled with `kubectl` or installed via `make`)
*   `make`

## Getting Started

### 1. Clone the Repository

```bash
# Replace with your repository URL if applicable
git clone <your-repository-url>
cd kubernetes-guestbook-operator
```

### 2. Install CRDs

This command installs the `GuestbookEntry` Custom Resource Definition into your cluster.

```bash
make install
```

### 3. Run the Operator

You can run the operator in two ways:

**Option A: Run Locally (for development)**

This method runs the operator on your local machine, using your local kubeconfig to communicate with the cluster.

```bash
make run
```
Keep this terminal window open. You will see logs from the operator here.

**Option B: Build and Deploy to Cluster**

This method builds a Docker image, pushes it to a container registry, and deploys the operator as a Kubernetes Deployment in your cluster.

```bash
# Set IMG to your container registry and image name
export IMG=<your-registry>/guestbook-operator:v0.0.1

# Build and push the Docker image
make docker-build docker-push IMG=$IMG

# Deploy the operator to the cluster
make deploy IMG=$IMG
```

### 4. Create a GuestbookEntry Custom Resource

Create a YAML file for your `GuestbookEntry`. For example, `config/samples/example_v1alpha1_guestbookentry.yaml`:

```yaml
apiVersion: example.connell.com/v1alpha1
kind: GuestbookEntry
metadata:
  name: my-first-entry
  namespace: default # Ensure this namespace exists or change as needed
spec:
  name: "Alice"
  message: "Hello from the Guestbook!"
```

Apply this CR to your cluster:

```bash
kubectl apply -f config/samples/example_v1alpha1_guestbookentry.yaml
```

### 5. Observe the Operator in Action

*   **Check Operator Logs**:
    *   If running locally (`make run`), observe the terminal where the operator is running.
    *   If deployed to the cluster, check the logs of the operator pod:
        ```bash
        kubectl logs -f deployment/kubernetes-guestbook-operator-controller-manager -n kubernetes-guestbook-operator-system # Adjust namespace if different
        ```

*   **Check the GuestbookEntry status**:
    ```bash
    kubectl get guestbookentry my-first-entry -n default -o yaml
    ```
    You should see `status.phase` as "Processed" and `status.message` indicating the ConfigMap status.

*   **Check the ConfigMap**:
    ```bash
    kubectl get configmap my-first-entry-entry -n default -o yaml
    ```
    You should see the ConfigMap created with the data from your `GuestbookEntry` CR and an `ownerReference` pointing to it.

## Cleanup

### 1. Delete the GuestbookEntry CR

```bash
kubectl delete -f config/samples/example_v1alpha1_guestbookentry.yaml
```
The associated `ConfigMap` will be automatically garbage collected due to the owner reference.

### 2. Undeploy the Operator (if deployed to cluster)

If you deployed the operator using `make deploy`:
```bash
make undeploy IMG=$IMG # Ensure IMG is set to the same value used for deployment
```

### 3. Uninstall CRDs

```bash
make uninstall
```

### 4. Stop Local Operator (if running locally)

If you are running the operator locally with `make run`, stop it by pressing `Ctrl+C` in its terminal.

## Development

To regenerate code, CRDs, and RBAC manifests after making changes to API definitions or controller logic:

```bash
make generate
make manifests
```
