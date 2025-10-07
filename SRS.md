# Software Requirements Specification (SRS)
## Kubernetes Registry Rewrite Operator

**Version:** 1.0
**Date:** 2025-10-02
**Status:** Draft

---

## 1. Project Overview

### 1.1 Objectives
Build a Kubernetes operator that implements a mutating admission webhook controller to dynamically rewrite container image registry sources for pods deployed in opt-in namespaces. This enables teams to redirect container images to private, team-specific registries without modifying deployment manifests.

### 1.2 Success Criteria
- ✅ Operator successfully deploys to Kubernetes cluster
- ✅ Namespaces can opt-in via label `registry-rewrite: "enabled"`
- ✅ Target registry is configurable per namespace via annotation
- ✅ Pod container images are rewritten to target registry
- ✅ Non-enabled namespaces are unaffected
- ✅ Operator handles errors gracefully without blocking pod creation
- ✅ All components have >80% test coverage

### 1.3 Scope

**In Scope:**
- Mutating admission webhook for pod creation
- Namespace-level opt-in mechanism via labels
- Per-namespace registry configuration via annotations
- Image registry rewriting for all container types (init, regular, ephemeral)
- TLS certificate management for webhook server
- Kubernetes deployment manifests

**Out of Scope:**
- Registry authentication/credential management
- Image tag validation or security scanning
- Cross-namespace registry sharing policies
- Image pull optimization or caching
- Support for non-pod resources (Deployments, StatefulSets - these create pods)

### 1.4 Associated ADRs
- ADR-001: Webhook Implementation Architecture (to be created)
- ADR-002: Certificate Management Strategy (to be created)
- ADR-003: Error Handling and Fail-Safe Design (to be created)

---

## 2. Functional Requirements

### 2.1 Core Webhook Infrastructure

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **FR-001** | Basic webhook HTTP server | HTTPS server listens on port 8443, responds to /healthz with 200 OK | S | Pending |
| **FR-002** | TLS certificate generation | Self-signed certs generated on startup, valid for 365 days, stored in memory | S | Pending |
| **FR-003** | Webhook admission endpoint | POST /mutate endpoint accepts AdmissionReview, returns valid AdmissionResponse | M | Pending |
| **FR-004** | MutatingWebhookConfiguration | K8s resource created with correct failurePolicy, namespaceSelector, and CA bundle | M | Pending |

**FR-001: Basic Webhook HTTP Server**
```gherkin
Given the operator is started
When the webhook server initializes
Then it SHALL listen on port 8443 using HTTPS
And it SHALL respond to GET /healthz with HTTP 200 status
And it SHALL respond to GET /readyz with HTTP 200 status
```

**FR-002: TLS Certificate Generation**
```gherkin
Given the operator starts for the first time
When TLS initialization occurs
Then the system SHALL generate a self-signed CA certificate
And the system SHALL generate a server certificate signed by the CA
And the certificates SHALL be valid for 365 days
And the certificates SHALL use RSA 2048-bit keys
And the certificates SHALL be stored in memory for server use
```

**FR-003: Webhook Admission Endpoint**
```gherkin
Given the webhook server is running
When a POST request is sent to /mutate with valid AdmissionReview JSON
Then the system SHALL parse the AdmissionReview v1 object
And the system SHALL return an AdmissionReview response
And the response SHALL include a valid AdmissionResponse
And the response SHALL have matching UID from the request
```

**FR-004: MutatingWebhookConfiguration Resource**
```gherkin
Given the operator has valid TLS certificates
When the MutatingWebhookConfiguration is created
Then it SHALL target the /mutate endpoint
And it SHALL have failurePolicy set to "Ignore"
And it SHALL have namespaceSelector matching "registry-rewrite: enabled"
And it SHALL include the CA bundle from the generated certificates
And it SHALL target Pod resources on CREATE operations
```

---

### 2.2 Namespace Opt-In Mechanism

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **FR-005** | Namespace label detection | Webhook only processes namespaces with label `registry-rewrite: "enabled"` | S | Pending |
| **FR-006** | Annotation-based registry config | Target registry read from annotation `image-rewriter.example.com/target-registry` | S | Pending |
| **FR-007** | Missing annotation handling | Pods pass through unchanged if annotation is missing in enabled namespace | S | Pending |

