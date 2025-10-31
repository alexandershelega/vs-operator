# VirtualService Operator - Development Guide

## Overview

This guide provides comprehensive instructions for setting up a development environment, contributing to the project, testing procedures, and development best practices for the VirtualService Operator.

## Development Environment Setup

### Prerequisites

#### Required Tools
- **Go 1.21+**: Programming language runtime
- **Docker**: Container runtime for building images
- **kubectl**: Kubernetes command-line tool
- **kind/minikube**: Local Kubernetes cluster for testing
- **Git**: Version control system
- **Make**: Build automation tool

#### Optional Tools
- **Helm**: Package manager for Kubernetes
- **Istio CLI**: Service mesh management
- **VS Code**: Recommended IDE with Go extension
- **Delve**: Go debugger
- **golangci-lint**: Go linter

### Installation Steps

#### 1. Install Go
```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify installation
go version
```

#### 2. Install Docker
```bash
# macOS
brew install docker

# Linux (Ubuntu)
sudo apt-get update
sudo apt-get install docker.io
sudo usermod -aG docker $USER

# Verify installation
docker version
```

#### 3. Install kubectl
```bash
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Verify installation
kubectl version --client
```

#### 4. Install kind (Kubernetes in Docker)
```bash
# macOS
brew install kind

# Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Verify installation
kind version
```

#### 5. Install Istio
```bash
# Download Istio
curl -L https://istio.io/downloadIstio | sh -
export PATH=$PWD/istio-1.19.0/bin:$PATH

# Verify installation
istioctl version
```

### Local Development Cluster

#### Create kind Cluster
```bash
# Create cluster configuration
cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF

# Create cluster
kind create cluster --config kind-config.yaml --name vs-operator-dev

# Verify cluster
kubectl cluster-info --context kind-vs-operator-dev
```

#### Install Istio on kind
```bash
# Install Istio
istioctl install --set values.defaultRevision=default -y

# Enable sidecar injection for default namespace
kubectl label namespace default istio-injection=enabled

# Verify Istio installation
kubectl get pods -n istio-system
```

## Project Structure

### Directory Layout
```
vs-operator/
├── .github/                 # GitHub workflows and templates
│   └── workflows/
├── .trae/                   # Documentation
│   └── documents/
├── bin/                     # Compiled binaries
├── controllers/             # Controller implementations
│   └── service_controller.go
├── deployments/             # Kubernetes manifests
│   └── deployment.yaml
├── internal/                # Internal packages
│   ├── config/             # Configuration management
│   │   └── config.go
│   └── utils/              # Utility functions
│       └── virtualservice.go
├── templates/               # Template files
│   └── vs-template.yaml
├── test/                    # Test files and utilities
├── Dockerfile              # Container build definition
├── Makefile                # Build automation
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── main.go                 # Application entry point
└── README.md               # Project documentation
```

### Key Files

#### main.go
Application entry point that:
- Initializes the controller manager
- Sets up signal handling
- Configures logging
- Starts the reconciliation loop

#### controllers/service_controller.go
Core controller logic that:
- Watches Service resources
- Manages VirtualService lifecycle
- Handles placeholder service creation
- Implements retry logic for conflicts

#### internal/config/config.go
Configuration management that:
- Loads configuration from ConfigMap
- Validates configuration parameters
- Provides configuration hot-reload

#### internal/utils/virtualservice.go
VirtualService utilities that:
- Generate VirtualService from templates
- Update existing VirtualService routes
- Handle template processing

## Development Workflow

### Getting Started

#### 1. Clone Repository
```bash
git clone https://github.com/your-org/vs-operator.git
cd vs-operator
```

#### 2. Install Dependencies
```bash
# Download Go modules
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/go-delve/delve/cmd/dlv@latest
```

#### 3. Build Project
```bash
# Build binary
make build

# Run tests
make test

# Run linter
golangci-lint run
```

#### 4. Run Locally
```bash
# Run against local cluster
make run

# Or run with specific configuration
go run main.go \
  -config-map-name=virtualservice-operator-config \
  -config-map-namespace=virtualservice-operator-system \
  -metrics-bind-address=:8080 \
  -health-probe-bind-address=:8081
```

### Development Process

