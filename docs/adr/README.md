# Architecture Decision Records (ADR)

This directory contains Architecture Decision Records (ADRs) for the Kubernetes Registry Rewriting Operator.

## ADR Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-kubernetes-registry-rewriting-operator-architecture.md) | Kubernetes Registry Rewriting Operator Architecture | Proposed | 2025-10-02 |

## ADR Process

1. **Propose**: Create new ADR with status "Proposed"
2. **Review**: Team reviews and provides feedback
3. **Accept**: ADR status changes to "Accepted" when consensus reached
4. **Implement**: Begin implementation based on accepted ADR
5. **Supersede**: If decision changes, create new ADR and mark old one as "Superseded"

## Quick Reference: Key Decisions

### Technology Stack
- **Framework**: Kubebuilder (controller-runtime)
- **Language**: Go
- **Certificate Management**: cert-manager
- **Image Parsing**: go-containerregistry

### Architecture
- **Deployment**: Cluster-scoped operator with inline webhook
- **Configuration**: Namespace labels and annotations
- **Scope**: Watches namespaces with `registry-rewrite: "enabled"` label

### Next Steps
1. Initialize Kubebuilder project
2. Implement image parsing and rewriting logic
3. Build mutating webhook handler
4. Add comprehensive testing
5. Integrate cert-manager for TLS