**FR-005: Namespace Label Detection**
```gherkin
Given a namespace exists with label "registry-rewrite: enabled"
When a pod is created in that namespace
Then the webhook SHALL receive the admission request
And the webhook SHALL process the pod for image rewriting

Given a namespace exists without label "registry-rewrite: enabled"
When a pod is created in that namespace
Then the webhook SHALL NOT receive the admission request
And the pod SHALL be created with original image references
```

**FR-006: Annotation-Based Registry Configuration**
```gherkin
Given a namespace has label "registry-rewrite: enabled"
And the namespace has annotation "image-rewriter.example.com/target-registry: team-a-registry.example.com"
When a pod admission request is processed
Then the system SHALL extract "team-a-registry.example.com" as the target registry
And the system SHALL use this registry for all image rewrites
```

**FR-007: Missing Annotation Handling**
```gherkin
Given a namespace has label "registry-rewrite: enabled"
And the namespace has NO annotation "image-rewriter.example.com/target-registry"
When a pod admission request is processed
Then the webhook SHALL allow the pod without modifications
And the webhook SHALL log a warning about missing configuration
And the admission response SHALL set "allowed: true" with no patch
```

---

### 2.3 Image Registry Rewriting Logic

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **FR-008** | Simple image rewrite (no registry) | `nginx:latest` → `target-registry.com/nginx:latest` | S | Pending |
| **FR-009** | Image rewrite with existing registry | `docker.io/nginx:latest` → `target-registry.com/nginx:latest` | S | Pending |
| **FR-010** | Image rewrite with path | `docker.io/library/nginx:latest` → `target-registry.com/library/nginx:latest` | M | Pending |
| **FR-011** | Init container image rewrite | All initContainers images are rewritten using same logic | S | Pending |
| **FR-012** | Ephemeral container image rewrite | All ephemeralContainers images are rewritten using same logic | S | Pending |
| **FR-013** | JSON patch generation | Webhook generates RFC 6902 JSON Patch for pod modifications | M | Pending |

**FR-008: Simple Image Rewrite (No Registry)**
```gherkin
Given target registry is "team-a-registry.example.com"
When a container specifies image "nginx:latest"
Then the system SHALL rewrite the image to "team-a-registry.example.com/nginx:latest"

Given target registry is "team-a-registry.example.com"
When a container specifies image "nginx"
Then the system SHALL rewrite the image to "team-a-registry.example.com/nginx"
```

**FR-009: Image Rewrite with Existing Registry**
```gherkin
Given target registry is "team-a-registry.example.com"
When a container specifies image "docker.io/nginx:latest"
Then the system SHALL strip "docker.io"
And the system SHALL rewrite the image to "team-a-registry.example.com/nginx:latest"

Given target registry is "team-a-registry.example.com"
When a container specifies image "gcr.io/project/app:v1.0"
Then the system SHALL strip "gcr.io"
And the system SHALL rewrite the image to "team-a-registry.example.com/project/app:v1.0"
```

**FR-010: Image Rewrite with Path Preservation**
```gherkin
Given target registry is "team-a-registry.example.com"
When a container specifies image "docker.io/library/nginx:latest"
Then the system SHALL preserve the path "library/nginx:latest"
And the system SHALL rewrite the image to "team-a-registry.example.com/library/nginx:latest"

Given target registry is "team-a-registry.example.com"
When a container specifies image "quay.io/org/team/app:v2.0"
Then the system SHALL preserve the path "org/team/app:v2.0"
And the system SHALL rewrite the image to "team-a-registry.example.com/org/team/app:v2.0"
```

**FR-011: Init Container Image Rewrite**
```gherkin
Given a pod has initContainers defined
When the webhook processes the pod
Then the system SHALL apply image rewriting to all initContainers
And each init container image SHALL follow the same rewrite rules as regular containers
```

**FR-012: Ephemeral Container Image Rewrite**
```gherkin
Given a pod has ephemeralContainers defined
When the webhook processes the pod
Then the system SHALL apply image rewriting to all ephemeralContainers
And each ephemeral container image SHALL follow the same rewrite rules as regular containers
```

**FR-013: JSON Patch Generation**
```gherkin
Given image rewrites have been determined
When the admission response is constructed
Then the system SHALL generate a JSON Patch (RFC 6902) document
And the patch SHALL use "replace" operations for container image fields
And the patch SHALL target paths like "/spec/containers/0/image"
And the AdmissionResponse patchType SHALL be "JSONPatch"
And the patch SHALL be base64 encoded in the response
```

---

