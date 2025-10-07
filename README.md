# Mutating Registry Hook

A Kubernetes operator that automatically rewrites container image registries for pods in specific namespaces.

## Overview

This operator implements a mutating admission webhook that intercepts pod creation requests and rewrites the container image registry based on namespace configuration. This is useful for:

- Redirecting pulls to private/mirror registries
- Implementing per-team registry policies
- Testing image sources without modifying manifests

## How It Works

Namespaces opt-in to registry rewriting using labels and annotations:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: team-a
  labels:
    registry-rewrite: "enabled"
  annotations:
    image-rewriter.example.com/target-registry: "team-a-registry.example.com"
```

When a pod is created in the `team-a` namespace:

- `nginx:latest` → `team-a-registry.example.com/nginx:latest`
- `docker.io/nginx:latest` → `team-a-registry.example.com/nginx:latest`
- `gcr.io/project/app:v1` → `team-a-registry.example.com/project/app:v1`

## Features

- **Namespace-scoped**: Only affects namespaces with `registry-rewrite: "enabled"` label
- **Preserves image paths**: Maintains repository paths, tags, and digests
- **Non-intrusive**: Fails open (never blocks pod creation on errors)
- **Supports all container types**: Init containers, regular containers, and ephemeral containers

## Documentation

- [Software Requirements Specification](SRS.md) - Detailed requirements and acceptance criteria
- [Architecture Decision Records](docs/adr/) - Design decisions and rationale

## Prerequisites

- Kubernetes cluster (v1.21+)
- [cert-manager](https://cert-manager.io/) installed in the cluster
- kubectl configured to access your cluster
- Go 1.21+ (for local development)

## Installation

### 1. Install cert-manager

The webhook requires cert-manager for TLS certificate management:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

Wait for cert-manager to be ready:
```bash
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-webhook -n cert-manager
```

### 2. Deploy the operator

```bash
# Deploy the operator to your cluster
make deploy
```

This will:
- Create the `mutating-registry-hook-system` namespace
- Deploy the webhook server
- Configure the mutating webhook
- Set up RBAC permissions
- Create cert-manager certificates

### 3. Verify the deployment

```bash
# Check the operator is running
kubectl get pods -n mutating-registry-hook-system

# Check the webhook configuration
kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io mutating-registry-hook-mutating-webhook-configuration
```

## Usage

### Quick Start

1. **Create a namespace with registry rewriting enabled:**

```bash
kubectl apply -f examples/namespace-with-rewrite.yaml
```

2. **Deploy a test pod:**

```bash
kubectl apply -f examples/test-pod.yaml
```

3. **Verify the images were rewritten:**

```bash
kubectl get pod test-pod -n team-a -o jsonpath='{.spec.containers[*].image}'
```

Expected output:
```
team-a-registry.example.com/nginx:latest team-a-registry.example.com/busybox:1.36
```

### Configuration

To enable registry rewriting for a namespace, add these labels and annotations:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: your-namespace
  labels:
    registry-rewrite: "enabled"  # Required: Enable the webhook
  annotations:
    image-rewriter.example.com/target-registry: "your-registry.example.com"  # Required: Target registry
```

The webhook will rewrite **all** container images in pods created in this namespace:
- Regular containers (`spec.containers`)
- Init containers (`spec.initContainers`)
- Ephemeral containers (`spec.ephemeralContainers`)

### Examples

See the [examples/](examples/) directory for more detailed examples and test scenarios.

## Development

### Build and run locally

```bash
# Install dependencies
go mod download

# Run tests
make test

# Build the operator
make build

# Run locally (requires KUBECONFIG)
make run
```

### Running tests

```bash
# Unit tests
go test ./internal/...

# With coverage
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Uninstall

```bash
# Remove the operator from the cluster
make undeploy
```

## Troubleshooting

### Webhook not rewriting images

1. Check namespace has the correct label:
   ```bash
   kubectl get namespace <your-namespace> -o jsonpath='{.metadata.labels.registry-rewrite}'
   ```
   Should output: `enabled`

2. Check namespace has the target registry annotation:
   ```bash
   kubectl get namespace <your-namespace> -o jsonpath='{.metadata.annotations.image-rewriter\.example\.com/target-registry}'
   ```

3. Check webhook logs:
   ```bash
   kubectl logs -n mutating-registry-hook-system deployment/mutating-registry-hook-controller-manager
   ```

### Pods failing to start

The webhook uses `failurePolicy: Ignore`, so it should never block pod creation. If pods are failing:

1. Check if it's a registry authentication issue (not related to the webhook)
2. Verify the target registry is accessible from your cluster
3. Check pod events: `kubectl describe pod <pod-name> -n <namespace>`

### Certificate issues

If the webhook is not working due to certificate problems:

```bash
# Check cert-manager certificate
kubectl get certificate -n mutating-registry-hook-system

# Check certificate secret
kubectl get secret webhook-server-cert -n mutating-registry-hook-system

# Restart cert-manager if needed
kubectl rollout restart deployment cert-manager -n cert-manager
```

## Documentation

- [Software Requirements Specification](SRS.md) - Detailed requirements and acceptance criteria
- [Architecture Decision Records](docs/adr/) - Design decisions and rationale
- [Examples](examples/) - Usage examples and test scenarios

## Contributing

This project follows TDD (Test-Driven Development) practices. When contributing:

1. Write tests first
2. Implement the minimal code to pass tests
3. Refactor while keeping tests green
4. Make small, focused commits

## License

Licensed under the Apache License, Version 2.0