#### Feature Development
1. **Create Feature Branch**
   ```bash
   git checkout -b feature/new-feature
   ```

2. **Implement Changes**
   - Write code following Go best practices
   - Add comprehensive tests
   - Update documentation
   - Follow existing code patterns

3. **Test Changes**
   ```bash
   # Run unit tests
   make test
   
   # Run integration tests
   make test-integration
   
   # Test locally
   make run
   ```

4. **Code Review**
   ```bash
   # Format code
   make fmt
   
   # Run linter
   make vet
   golangci-lint run
   
   # Check for security issues
   gosec ./...
   ```

5. **Submit Pull Request**
   - Create descriptive PR title and description
   - Link related issues
   - Ensure CI passes
   - Request review from maintainers

### Code Style Guidelines

#### Go Style
Follow standard Go conventions:
- Use `gofmt` for formatting
- Follow effective Go guidelines
- Use meaningful variable names
- Add comments for exported functions
- Handle errors appropriately

#### Example Code Style
```go
// ServiceReconciler reconciles Service objects
type ServiceReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Config *config.OperatorConfig
}

// Reconcile handles Service reconciliation
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    
    // Get the service
    service := &corev1.Service{}
    if err := r.Get(ctx, req.NamespacedName, service); err != nil {
        if errors.IsNotFound(err) {
            // Service was deleted, handle cleanup
            return r.handleServiceDeletion(ctx, req.Name, req.Namespace, r.Config)
        }
        log.Error(err, "Failed to get service")
        return ctrl.Result{}, err
    }
    
    // Process service based on namespace
    if service.Namespace == r.Config.DefaultNamespace {
        return r.handleDefaultNamespaceService(ctx, service, r.Config)
    }
    
    return r.handleDeveloperNamespaceService(ctx, service, r.Config)
}
```

#### Error Handling
```go
// Good error handling
func (r *ServiceReconciler) createVirtualService(ctx context.Context, vs *v1beta1.VirtualService) error {
    if err := r.Create(ctx, vs); err != nil {
        if errors.IsAlreadyExists(err) {
            // Handle existing resource
            return r.updateVirtualService(ctx, vs)
        }
        return fmt.Errorf("failed to create VirtualService %s/%s: %w", vs.Namespace, vs.Name, err)
    }
    return nil
}
```

#### Logging
```go
// Structured logging
log := log.FromContext(ctx).WithValues(
    "service", service.Name,
    "namespace", service.Namespace,
    "virtualservice", vsName,
)

log.Info("Creating VirtualService")
log.Error(err, "Failed to create VirtualService", "error", err)
```

## Testing

### Unit Tests

#### Test Structure
```go
func TestServiceReconciler_Reconcile(t *testing.T) {
    tests := []struct {
        name           string
        service        *corev1.Service
        existingVS     *v1beta1.VirtualService
        config         *config.OperatorConfig
        expectedResult ctrl.Result
        expectedError  bool
    }{
        {
            name: "create VirtualService for new service",
            service: &corev1.Service{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-service",
                    Namespace: "default",
                },
            },
            config: &config.OperatorConfig{
                DefaultNamespace:    "default",
                DeveloperNamespaces: []string{"dev"},
            },
            expectedResult: ctrl.Result{},
            expectedError:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -run TestServiceReconciler_Reconcile ./controllers

# Run tests with verbose output
go test -v ./...
```

### Integration Tests

#### Test Environment Setup
```go
func setupTestEnvironment(t *testing.T) (*envtest.Environment, client.Client) {
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{
            filepath.Join("..", "config", "crd", "bases"),
        },
    }
    
    cfg, err := testEnv.Start()
    require.NoError(t, err)
    
    scheme := runtime.NewScheme()
    require.NoError(t, corev1.AddToScheme(scheme))
    require.NoError(t, v1beta1.AddToScheme(scheme))
    
    k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
    require.NoError(t, err)
    
    return testEnv, k8sClient
}
```

