# ADR-0001: Kubernetes Registry Rewriting Operator Architecture

## Status
**Proposed** - 2025-10-02

## Context and Problem Statement

We need to build a Kubernetes operator that automatically rewrites container image registries for pods deployed in specific namespaces. This is required to support scenarios where:

- Organizations need to redirect pulls to internal/mirror registries for security, compliance, or cost reasons
- Different namespaces may require different target registries
- The rewriting should be transparent to application developers
- The solution must be reliable, performant, and maintainable

The operator must:
1. Watch for namespaces with the label `registry-rewrite: "enabled"`
2. Intercept pod creation via a mutating admission webhook
3. Rewrite image references based on the namespace annotation `image-rewriter.example.com/target-registry`
4. Handle various image reference formats (with/without registry, tags, digests)

## Decision Drivers

- **Reliability**: Must not disrupt pod creation or cause service outages
- **Performance**: Webhook latency must be minimal (< 100ms)
- **Maintainability**: Code should be clear, well-tested, and easy to evolve
- **Kubernetes Ecosystem Fit**: Should follow Kubernetes conventions and best practices
- **Security**: Must handle TLS certificates properly and validate inputs
- **Operational Simplicity**: Should be easy to deploy, configure, and troubleshoot
- **Development Velocity**: Framework should accelerate development, not hinder it

## Decision 1: Operator Framework

### Options Considered

#### Option 1: Kubebuilder (controller-runtime)
**Pros:**
- Official Kubernetes community project
- Excellent scaffolding and code generation
- Built-in support for webhooks with minimal boilerplate
- Strong testing utilities
- Active community and comprehensive documentation
- Follows Kubernetes API conventions by default
- Integrated with controller-runtime for efficient reconciliation

**Cons:**
- Some generated code may be more than needed for simple use cases
- Learning curve for developers unfamiliar with controller patterns

#### Option 2: Operator SDK
**Pros:**
- Built on top of Kubebuilder
- Additional tooling for operator lifecycle management
- Good for operators that need OLM (Operator Lifecycle Manager) integration

**Cons:**
- Additional abstraction layer over Kubebuilder
- More complex than needed for this use case
- Primarily valuable when targeting operator marketplaces (OperatorHub)

#### Option 3: Raw client-go
**Pros:**
- Maximum control and flexibility
- Minimal dependencies
- Deep understanding of Kubernetes API interactions

**Cons:**
- Significant boilerplate code required
- Manual implementation of watch logic, caching, work queues
- Webhook setup is entirely manual
- Higher maintenance burden
- Slower development velocity

### Decision: **Kubebuilder (controller-runtime)**

**Rationale:**
- Kubebuilder provides the best balance of productivity and control
- Webhook scaffolding eliminates error-prone manual TLS setup
- controller-runtime's caching and watch mechanisms are battle-tested
- The generated project structure follows Kubernetes conventions
- Strong typing and code generation reduce bugs
- Excellent testing support with envtest (real API server for integration tests)

## Decision 2: Programming Language

### Options Considered

#### Option 1: Go
**Pros:**
- Native language for Kubernetes ecosystem
- Kubernetes client libraries are first-class (client-go)
- Excellent performance and low resource footprint
- Strong static typing prevents runtime errors
- Fast compilation and cross-platform builds
- Large ecosystem of Kubernetes tooling

**Cons:**
- Steeper learning curve for developers unfamiliar with Go
- More verbose than some alternatives

#### Option 2: Python (Kopf framework)
**Pros:**
- Rapid prototyping
- Familiar to many developers
- Kopf provides good operator abstractions

**Cons:**
- Python Kubernetes client is less mature
- Higher memory footprint
- Performance concerns for webhook latency
- Deployment more complex (container size, dependencies)

#### Option 3: Rust (kube-rs)
**Pros:**
- Excellent performance and memory safety
- Growing Kubernetes ecosystem

**Cons:**
- Smaller community and fewer examples
- Longer compilation times
- Steeper learning curve
- Less mature tooling for operators

### Decision: **Go**

**Rationale:**
- Go is the de facto standard for Kubernetes operators
- Best performance characteristics for webhook latency requirements
- First-class support in Kubebuilder and controller-runtime
- Easier to find examples and community support
- Better operational characteristics (single binary, minimal container images)

## Decision 3: Certificate Management for Webhook

### Options Considered

#### Option 1: cert-manager
**Pros:**
- Industry standard for Kubernetes certificate management
- Automatic certificate rotation
- Well-tested and widely deployed
- Handles CA bundle injection into webhook configuration
- Integrates seamlessly with Kubebuilder

**Cons:**
- Additional dependency to install
- Requires cluster-admin privileges to install cert-manager initially
- Adds complexity to initial setup