### 2.4 Kubernetes Deployment & Operations

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **FR-014** | Deployment manifest | Complete K8s Deployment YAML with resource limits, health checks | M | Pending |
| **FR-015** | Service manifest | ClusterIP service exposes webhook on port 443 → 8443 | S | Pending |
| **FR-016** | RBAC configuration | ServiceAccount, ClusterRole, ClusterRoleBinding for webhook operations | M | Pending |
| **FR-017** | Webhook registration | MutatingWebhookConfiguration installed with correct selectors | S | Pending |

**FR-014: Deployment Manifest**
```gherkin
Given a Kubernetes cluster is available
When the operator Deployment is applied
Then it SHALL create a single replica pod
And the pod SHALL have resource requests (100m CPU, 128Mi memory)
And the pod SHALL have resource limits (500m CPU, 256Mi memory)
And the pod SHALL have livenessProbe on /healthz
And the pod SHALL have readinessProbe on /readyz
And the container SHALL run as non-root user (UID 65532)
```

**FR-015: Service Manifest**
```gherkin
Given the operator Deployment is running
When the Service is applied
Then it SHALL be of type ClusterIP
And it SHALL expose port 443 targeting container port 8443
And it SHALL select pods with the operator label
And it SHALL be named "registry-rewrite-webhook"
```

**FR-016: RBAC Configuration**
```gherkin
Given the operator needs to function in the cluster
When RBAC resources are applied
Then a ServiceAccount SHALL be created in the operator namespace
And a ClusterRole SHALL grant "get" access to namespaces
And a ClusterRoleBinding SHALL bind the ClusterRole to the ServiceAccount
And the Deployment SHALL use the ServiceAccount
```

**FR-017: Webhook Registration**
```gherkin
Given the webhook service is running and healthy
When the MutatingWebhookConfiguration is applied
Then it SHALL register the webhook with the API server
And it SHALL configure the webhook to intercept Pod CREATE operations
And it SHALL set the failurePolicy to "Ignore" for safety
And it SHALL include the namespaceSelector for "registry-rewrite: enabled"
And it SHALL configure the clientConfig to point to the service
```

---

## 3. Non-Functional Requirements

### 3.1 Security

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **NFR-001** | TLS encryption | All webhook communication uses TLS 1.2+ | S | Pending |
| **NFR-002** | Least privilege RBAC | ServiceAccount has minimum required permissions | S | Pending |
| **NFR-003** | Non-root container | Webhook runs as non-root user (UID 65532) | S | Pending |
| **NFR-004** | No secret logging | Webhook does not log sensitive data (tokens, certs) | S | Pending |

**NFR-001: TLS Encryption**
```
When the webhook server accepts connections
Then it SHALL enforce TLS version 1.2 or higher
And it SHALL reject non-TLS connections
And it SHALL use the generated server certificate for authentication
```

**NFR-002: Least Privilege RBAC**
```
When the operator ServiceAccount permissions are reviewed
Then it SHALL have ONLY "get" permission on namespaces
And it SHALL NOT have cluster-admin or elevated privileges
And it SHALL NOT have write access to any resources
```

**NFR-003: Non-Root Container**
```
Given the operator container is running
When the process user is inspected
Then it SHALL run as UID 65532 (nonroot)
And it SHALL NOT run as UID 0 (root)
And the container securityContext SHALL set runAsNonRoot: true
```

**NFR-004: No Secret Logging**
```
When the webhook logs messages
Then it SHALL NOT log TLS private keys
And it SHALL NOT log certificate contents
And it SHALL NOT log admission request authentication tokens
And it SHALL redact sensitive fields in debug output
```

---

### 3.2 Performance & Reliability

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **NFR-005** | Admission latency | Webhook responds to admission requests within 100ms (p95) | M | Pending |
| **NFR-006** | Fail-safe design | Webhook failures do not block pod creation | S | Pending |
| **NFR-007** | Resource efficiency | Operator uses <256Mi memory, <500m CPU under normal load | S | Pending |
| **NFR-008** | Concurrent requests | Handles 50 concurrent admission requests without degradation | M | Pending |

**NFR-005: Admission Latency**
```
Given the webhook is processing admission requests
When measured over 100 requests
Then the 95th percentile response time SHALL be ≤100ms
And the median response time SHALL be ≤50ms
And no single request SHALL exceed 500ms
```

