# VirtualService Operator

![Version](https://img.shields.io/badge/version-v1.2.3-blue)
![Go Version](https://img.shields.io/badge/go-1.21-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Architecture](https://img.shields.io/badge/arch-amd64%20%7C%20arm64-lightgrey)

A lightweight Kubernetes operator that automatically manages Istio VirtualServices for intelligent traffic routing between production and developer environments using header-based routing.

## 🚀 Overview

The VirtualService Operator simplifies multi-environment traffic management in Istio service meshes by automatically creating and managing VirtualServices based on Service deployments. It enables seamless traffic routing between a main namespace (production) and multiple developer namespaces using the `x-developer` header.

### Key Benefits

- **Zero Configuration Overhead**: No CRDs to manage - uses simple ConfigMap configuration
- **Automatic Service Discovery**: Dynamically creates routes only for services that exist
- **Intelligent Routing**: Header-based traffic splitting with fallback to production
- **Conflict Resolution**: Built-in retry logic with exponential backoff for concurrent updates
- **System Service Exclusion**: Automatically ignores Kubernetes system services
- **Multi-Architecture**: Supports both AMD64 and ARM64 platforms

## ✨ Features

- 🎯 **ConfigMap-Only Configuration** - No complex CRDs, just simple YAML configuration
- 🔄 **Automatic VirtualService Management** - Creates, updates, and deletes VirtualServices based on Service lifecycle
- 🏷️ **Header-Based Routing** - Routes traffic using `x-developer` header matching
- 🌐 **Multi-Namespace Support** - Manages traffic across multiple developer environments
- 🔒 **System Service Filtering** - Excludes kube-system and istio-system services automatically
- 🛡️ **Conflict Resolution** - Robust retry logic handles concurrent VirtualService updates
- 📊 **Observability** - Built-in metrics and health check endpoints
- 🏗️ **Multi-Architecture** - Native support for AMD64 and ARM64 architectures

## 🏗️ Architecture

The operator follows a clean, event-driven architecture:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   ConfigMap     │    │  Service Events  │    │ VirtualServices │
│  Configuration  │───▶│   Controller     │───▶│   Management    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │ Conflict Resolution│
                       │ & Retry Logic    │
                       └──────────────────┘
```

### Components

- **Service Controller**: Watches Service create/update/delete events in configured namespaces
- **Configuration Manager**: Reads operator configuration from ConfigMap with hot-reload capability
- **VirtualService Utils**: Handles VirtualService generation, templating, and lifecycle management
- **Retry Engine**: Manages conflict resolution with exponential backoff for concurrent updates

## ⚙️ Configuration

The operator is configured via a simple ConfigMap - no CRDs required:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "default"
    developerNamespaces:
      - "dev-alice"
      - "dev-bob"
      - "staging"
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "${serviceName}-virtual-service"
        namespace: "${defaultNamespace}"
      spec:
        hosts:
        - ${serviceName}
        http:
        # Developer routes will be injected here
        - route:
          - destination:
              host: ${serviceName}.${defaultNamespace}.svc.cluster.local
```

### Configuration Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `defaultNamespace` | Main production namespace | `"default"` |
| `developerNamespaces` | List of developer/staging namespaces | `["dev-alice", "staging"]` |
| `virtualServiceTemplate` | Template for generated VirtualServices | See example above |

## 📦 Installation

### Prerequisites

- Kubernetes cluster (v1.25+)
- Istio service mesh installed and configured
- `kubectl` configured to access your cluster

### Quick Install

```bash
# Deploy the operator
kubectl apply -f https://raw.githubusercontent.com/your-org/vs-operator/main/deployments/deployment.yaml
```

### Manual Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/your-org/vs-operator.git
   cd vs-operator
   ```

2. **Deploy the operator:**
   ```bash
   kubectl apply -f deployments/deployment.yaml
   ```

3. **Verify installation:**
   ```bash
   kubectl get pods -n virtualservice-operator-system
   kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
   ```

### Configuration Customization

Edit the ConfigMap to match your environment:

```bash
kubectl edit configmap virtualservice-operator-config -n virtualservice-operator-system
```

## 🎯 Usage

### Basic Usage

1. **Deploy a service in your default namespace:**
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: my-app
     namespace: default
   spec:
     selector:
       app: my-app
     ports:
     - port: 80
       targetPort: 8080
   ```

2. **Deploy the same service in developer namespaces:**
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: my-app
     namespace: dev-alice
   spec:
     selector:
       app: my-app
     ports:
     - port: 80
       targetPort: 8080
   ```

3. **The operator automatically creates a VirtualService:**
   ```yaml
   apiVersion: networking.istio.io/v1beta1
   kind: VirtualService
   metadata:
     name: my-app-virtual-service
     namespace: default
     labels:
       managed-by: virtualservice-operator
   spec:
     hosts:
     - my-app
     http:
     # Developer route
     - match:
       - headers:
           x-developer:
             exact: dev-alice
       route:
       - destination:
           host: my-app.dev-alice.svc.cluster.local
     # Default route
     - route:
       - destination:
           host: my-app.default.svc.cluster.local
   ```

### Traffic Routing Examples

```bash
# Route to production (default namespace)
curl http://my-app/api/health

# Route to Alice's development environment
curl -H "x-developer: dev-alice" http://my-app/api/health

# Route to staging environment
curl -H "x-developer: staging" http://my-app/api/health
```

## 🔄 Operator Behavior

### Service Lifecycle Management

#### Service in Default Namespace
- **Created**: Generates VirtualService with default route + routes to existing developer services
- **Updated**: Updates VirtualService if needed
- **Deleted**: Removes the entire VirtualService

#### Service in Developer Namespace
- **Created**: Adds developer route to existing VirtualService in default namespace
- **Updated**: Updates the corresponding route if needed
- **Deleted**: Removes the developer route from VirtualService

### Smart Service Discovery

The operator only creates routes for services that actually exist:

```yaml
# If you have services in: default, dev-alice, staging
# But NOT in: dev-bob
# The VirtualService will only include routes for: default, dev-alice, staging
```

### System Service Exclusion

Automatically excludes services from system namespaces:
- `kube-system`
- `kube-public`
- `kube-node-lease`
- `istio-system`
- `istio-injection`

### Conflict Resolution

Built-in retry logic handles concurrent VirtualService updates:
- Exponential backoff: 1s, 2s, 4s, 8s, 16s
- Automatic conflict detection and resolution
- Fetches latest resource version before each retry

## 🛠️ Development

### Building Locally

```bash
# Build the binary
make build

# Run locally (requires kubeconfig)
make run

# Run tests
make test
```

### Docker Build

```bash
# Build multi-architecture image
make docker-buildx IMG=your-registry/virtualservice-operator:latest

# Build for specific architecture
docker build --platform linux/amd64 -t your-registry/virtualservice-operator:latest .
```

### Project Structure

```
vs-operator/
├── controllers/           # Service controller logic
│   └── service_controller.go
├── internal/
│   ├── config/           # ConfigMap configuration management
│   │   └── config.go
│   └── utils/            # VirtualService utilities
│       └── virtualservice.go
├── deployments/          # Kubernetes manifests
│   └── deployment.yaml
├── main.go              # Application entry point
├── Dockerfile           # Multi-arch container build
└── Makefile            # Build automation
```

## 🔧 Troubleshooting

### Common Issues

#### VirtualService Not Created
```bash
# Check if service exists in default namespace
kubectl get svc -n default

# Check operator logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator

# Verify ConfigMap configuration
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml
```

#### Routes Not Updated
```bash
# Check if developer namespace service exists
kubectl get svc my-app -n dev-alice

# Force reconciliation by restarting operator
kubectl rollout restart deployment/virtualservice-operator -n virtualservice-operator-system
```

#### Conflict Errors
The operator automatically handles conflicts with retry logic. If you see persistent conflict errors:

```bash
# Check for multiple operator instances
kubectl get pods -n virtualservice-operator-system

# Verify RBAC permissions
kubectl auth can-i update virtualservices --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator
```

### Debug Mode

Enable debug logging:

```yaml
# In deployment.yaml, add to container args:
- -zap-log-level=debug
```

### Health Checks

The operator exposes health endpoints:

```bash
# Health check
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8081:8081
curl http://localhost:8081/healthz

# Metrics
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8080:8080
curl http://localhost:8080/metrics
```

## 📊 Monitoring

### Metrics

The operator exposes Prometheus metrics on `:8080/metrics`:

- `controller_runtime_*` - Controller runtime metrics
- `workqueue_*` - Work queue metrics
- `rest_client_*` - Kubernetes API client metrics

### Logging

Structured logging with configurable levels:
- `info` - Normal operations
- `debug` - Detailed debugging information
- `error` - Error conditions

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes** with tests
4. **Run the test suite**: `make test`
5. **Build and verify**: `make build`
6. **Commit your changes**: `git commit -m 'Add amazing feature'`
7. **Push to the branch**: `git push origin feature/amazing-feature`
8. **Open a Pull Request**

### Development Setup

```bash
# Install dependencies
go mod download

# Run tests
make test

# Run locally against your cluster
make run
```

## 📋 Examples

### Multi-Environment Setup

```yaml
# ConfigMap for multiple environments
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-a"
      - "dev-team-b"
      - "staging"
      - "qa"
```

### Custom VirtualService Template

```yaml
# Advanced template with custom routing
virtualServiceTemplate: |
  apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: "${serviceName}-vs"
    namespace: "${defaultNamespace}"
    labels:
      managed-by: virtualservice-operator
      service: "${serviceName}"
  spec:
    hosts:
    - ${serviceName}
    - ${serviceName}.${defaultNamespace}.svc.cluster.local
    http:
    # Developer routes will be injected here
    - route:
      - destination:
          host: ${serviceName}.${defaultNamespace}.svc.cluster.local
        weight: 100
```

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🏷️ Version History

- **v1.2.3** - ConfigMap-only architecture, removed CRD complexity
- **v1.2.2** - Added conflict resolution and retry logic
- **v1.2.1** - Service existence validation and system service exclusion
- **v1.2.0** - Multi-namespace support and improved routing
- **v1.1.0** - Header-based routing implementation
- **v1.0.0** - Initial release

---

**Image**: `harbor.intent.ai/library/virtualservice-operator:v1.2.3`

**Supported Architectures**: `linux/amd64`, `linux/arm64`