#### Option 2: Self-signed certificates generated at startup
**Pros:**
- No external dependencies
- Simpler initial deployment
- Full control over certificate lifecycle

**Cons:**
- Manual certificate rotation required
- Manual CA bundle injection into webhook configuration
- More complex operator code
- Higher risk of certificate expiration issues

#### Option 3: External CA (Vault, cloud provider CA)
**Pros:**
- Enterprise-grade certificate management
- Centralized certificate policy

**Cons:**
- Tight coupling to specific infrastructure
- Reduces portability
- More complex setup and prerequisites

### Decision: **cert-manager with fallback documentation**

**Rationale:**
- cert-manager is the Kubernetes-native solution and handles the complex parts (rotation, CA injection)
- It's a reasonable prerequisite for production Kubernetes clusters
- Kubebuilder has built-in support for cert-manager markers
- For development/testing, we'll document how to use Kubebuilder's built-in development certificates
- This provides the best long-term operational experience

**Implementation Notes:**
- Use cert-manager `Certificate` resource for webhook server cert
- Use cert-manager `Injector` for automatic CA bundle injection
- Document manual certificate approach for environments without cert-manager

## Decision 4: Deployment Scope

### Options Considered

#### Option 1: Cluster-scoped deployment
**Pros:**
- Can watch all namespaces
- Single deployment for entire cluster
- Simpler operational model
- Matches typical operator patterns

**Cons:**
- Requires cluster-admin or broad RBAC permissions
- Single point of failure for entire cluster
- Cannot delegate control to namespace administrators

#### Option 2: Namespace-scoped deployment (per namespace)
**Pros:**
- Limited blast radius
- Can be managed by namespace administrators
- Better multi-tenancy support

**Cons:**
- Webhooks must be cluster-scoped (API limitation)
- Requires one deployment per target namespace
- More complex management at scale
- Webhook configuration is still cluster-scoped

#### Option 3: Hybrid (cluster-scoped operator, namespace-scoped permissions where possible)
**Pros:**
- Single deployment
- Minimal required permissions for most operations
- Clear security boundaries

**Cons:**
- Webhook itself still requires cluster-scoped configuration

### Decision: **Cluster-scoped deployment**

**Rationale:**
- Kubernetes webhooks are inherently cluster-scoped (MutatingWebhookConfiguration is cluster resource)
- Single deployment is simpler to operate and monitor
- Namespace filtering via label selectors provides sufficient isolation
- Standard pattern for admission webhook operators
- Resource efficiency (one deployment vs many)

**Security Considerations:**
- Use principle of least privilege in RBAC (read-only except for events)
- Webhook only processes namespaces with specific label
- No modifications to cluster state except pod mutation during admission

## Decision 5: Webhook Implementation Strategy

### Options Considered

#### Option 1: Inline webhook handler in operator
**Pros:**
- Single deployment
- Shared code and dependencies
- Simpler deployment model

**Cons:**
- Controller reconciliation and webhook run in same process
- Webhook latency affects controller performance and vice versa

#### Option 2: Separate webhook server deployment
**Pros:**
- Independent scaling of webhook and controller
- Isolation of concerns
- Webhook can be optimized purely for latency

**Cons:**
- Two deployments to manage
- More complex deployment
- Code duplication or shared library needed

#### Option 3: Inline with separate ports and metrics
**Pros:**
- Single deployment
- Separate metrics endpoints for observability
- Can still optimize webhook path independently

**Cons:**
- Shared process resources

### Decision: **Inline webhook handler with dedicated metrics**

**Rationale:**
- For this use case, the controller is extremely lightweight (just watching namespace labels)
- Kubebuilder's default pattern is inline, simplifying development
- Can monitor webhook performance independently via metrics
- Easier to deploy and operate
- Can separate later if needed without API changes

**Implementation Notes:**
- Expose separate metrics for webhook latency
- Implement proper timeouts and resource limits
- Use separate logger tags for webhook vs controller

## Decision 6: Image Rewrite Logic

### Design Decisions

#### Image Reference Parsing Strategy
**Decision:** Use container image reference parsing library (e.g., `github.com/google/go-containerregistry/pkg/name`)

**Rationale:**
- Image references have complex syntax (registry, repository, tag, digest)
- Well-tested library reduces bugs
- Handles edge cases (implicit docker.io, library/ prefix, localhost)

#### Image Format Handling

The webhook must handle these formats correctly:

| Input Format | Example | Rewrite Strategy |
|-------------|---------|------------------|
| Image only | `nginx` | `{target-registry}/library/nginx` |
| Image:tag | `nginx:1.20` | `{target-registry}/library/nginx:1.20` |
| Repository/image | `myrepo/myapp` | `{target-registry}/myrepo/myapp` |
| Registry/repo/image | `gcr.io/project/app` | `{target-registry}/project/app` |
| Image@digest | `nginx@sha256:abc123...` | `{target-registry}/library/nginx@sha256:abc123...` |
| Full reference | `gcr.io/proj/app:v1@sha256:abc...` | `{target-registry}/proj/app:v1@sha256:abc...` |