**NFR-006: Fail-Safe Design**
```
When the webhook server is unavailable
Then the MutatingWebhookConfiguration failurePolicy "Ignore" SHALL allow pods to be created
And pods SHALL be created with original image references
And the Kubernetes API server SHALL log the webhook failure

When the webhook returns an error response
Then the admission SHALL be allowed with no modifications
And the error SHALL be logged for operator troubleshooting
```

**NFR-007: Resource Efficiency**
```
Given the operator is running under normal load (10 pods/min)
When resource usage is measured over 1 hour
Then memory usage SHALL remain below 256Mi
And CPU usage SHALL remain below 500m
And the operator SHALL NOT have memory leaks
```

**NFR-008: Concurrent Requests**
```
Given 50 pods are created simultaneously in enabled namespaces
When the webhook processes all admission requests
Then all requests SHALL complete successfully
And the p95 latency SHALL remain ≤150ms
And no requests SHALL timeout or fail
```

---

### 3.3 Observability & Operations

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **NFR-009** | Structured logging | All logs use JSON format with consistent fields | S | Pending |
| **NFR-010** | Metrics exposure | Prometheus metrics exposed on /metrics endpoint | M | Pending |
| **NFR-011** | Health endpoints | /healthz and /readyz endpoints for K8s probes | S | Pending |
| **NFR-012** | Audit trail | All image rewrites logged with namespace, pod, original, and rewritten image | S | Pending |

**NFR-009: Structured Logging**
```
When the webhook logs messages
Then all logs SHALL use JSON format
And each log SHALL include: timestamp, level, message, component
And admission logs SHALL include: namespace, pod_name, operation
And error logs SHALL include: error_type, error_message, stack_trace
```

**NFR-010: Metrics Exposure**
```
Given the webhook server is running
When GET /metrics is called
Then it SHALL return Prometheus-format metrics
And it SHALL include: admission_requests_total (counter)
And it SHALL include: admission_request_duration_seconds (histogram)
And it SHALL include: image_rewrites_total (counter)
And it SHALL include: admission_errors_total (counter)
And metrics SHALL be labeled by: namespace, operation, result
```

**NFR-011: Health Endpoints**
```
Given the webhook server is running
When GET /healthz is called
Then it SHALL return 200 OK if the server is alive
And it SHALL return 503 Service Unavailable if unhealthy

When GET /readyz is called
Then it SHALL return 200 OK if ready to accept requests
And it SHALL verify TLS certificates are loaded
And it SHALL return 503 Service Unavailable if not ready
```

**NFR-012: Audit Trail**
```
When an image is rewritten
Then the system SHALL log at INFO level
And the log SHALL include: namespace, pod_name
And the log SHALL include: container_name, original_image, rewritten_image
And the log SHALL include: target_registry from annotation

When an admission request is denied or errors occur
Then the system SHALL log at ERROR level
And the log SHALL include: reason, namespace, pod_name
```

---

## 4. Error Handling & Edge Cases

### 4.1 Error Scenarios

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **ERR-001** | Malformed image reference | Invalid image strings are logged and pod is allowed unchanged | S | Pending |
| **ERR-002** | Invalid target registry format | Invalid registry annotation is logged, pod is allowed unchanged | S | Pending |
| **ERR-003** | Namespace annotation missing | Pod is allowed unchanged with warning log | S | Pending |
| **ERR-004** | JSON patch generation failure | Admission is allowed with error log, original pod is created | S | Pending |
| **ERR-005** | Admission review parse error | 400 Bad Request returned, API server retries or fails based on policy | S | Pending |

**ERR-001: Malformed Image Reference**
```
Given a pod contains an invalid image reference "::invalid::"
When the webhook processes the admission request
Then the system SHALL log an ERROR with the invalid image
And the system SHALL allow the pod unchanged
And the AdmissionResponse SHALL set allowed: true with no patch
```

**ERR-002: Invalid Target Registry Format**
```
Given namespace annotation "image-rewriter.example.com/target-registry: https://invalid:8080/path"
When the webhook reads the target registry
Then the system SHALL detect the invalid format (protocol, port)
And the system SHALL log a WARNING
And the system SHALL allow the pod unchanged
```

**ERR-003: Namespace Annotation Missing**
```
Given a namespace with label "registry-rewrite: enabled"
And NO annotation "image-rewriter.example.com/target-registry"
When the webhook processes a pod admission
Then the system SHALL log a WARNING about missing configuration
And the system SHALL allow the pod unchanged
And the log SHALL include the namespace name
```