#### Integration Test Example
```go
func TestServiceController_Integration(t *testing.T) {
    testEnv, k8sClient := setupTestEnvironment(t)
    defer testEnv.Stop()
    
    // Create test namespace
    ns := &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
    }
    require.NoError(t, k8sClient.Create(context.Background(), ns))
    
    // Create test service
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-service",
            Namespace: "test-namespace",
        },
        Spec: corev1.ServiceSpec{
            Ports: []corev1.ServicePort{{Port: 80}},
        },
    }
    require.NoError(t, k8sClient.Create(context.Background(), service))
    
    // Test controller behavior
    // ...
}
```

### End-to-End Tests

#### E2E Test Setup
```bash
#!/bin/bash
# e2e-test.sh

set -e

# Create test cluster
kind create cluster --name e2e-test

# Install Istio
istioctl install --set values.defaultRevision=default -y

# Deploy operator
kubectl apply -f deployments/

# Wait for operator to be ready
kubectl wait --for=condition=available --timeout=300s deployment/virtualservice-operator -n virtualservice-operator-system

# Run tests
go test -tags=e2e ./test/e2e/...

# Cleanup
kind delete cluster --name e2e-test
```

#### E2E Test Example
```go
//go:build e2e
// +build e2e

func TestE2E_ServiceToVirtualService(t *testing.T) {
    // Create test service
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "e2e-test-service",
            Namespace: "default",
        },
        Spec: corev1.ServiceSpec{
            Selector: map[string]string{"app": "test"},
            Ports:    []corev1.ServicePort{{Port: 80}},
        },
    }
    
    err := k8sClient.Create(context.Background(), service)
    require.NoError(t, err)
    
    // Wait for VirtualService to be created
    vs := &v1beta1.VirtualService{}
    eventually := func() bool {
        err := k8sClient.Get(context.Background(), 
            types.NamespacedName{Name: "e2e-test-service-virtual-service", Namespace: "default"}, vs)
        return err == nil
    }
    
    require.Eventually(t, eventually, 30*time.Second, 1*time.Second)
    
    // Verify VirtualService content
    assert.Equal(t, "e2e-test-service", vs.Spec.Hosts[0])
    assert.NotEmpty(t, vs.Spec.Http)
}
```

## Debugging

### Local Debugging

#### Using Delve
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug main.go -- \
  -config-map-name=virtualservice-operator-config \
  -config-map-namespace=virtualservice-operator-system
```

#### VS Code Debug Configuration
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Operator",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/main.go",
            "args": [
                "-config-map-name=virtualservice-operator-config",
                "-config-map-namespace=virtualservice-operator-system",
                "-zap-log-level=debug"
            ],
            "env": {
                "KUBECONFIG": "${env:HOME}/.kube/config"
            }
        }
    ]
}
```

### Remote Debugging

#### Debug in Kubernetes
```yaml
# Debug deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: virtualservice-operator-debug
  namespace: virtualservice-operator-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: virtualservice-operator-debug
  template:
    metadata:
      labels:
        app: virtualservice-operator-debug
    spec:
      containers:
      - name: manager
        image: harbor.intent.ai/library/virtualservice-operator:debug
        command: ["/dlv"]
        args:
        - "--listen=:40000"
        - "--headless=true"
        - "--api-version=2"
        - "--accept-multiclient"
        - "exec"
        - "/manager"
        - "--"
        - "-config-map-name=virtualservice-operator-config"
        - "-config-map-namespace=virtualservice-operator-system"
        ports:
        - containerPort: 40000
          name: debug
```

```bash
# Port forward debug port
kubectl port-forward deployment/virtualservice-operator-debug 40000:40000 -n virtualservice-operator-system

# Connect with delve
dlv connect localhost:40000
```

### Troubleshooting Common Issues

#### Controller Not Starting
```bash
# Check pod status
kubectl get pods -n virtualservice-operator-system

# Check events
kubectl get events -n virtualservice-operator-system --sort-by='.lastTimestamp'

# Check logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator

# Common issues:
# - RBAC permissions
# - ConfigMap not found
# - Invalid configuration
# - Resource constraints
```

#### VirtualServices Not Created
```bash
# Check if services exist
kubectl get services -A

# Check operator logs for service processing
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep "service-name"

# Check configuration
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml

# Force reconciliation
kubectl annotate service my-service reconcile.virtualservice-operator/trigger="$(date)"
```