**Decision:** Preserve repository paths and tags/digests, only replace registry

**Rationale:**
- Preserves semantic meaning of repository structure
- Maintains version/digest specificity
- Mirrors are typically structured to preserve paths
- Simplifies validation and testing

#### Special Cases

1. **localhost and private IPs**: Skip rewriting (likely local development)
2. **Already matching target registry**: Skip rewriting (idempotent)
3. **Invalid image references**: Reject pod creation with clear error message
4. **Missing target-registry annotation**: Skip rewriting with warning event

### Decision: **Parser-based rewriting with explicit special case handling**

**Rationale:**
- Correctness is critical (broken image refs prevent pod creation)
- Library-based parsing is more reliable than regex
- Explicit special cases make behavior predictable
- Easy to test with comprehensive test matrix

## Decision 7: Configuration and Extensibility

### Configuration Strategy

**Decision:** Use namespace annotations for per-namespace configuration, with potential for ConfigMap-based global defaults

**Current Configuration:**
- `image-rewriter.example.com/target-registry`: Target registry for this namespace (REQUIRED)

**Potential Future Configuration:**
- `image-rewriter.example.com/skip-verification`: Skip TLS verification (default: false)
- `image-rewriter.example.com/preserve-registry`: Comma-separated list of registries to not rewrite
- Global ConfigMap for default settings

**Rationale:**
- Annotations are namespace-scoped and follow Kubernetes patterns
- Easy to read in webhook handler (passed in admission review)
- No additional API calls required during webhook execution
- Can extend without breaking changes

## Consequences

### Positive

1. **Development Velocity**: Kubebuilder scaffolding accelerates initial development and reduces boilerplate
2. **Reliability**: Using established frameworks and libraries reduces bug surface area
3. **Performance**: Go implementation meets webhook latency requirements
4. **Operational Simplicity**: Single deployment, standard Kubernetes patterns
5. **Security**: cert-manager provides automatic certificate rotation
6. **Maintainability**: Well-structured code following Kubernetes conventions
7. **Testing**: envtest enables comprehensive integration testing without full cluster
8. **Community Support**: Using standard tools means abundant examples and help available

### Negative

1. **cert-manager Dependency**: Requires cert-manager installation (though common in modern clusters)
2. **Go Learning Curve**: Team members unfamiliar with Go need to learn it
3. **Framework Lock-in**: Kubebuilder patterns may make it harder to migrate to different approaches
4. **Cluster-scoped Permissions**: Requires elevated permissions at installation time
5. **Single Process**: Webhook and controller share resources (though mitigated by lightweight controller)

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Webhook latency impacts pod creation | HIGH | Implement timeout, monitoring, and performance testing |
| Certificate expiration breaks webhook | HIGH | Use cert-manager for automatic rotation, monitor cert expiry |
| Image parsing bugs prevent pod creation | HIGH | Comprehensive test suite, fail-safe fallbacks, clear error messages |
| Cluster-wide failure if operator crashes | MEDIUM | Implement health checks, resource limits, and webhook failure policy |
| Configuration errors cause wrong rewrites | MEDIUM | Validation webhook for namespace annotations, audit logging |

## Implementation Plan

### Phase 1: Foundation
1. Initialize Kubebuilder project
2. Set up basic webhook structure
3. Implement image reference parsing
4. Create comprehensive unit tests

### Phase 2: Core Functionality
1. Implement namespace label watching
2. Implement pod mutation webhook
3. Add annotation-based configuration
4. Integration tests with envtest

### Phase 3: Production Readiness
1. Add cert-manager integration
2. Implement metrics and observability
3. Create deployment manifests
4. Documentation and examples

### Phase 4: Hardening
1. Performance testing and optimization
2. Failure mode testing
3. Security review
4. Operational runbooks

## Related Decisions

- **Future ADR**: Monitoring and observability strategy
- **Future ADR**: Multi-tenancy and security boundaries
- **Future ADR**: Registry authentication and credentials management (if needed)

## References

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Kubernetes Admission Controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [go-containerregistry](https://github.com/google/go-containerregistry)

## Notes

This ADR represents the initial architectural decisions. As implementation progresses, we may need to create additional ADRs for:
- Registry credential management (if mirror requires authentication)
- Image signature verification integration
- Multi-registry fallback strategies
- Performance optimization approaches

---

**Decision Date:** 2025-10-02
**Reviewers:** [To be added]
**Status:** Proposed → Accepted (pending implementation validation)
