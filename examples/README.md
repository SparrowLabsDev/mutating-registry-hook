# Examples

This directory contains example manifests demonstrating how to use the mutating registry hook operator.

## Example Namespaces

### Namespace with Registry Rewriting Enabled

```yaml
# examples/namespace-with-rewrite.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: team-a
  labels:
    registry-rewrite: "enabled"  # Enable registry rewriting
  annotations:
    image-rewriter.example.com/target-registry: "team-a-registry.example.com"  # Target registry
```

Apply this namespace:
```bash
kubectl apply -f examples/namespace-with-rewrite.yaml
```

### Namespace without Registry Rewriting

```yaml
# examples/namespace-without-rewrite.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: team-b
  # No registry-rewrite label - pods will not be modified
```

Apply this namespace:
```bash
kubectl apply -f examples/namespace-without-rewrite.yaml
```

## Testing the Webhook

### 1. Deploy a test pod in the enabled namespace

```bash
kubectl apply -f examples/test-pod.yaml
```

### 2. Verify the images were rewritten

```bash
kubectl get pod test-pod -n team-a -o jsonpath='{.spec.containers[*].image}'
```

**Expected output:**
```
team-a-registry.example.com/nginx:latest team-a-registry.example.com/busybox:1.36
```

**Original images in the manifest:**
- `nginx:latest`
- `docker.io/busybox:1.36`

### 3. Check init containers

```bash
kubectl get pod test-pod -n team-a -o jsonpath='{.spec.initContainers[*].image}'
```

**Expected output:**
```
team-a-registry.example.com/google-samples/hello-app:1.0
```

**Original image:**
- `gcr.io/google-samples/hello-app:1.0`

### 4. Deploy the same pod in a namespace without rewriting

```bash
# Create the pod in team-b namespace
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: team-b
spec:
  containers:
  - name: nginx
    image: nginx:latest
EOF
```

Verify the image was NOT rewritten:
```bash
kubectl get pod test-pod -n team-b -o jsonpath='{.spec.containers[0].image}'
```

**Expected output:**
```
nginx:latest
```

## Image Rewriting Examples

The webhook rewrites images based on these rules:

| Original Image | Target Registry | Result |
|---|---|---|
| `nginx:latest` | `my-registry.com` | `my-registry.com/nginx:latest` |
| `docker.io/nginx:latest` | `my-registry.com` | `my-registry.com/nginx:latest` |
| `gcr.io/project/app:v1` | `my-registry.com` | `my-registry.com/project/app:v1` |
| `nginx@sha256:abc123...` | `my-registry.com` | `my-registry.com/nginx@sha256:abc123...` |
| `localhost:5000/image:v1` | `my-registry.com` | `my-registry.com/image:v1` |

## Cleanup

```bash
kubectl delete -f examples/
```
