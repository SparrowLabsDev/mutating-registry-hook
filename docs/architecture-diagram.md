# Architecture Diagram

## Mutating Admission Controller Flow

```mermaid
sequenceDiagram
    autonumber
    participant User as User/Controller
    participant API as Kubernetes API Server
    participant Webhook as Mutating Webhook
    participant NS as Namespace Store
    participant Rewriter as Image Rewriter

    User->>API: Create Pod (team-a namespace)
    Note over User,API: spec.containers[0].image: nginx:latest

    API->>API: Validate request
    API->>Webhook: AdmissionReview Request
    Note over API,Webhook: POST /mutate--v1-pod<br/>TLS encrypted

    Webhook->>NS: Get Namespace(team-a)
    NS-->>Webhook: Namespace object

    alt Namespace has registry-rewrite: enabled
        Webhook->>Webhook: Check annotation:<br/>image-rewriter.example.com/target-registry

        alt Annotation exists
            loop For each container type
                Webhook->>Rewriter: RewriteImage(nginx:latest, team-a-registry.example.com)
                Rewriter-->>Webhook: team-a-registry.example.com/nginx:latest
            end

            Webhook->>API: AdmissionReview Response<br/>(allowed: true, patches: [...])
            Note over Webhook,API: JSON Patch to update images
        else No annotation
            Webhook->>API: AdmissionReview Response<br/>(allowed: true, patches: [])
            Note over Webhook,API: No changes
        end
    else Label not enabled
        Webhook->>API: AdmissionReview Response<br/>(allowed: true, patches: [])
        Note over Webhook,API: No changes
    end

    API->>API: Apply patches
    API->>User: Pod created successfully
    Note over User,API: spec.containers[0].image:<br/>team-a-registry.example.com/nginx:latest
```

## System Components

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        subgraph "mutating-registry-hook-system Namespace"
            WH[Webhook Server<br/>controller-manager]
            CERT[TLS Certificate<br/>webhook-server-cert]
            SVC[Service<br/>webhook-service:443]
        end

        subgraph "cert-manager Namespace"
            CM[cert-manager]
            ISSUER[Self-Signed Issuer]
        end

        API[API Server]
        MWC[MutatingWebhookConfiguration]

        subgraph "Application Namespaces"
            NS1[Namespace: team-a<br/>label: registry-rewrite=enabled<br/>annotation: target-registry=...]
            POD1[Pod]

            NS2[Namespace: team-b<br/>no label]
            POD2[Pod]
        end
    end

    API -->|1. Intercept Pod CREATE| MWC
    MWC -->|2. Call webhook| SVC
    SVC -->|3. Route to| WH
    WH -->|4. Read namespace| NS1
    WH -->|4. Read namespace| NS2
    WH -->|5. Rewrite images if enabled| POD1
    WH -.->|5. Skip if not enabled| POD2

    CM -->|Manages| CERT
    ISSUER -->|Issues| CERT
    SVC -->|Uses for TLS| CERT

    style WH fill:#4CAF50
    style MWC fill:#2196F3
    style NS1 fill:#FFC107
    style NS2 fill:#9E9E9E
    style POD1 fill:#8BC34A
    style POD2 fill:#BDBDBD
```

## Request/Response Flow

```mermaid
flowchart TD
    START([Pod Creation Request]) --> API[API Server]
    API --> CHECK{Mutating Webhooks<br/>configured?}

    CHECK -->|Yes| CALL[Call Webhook:<br/>POST /mutate--v1-pod]
    CHECK -->|No| ADMIT[Admit Pod]

    CALL --> AUTH{TLS Certificate<br/>Valid?}
    AUTH -->|No| FAIL[Webhook Fails]
    AUTH -->|Yes| WEBHOOK[Webhook Handler]

    WEBHOOK --> GETNS[Get Namespace]
    GETNS --> NSFAIL{Namespace<br/>found?}
    NSFAIL -->|No| ALLOW1[Return: allowed=true<br/>no patches]
    NSFAIL -->|Yes| CHKLBL

    CHKLBL{Label:<br/>registry-rewrite<br/>=enabled?}
    CHKLBL -->|No| ALLOW1
    CHKLBL -->|Yes| CHKANN

    CHKANN{Annotation:<br/>target-registry<br/>exists?}
    CHKANN -->|No| ALLOW1
    CHKANN -->|Yes| REWRITE

    REWRITE[Rewrite Images] --> GENCONT{For each<br/>container}
    GENCONT --> PARSE[Parse Image Reference]
    PARSE --> REPLACE[Replace Registry]
    REPLACE --> NEXT{More<br/>containers?}
    NEXT -->|Yes| GENCONT
    NEXT -->|No| PATCH

    PATCH[Generate JSON Patch] --> ALLOW2[Return: allowed=true<br/>patches=[...]]

    FAIL -->|failurePolicy:<br/>Ignore| ALLOW1
    ALLOW1 --> ADMIT
    ALLOW2 --> APPLY[Apply Patches]
    APPLY --> ADMIT

    ADMIT --> END([Pod Created])

    style START fill:#E3F2FD
    style END fill:#C8E6C9
    style WEBHOOK fill:#4CAF50
    style REWRITE fill:#FFC107
    style FAIL fill:#F44336
    style ADMIT fill:#8BC34A