**ERR-004: JSON Patch Generation Failure**
```
Given the image rewrite logic completes successfully
When JSON patch generation encounters an error
Then the system SHALL log an ERROR with the failure reason
And the system SHALL allow the admission without modifications
And the system SHALL NOT block pod creation
```

**ERR-005: Admission Review Parse Error**
```
Given the webhook receives malformed AdmissionReview JSON
When the request is parsed
Then the system SHALL return HTTP 400 Bad Request
And the response SHALL include a descriptive error message
And the system SHALL log the parse error with request details
```

---

### 4.2 Edge Cases

| REQ-ID | Description | Acceptance Criteria | Effort | Status |
|--------|-------------|---------------------|--------|--------|
| **EDGE-001** | Image with digest | `nginx@sha256:abc123` preserves digest in rewrite | S | Pending |
| **EDGE-002** | Image with port in registry | `registry.com:5000/image:tag` correctly strips registry+port | S | Pending |
| **EDGE-003** | Empty container list | Pods with no containers are allowed unchanged | S | Pending |
| **EDGE-004** | Multiple containers | All containers in a pod are rewritten consistently | S | Pending |
| **EDGE-005** | Localhost registry | `localhost:5000/image:tag` is rewritten to target registry | S | Pending |

**EDGE-001: Image with Digest**
```
Given target registry is "team-a-registry.example.com"
When a container specifies image "nginx@sha256:abc123def456"
Then the system SHALL preserve the digest
And the system SHALL rewrite to "team-a-registry.example.com/nginx@sha256:abc123def456"

Given target registry is "team-a-registry.example.com"
When a container specifies image "docker.io/nginx@sha256:abc123"
Then the system SHALL strip "docker.io"
And the system SHALL rewrite to "team-a-registry.example.com/nginx@sha256:abc123"
```

**EDGE-002: Image with Port in Registry**
```
Given target registry is "team-a-registry.example.com"
When a container specifies image "registry.example.com:5000/app:v1"
Then the system SHALL strip "registry.example.com:5000"
And the system SHALL rewrite to "team-a-registry.example.com/app:v1"
```

**EDGE-003: Empty Container List**
```
Given a pod has no containers defined (edge case)
When the webhook processes the admission request
Then the system SHALL allow the pod unchanged
And the system SHALL NOT generate a patch
And the system SHALL log a DEBUG message about empty containers
```

**EDGE-004: Multiple Containers**
```
Given a pod has 3 containers with different images
When the webhook processes the admission request
Then the system SHALL rewrite all 3 container images
And the system SHALL use the same target registry for all containers
And the JSON patch SHALL include replace operations for all 3 images
```

**EDGE-005: Localhost Registry**
```
Given target registry is "team-a-registry.example.com"
When a container specifies image "localhost:5000/test-image:dev"
Then the system SHALL strip "localhost:5000"
And the system SHALL rewrite to "team-a-registry.example.com/test-image:dev"
```

---

## 5. Testing Requirements

### 5.1 Unit Testing

| TEST-ID | Description | Coverage Target | Effort | Status |
|---------|-------------|-----------------|--------|--------|
| **UT-001** | Image parsing logic | 100% coverage of all image format variations | M | Pending |
| **UT-002** | Image rewrite logic | 100% coverage including edge cases | M | Pending |
| **UT-003** | JSON patch generation | 100% coverage of patch operations | M | Pending |
| **UT-004** | Configuration parsing | 100% coverage of annotation/label extraction | S | Pending |
| **UT-005** | Error handling paths | 100% coverage of all error scenarios | M | Pending |

### 5.2 Integration Testing

| TEST-ID | Description | Coverage Target | Effort | Status |
|---------|-------------|-----------------|--------|--------|
| **IT-001** | Webhook admission flow | End-to-end admission request processing | M | Pending |
| **IT-002** | TLS certificate validation | Certificate generation and server startup | M | Pending |
| **IT-003** | Kubernetes client operations | Namespace lookup and annotation reading | M | Pending |

### 5.3 End-to-End Testing

| TEST-ID | Description | Coverage Target | Effort | Status |
|---------|-------------|-----------------|--------|--------|
| **E2E-001** | Pod creation in enabled namespace | Full flow from pod create to rewritten image | L | Pending |
| **E2E-002** | Pod creation in disabled namespace | Verify no rewriting occurs | M | Pending |
| **E2E-003** | Webhook failure scenario | Verify fail-safe behavior | M | Pending |
| **E2E-004** | Multiple namespace isolation | Verify per-namespace registry configuration | M | Pending |