#### Performance Issues
```bash
# Check resource usage
kubectl top pod -n virtualservice-operator-system

# Check metrics
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8080:8080
curl http://localhost:8080/metrics | grep controller_runtime

# Enable profiling
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 6060:6060
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Contributing

### Contribution Guidelines

#### Before Contributing
1. Read the project README and documentation
2. Check existing issues and pull requests
3. Set up development environment
4. Run tests to ensure everything works

#### Making Contributions
1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow code style guidelines
   - Add tests for new functionality
   - Update documentation as needed
   - Ensure all tests pass

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a pull request**
   - Use descriptive title and description
   - Link related issues
   - Add screenshots if applicable
   - Request review from maintainers

### Commit Message Format
Follow conventional commits format:
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

Examples:
```
feat(controller): add placeholder service management
fix(virtualservice): resolve route conflict issue
docs(readme): update installation instructions
test(controller): add unit tests for service reconciliation
```

### Code Review Process

#### Review Checklist
- [ ] Code follows project style guidelines
- [ ] Tests are included and passing
- [ ] Documentation is updated
- [ ] No breaking changes (or properly documented)
- [ ] Security considerations addressed
- [ ] Performance impact considered

#### Review Guidelines
- Be constructive and respectful
- Focus on code quality and maintainability
- Suggest improvements with examples
- Ask questions for clarification
- Approve when ready or request changes

### Release Process

#### Version Management
- Follow semantic versioning (SemVer)
- Update version in relevant files
- Create release notes
- Tag releases appropriately

#### Release Steps
1. **Prepare Release**
   ```bash
   # Update version
   git checkout main
   git pull origin main
   
   # Create release branch
   git checkout -b release/v1.4.0
   
   # Update version in files
   # - README.md
   # - deployment.yaml
   # - Makefile
   ```

2. **Build and Test**
   ```bash
   # Run full test suite
   make test
   make test-integration
   
   # Build multi-arch images
   make docker-buildx IMG=harbor.intent.ai/library/virtualservice-operator:v1.4.0
   ```

3. **Create Release**
   ```bash
   # Merge release branch
   git checkout main
   git merge release/v1.4.0
   
   # Tag release
   git tag -a v1.4.0 -m "Release v1.4.0"
   git push origin main --tags
   ```

4. **Deploy Release**
   ```bash
   # Push images
   docker push harbor.intent.ai/library/virtualservice-operator:v1.4.0
   docker push harbor.intent.ai/library/virtualservice-operator:latest
   
   # Update deployment manifests
   kubectl set image deployment/virtualservice-operator manager=harbor.intent.ai/library/virtualservice-operator:v1.4.0 -n virtualservice-operator-system
   ```

## Best Practices

### Development Best Practices

#### Code Organization
- Keep functions small and focused
- Use meaningful names for variables and functions
- Organize code into logical packages
- Avoid circular dependencies
- Use interfaces for testability

#### Error Handling
- Always handle errors appropriately
- Use wrapped errors for context
- Log errors with sufficient detail
- Return meaningful error messages
- Don't ignore errors

#### Testing Best Practices
- Write tests before or alongside code
- Aim for high test coverage
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error conditions

#### Performance Considerations
- Use context for cancellation
- Implement proper resource cleanup
- Avoid memory leaks
- Use efficient data structures
- Profile performance-critical code

### Security Best Practices

#### Input Validation
- Validate all configuration inputs
- Sanitize user-provided data
- Use type-safe parsing
- Implement bounds checking
- Validate Kubernetes resource names

#### Secrets Management
- Never hardcode secrets
- Use Kubernetes secrets appropriately
- Rotate credentials regularly
- Limit secret access
- Audit secret usage

#### RBAC Principles
- Follow least privilege principle
- Use namespace-scoped permissions when possible
- Regularly audit permissions
- Document permission requirements
- Test with minimal permissions

### Operational Best Practices

#### Monitoring
- Expose relevant metrics
- Implement health checks
- Use structured logging
- Monitor resource usage
- Set up alerting

#### Documentation
- Keep documentation up to date
- Document configuration options
- Provide examples
- Include troubleshooting guides
- Document breaking changes

#### Deployment
- Use rolling updates
- Implement readiness probes
- Set resource limits
- Use multiple replicas for HA
- Test deployment procedures