```

## Namespace Configuration

```mermaid
classDiagram
    class Namespace {
        +metadata.labels
        +metadata.annotations
    }

    class EnabledNamespace {
        +labels["registry-rewrite"] = "enabled"
        +annotations["image-rewriter.example.com/target-registry"] = "team-a-registry.example.com"
        +behavior: Images rewritten
    }

    class DisabledNamespace {
        +labels: no registry-rewrite label
        +behavior: Images unchanged
    }

    class PartiallyConfiguredNamespace {
        +labels["registry-rewrite"] = "enabled"
        +annotations: missing target-registry
        +behavior: Images unchanged (fail-safe)
    }

    Namespace <|-- EnabledNamespace
    Namespace <|-- DisabledNamespace
    Namespace <|-- PartiallyConfiguredNamespace
```

## Image Rewriting Logic

```mermaid
flowchart LR
    INPUT[/"Original Image<br/>nginx:latest"/] --> PARSE[Parse Image]

    PARSE --> COMPONENTS{Extract Components}

    COMPONENTS --> REG[Registry:<br/>empty or docker.io]
    COMPONENTS --> REPO[Repository:<br/>nginx]
    COMPONENTS --> TAG[Tag:<br/>latest]
    COMPONENTS --> DIGEST[Digest:<br/>none]

    REG --> REBUILD
    REPO --> REBUILD[Rebuild Image Reference]
    TAG --> REBUILD
    DIGEST --> REBUILD

    REBUILD --> TARGET{Apply Target<br/>Registry}
    TARGET --> OUTPUT[/"Rewritten Image<br/>team-a-registry.example.com/nginx:latest"/]

    style INPUT fill:#E3F2FD
    style OUTPUT fill:#C8E6C9
    style REBUILD fill:#FFF9C4
```

## Security & Reliability

```mermaid
mindmap
    root((Mutating Registry<br/>Hook))
        Security
            TLS Encryption
                cert-manager managed
                Auto-rotation
                CA bundle injection
            RBAC
                Least privilege
                Namespace read-only
                Pod mutation only
            Network Policy
                Restricted ingress
                Webhook traffic only
        Reliability
            Fail-Safe Design
                failurePolicy: Ignore
                Never blocks pods
                Logs errors only
            Error Handling
                Invalid images → skip
                Missing namespace → skip
                Missing annotation → skip
            Observability
                Structured logging
                Metrics endpoint
                Health checks
        Performance
            No external calls
                Namespace cached
                In-memory processing
            Minimal latency
                &lt;100ms p95
                Simple string manipulation
```

## Deployment Architecture

```mermaid
C4Context
    title System Context - Mutating Registry Hook Operator

    Person(dev, "Developer", "Deploys pods to cluster")
    Person(ops, "Operations", "Manages operator")

    System_Boundary(k8s, "Kubernetes Cluster") {
        System(webhook, "Mutating Webhook", "Rewrites container images")
        System(api, "API Server", "Kubernetes control plane")
        System(certmgr, "cert-manager", "Certificate management")
    }

    System_Ext(registry, "Target Registry", "Private/mirror registry")

    Rel(dev, api, "Creates pods", "kubectl")
    Rel(ops, webhook, "Configures", "kubectl")
    Rel(api, webhook, "Calls for admission", "HTTPS")
    Rel(webhook, api, "Reads namespaces", "Kubernetes API")
    Rel(certmgr, webhook, "Provides TLS certs")
    Rel(api, registry, "Pulls images", "HTTPS")
```