---

## 6. Deployment Phases

### Phase 1: Core Infrastructure (Week 1)
- **FR-001 to FR-004**: Webhook server and TLS setup
- **NFR-001, NFR-003**: Security foundations
- **UT-001 to UT-003**: Core unit tests

### Phase 2: Image Rewriting Logic (Week 1-2)
- **FR-008 to FR-013**: Image parsing and rewriting
- **EDGE-001 to EDGE-005**: Edge case handling
- **UT-002**: Image rewrite unit tests

### Phase 3: Namespace Integration (Week 2)
- **FR-005 to FR-007**: Namespace opt-in mechanism
- **ERR-001 to ERR-003**: Error handling
- **IT-001 to IT-003**: Integration tests

### Phase 4: Kubernetes Deployment (Week 2-3)
- **FR-014 to FR-017**: K8s manifests and RBAC
- **NFR-009 to NFR-012**: Observability
- **E2E-001 to E2E-004**: End-to-end tests

### Phase 5: Production Readiness (Week 3)
- **NFR-005 to NFR-008**: Performance tuning
- **ERR-004 to ERR-005**: Final error handling
- Documentation and deployment guide

---

## 7. Dependencies & Assumptions

### 7.1 Dependencies
- Kubernetes cluster v1.24+ with admission webhook support
- Go 1.21+ for operator development
- Container runtime that respects image rewrites (containerd, CRI-O, Docker)
- Network connectivity from API server to webhook service

### 7.2 Assumptions
- Target registries are pre-configured and accessible from cluster nodes
- Image pull secrets are already configured in namespaces if needed
- Cluster has sufficient resources to run the operator (256Mi memory, 500m CPU)
- Cluster DNS is functional for service discovery
- Users understand that image rewriting does NOT handle authentication

### 7.3 Constraints
- Webhook timeout limit: 10 seconds (Kubernetes default)
- Maximum pod size: 1.5MB (Kubernetes default for admission)
- Certificate renewal: Manual process (auto-renewal out of scope for v1.0)

---

## 8. Implementation Notes

### 8.1 Technology Stack
- **Language**: Go 1.21+
- **Framework**: controller-runtime or custom HTTP server
- **Testing**: Go standard library testing, testify for assertions
- **Kubernetes Client**: client-go v0.28+
- **TLS**: Go crypto/tls and crypto/x509 packages

### 8.2 Key Design Decisions
1. **Fail-Safe First**: `failurePolicy: Ignore` ensures pod creation is never blocked
2. **No State Management**: Webhook is stateless, reads namespace config on each request
3. **In-Memory Certificates**: TLS certs stored in memory, not persisted (simplicity)
4. **Path Preservation**: Image paths preserved to maintain registry structure compatibility

### 8.3 Development Workflow
1. TDD approach: Write tests first for each requirement
2. Small commits: Each FR/NFR/ERR requirement is a separate commit
3. Branch strategy: `feature/REQ-XXX-brief-description`
4. Use specialized agents for complex tasks (architecture, testing, deployment)

---

## 9. Acceptance & Sign-Off

### 9.1 Definition of Done
- [ ] All FR requirements implemented and tested
- [ ] All NFR requirements verified with metrics
- [ ] All ERR and EDGE cases handled with tests
- [ ] Unit test coverage >80%
- [ ] Integration tests passing
- [ ] E2E tests passing in local cluster
- [ ] Documentation complete (README, deployment guide)
- [ ] Code review completed
- [ ] Security review completed

### 9.2 Approval
- **Product Owner**: _________________________  Date: __________
- **Technical Lead**: _________________________  Date: __________
- **QA Lead**: _________________________  Date: __________

---

## 10. Appendix

### 10.1 Glossary
- **Mutating Admission Webhook**: K8s mechanism to modify resources before persistence
- **AdmissionReview**: K8s API object containing admission request/response
- **JSON Patch (RFC 6902)**: Standard for describing modifications to JSON documents
- **Fail-Safe**: Design pattern where failures result in safe default behavior
- **EARS**: Event-Action-Response System for requirements specification

### 10.2 Reference Materials
- [Kubernetes Admission Controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
- [Dynamic Admission Control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [JSON Patch RFC 6902](https://tools.ietf.org/html/rfc6902)
- [Container Image Specification](https://github.com/opencontainers/image-spec/blob/main/descriptor.md)

### 10.3 Revision History
| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-02 | Claude Code | Initial SRS creation |
