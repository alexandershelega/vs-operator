# VirtualService Operator - Configuration Reference

## Overview

The VirtualService Operator is configured entirely through Kubernetes ConfigMaps, eliminating the need for Custom Resource Definitions (CRDs). This approach simplifies deployment and reduces cluster complexity while providing comprehensive configuration options.

## Configuration Structure

### Primary Configuration

The operator reads its configuration from a ConfigMap specified during startup. The default configuration is located at:

- **ConfigMap Name**: `virtualservice-operator-config`
- **Namespace**: `virtualservice-operator-system`
- **Key**: `config.yaml`

### Configuration Schema

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    # Default namespace where production services are deployed
    defaultNamespace: "production"
    
    # List of developer namespaces for traffic routing
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "staging"
    
    # Enable automatic placeholder service creation
    enablePlaceholderServices: true
    
    # VirtualService template for generating resources
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

## Configuration Parameters

### Core Parameters

#### defaultNamespace
- **Type**: `string`
- **Required**: Yes
- **Description**: The namespace where production services are deployed. Services in this namespace will have VirtualServices created for them.
- **Example**: `"production"`
- **Default**: `"default"`

```yaml
defaultNamespace: "production"
```

#### developerNamespaces
- **Type**: `[]string`
- **Required**: Yes
- **Description**: List of namespaces that contain developer services. Traffic with matching `x-developer` headers will be routed to services in these namespaces.
- **Example**: `["dev-team-1", "dev-team-2", "staging"]`
- **Default**: `[]`

```yaml
developerNamespaces:
  - "dev-team-1"
  - "dev-team-2"
  - "staging"
  - "feature-branch-ns"
```

#### enablePlaceholderServices
- **Type**: `boolean`
- **Required**: No
- **Description**: Enable automatic creation of placeholder services when services are deleted from the default namespace but still referenced by VirtualServices.
- **Example**: `true`
- **Default**: `false`

```yaml
enablePlaceholderServices: true
```

#### virtualServiceTemplate
- **Type**: `string` (YAML template)
- **Required**: Yes
- **Description**: Go template for generating VirtualService resources. Supports template variables for dynamic content generation.
- **Template Variables**:
  - `{{.ServiceName}}`: Name of the source service
  - `{{.ServiceNamespace}}`: Namespace of the source service
  - `{{.DeveloperName}}`: Developer identifier from namespace
  - `{{.DeveloperNamespace}}`: Developer namespace

### Template Variables Reference

#### Available Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `{{.ServiceName}}` | string | Name of the service | `"user-service"` |
| `{{.ServiceNamespace}}` | string | Namespace of the service | `"production"` |
| `{{.DeveloperName}}` | string | Developer identifier extracted from namespace | `"team-1"` |
| `{{.DeveloperNamespace}}` | string | Full developer namespace name | `"dev-team-1"` |

#### Template Functions

The template engine supports standard Go template functions plus custom functions:

```yaml
virtualServiceTemplate: |
  apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: "{{.ServiceName}}-virtual-service"
    namespace: "{{.ServiceNamespace}}"
    labels:
      service: "{{.ServiceName}}"
      managed-by: "virtualservice-operator"
      # Use template functions
      timestamp: "{{now.Format "2006-01-02T15:04:05Z"}}"
  spec:
    hosts:
    - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
    # Conditional logic
    {{- if .DeveloperNamespace}}
    - "{{.ServiceName}}-dev.example.com"
    {{- end}}
```

## Configuration Examples

### Basic Configuration

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
      - "dev"
    enablePlaceholderServices: false
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-vs"
        namespace: "{{.ServiceNamespace}}"
      spec:
        hosts:
        - "{{.ServiceName}}"
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

### Multi-Environment Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "dev-team-3"
      - "staging"
      - "qa"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          app.kubernetes.io/managed-by: "virtualservice-operator"
          app.kubernetes.io/component: "virtualservice"
          service: "{{.ServiceName}}"
          environment: "{{.ServiceNamespace}}"
        annotations:
          virtualservice-operator/source-service: "{{.ServiceName}}.{{.ServiceNamespace}}"
          virtualservice-operator/created-at: "{{now.Format "2006-01-02T15:04:05Z"}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}.example.com"
        gateways:
        - istio-system/main-gateway
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
            weight: 100
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
            weight: 100
```

### Advanced Configuration with Traffic Splitting

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "staging"
      - "canary"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          app.kubernetes.io/managed-by: "virtualservice-operator"
          service: "{{.ServiceName}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}.example.com"
        gateways:
        - istio-system/main-gateway
        http:
        # Developer routing
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        # Canary routing (5% traffic)
        - match:
          - headers:
              x-canary:
                exact: "true"
          route:
          - destination:
              host: "{{.ServiceName}}.canary.svc.cluster.local"
            weight: 5
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
            weight: 95
        # Default routing
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

### Configuration with Custom Headers and Fault Injection

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "testing"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        http:
        # Developer routing with custom headers
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
            headers:
              request:
                add:
                  x-developer-env: "{{.DeveloperNamespace}}"
                  x-routed-by: "virtualservice-operator"
        # Testing environment with fault injection
        - match:
          - headers:
              x-developer:
                exact: "testing"
          fault:
            delay:
              percentage:
                value: 10
              fixedDelay: 5s
            abort:
              percentage:
                value: 1
              httpStatus: 500
          route:
          - destination:
              host: "{{.ServiceName}}.testing.svc.cluster.local"
        # Production routing
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

## Configuration Management

### Updating Configuration

#### Method 1: kubectl patch
```bash
# Update a single parameter
kubectl patch configmap virtualservice-operator-config \
  -n virtualservice-operator-system \
  --type merge \
  -p '{"data":{"config.yaml":"defaultNamespace: \"production\"\ndeveloperNamespaces:\n  - \"dev-team-1\"\n  - \"dev-team-2\"\n  - \"new-team\"\nenablePlaceholderServices: true"}}'
```

#### Method 2: kubectl edit
```bash
# Edit configuration interactively
kubectl edit configmap virtualservice-operator-config -n virtualservice-operator-system
```

#### Method 3: Apply updated manifest
```bash
# Apply updated configuration file
kubectl apply -f updated-config.yaml
```

### Configuration Reload

The operator automatically detects configuration changes and reloads without restart:

```bash
# Check if configuration was reloaded
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep "Configuration reloaded"

# Force operator restart if needed
kubectl rollout restart deployment/virtualservice-operator -n virtualservice-operator-system
```

### Configuration Validation

#### Pre-deployment Validation
```bash
# Validate YAML syntax
kubectl apply --dry-run=client -f config.yaml

# Validate template syntax
kubectl create configmap test-config --from-file=config.yaml --dry-run=client -o yaml
```

#### Runtime Validation
```bash
# Check operator logs for configuration errors
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i "config\|error"

# Test configuration by creating a test service
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: config-test-service
  namespace: production
spec:
  selector:
    app: test
  ports:
  - port: 80
EOF

# Verify VirtualService creation
kubectl get virtualservice config-test-service-virtual-service -o yaml

# Clean up
kubectl delete service config-test-service
```

## Environment-Specific Configurations

### Development Environment

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
      - "dev"
      - "feature"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-vs"
        namespace: "{{.ServiceNamespace}}"
        labels:
          environment: "development"
      spec:
        hosts:
        - "{{.ServiceName}}.local"
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

### Staging Environment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "staging"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "qa"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          environment: "staging"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}-staging.example.com"
        gateways:
        - istio-system/staging-gateway
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

### Production Environment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "dev-team-3"
      - "staging"
      - "canary"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          app.kubernetes.io/managed-by: "virtualservice-operator"
          environment: "production"
          service: "{{.ServiceName}}"
        annotations:
          virtualservice-operator/source-service: "{{.ServiceName}}.{{.ServiceNamespace}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}.example.com"
        gateways:
        - istio-system/main-gateway
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
          timeout: 30s
          retries:
            attempts: 3
            perTryTimeout: 10s
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
          timeout: 30s
          retries:
            attempts: 3
            perTryTimeout: 10s
```

## Advanced Configuration Patterns

### Multi-Cluster Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}.global"
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        - match:
          - headers:
              x-cluster:
                exact: "cluster-2"
          route:
          - destination:
              host: "{{.ServiceName}}.production.svc.cluster.local"
            subset: cluster-2
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

### Blue-Green Deployment Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "blue"
      - "green"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        http:
        - match:
          - headers:
              x-developer:
                exact: "{{.DeveloperName}}"
          route:
          - destination:
              host: "{{.ServiceName}}.{{.DeveloperNamespace}}.svc.cluster.local"
        - match:
          - headers:
              x-deployment:
                exact: "blue"
          route:
          - destination:
              host: "{{.ServiceName}}.blue.svc.cluster.local"
        - match:
          - headers:
              x-deployment:
                exact: "green"
          route:
          - destination:
              host: "{{.ServiceName}}.green.svc.cluster.local"
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

## Configuration Troubleshooting

### Common Configuration Issues

#### 1. Invalid YAML Syntax
```bash
# Validate YAML syntax
kubectl apply --dry-run=client -f config.yaml

# Check for common issues:
# - Incorrect indentation
# - Missing quotes around strings
# - Invalid template syntax
```

#### 2. Template Rendering Errors
```bash
# Check operator logs for template errors
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i template

# Common template issues:
# - Undefined variables
# - Incorrect template syntax
# - Missing template delimiters
```

#### 3. Configuration Not Loading
```bash
# Verify ConfigMap exists
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system

# Check operator startup arguments
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o yaml | grep args

# Verify operator has permission to read ConfigMap
kubectl auth can-i get configmaps --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator -n virtualservice-operator-system
```

### Configuration Validation Script

```bash
#!/bin/bash
# validate-config.sh

CONFIG_FILE="$1"
NAMESPACE="virtualservice-operator-system"
CONFIGMAP_NAME="virtualservice-operator-config"

if [ -z "$CONFIG_FILE" ]; then
    echo "Usage: $0 <config-file>"
    exit 1
fi

echo "Validating configuration file: $CONFIG_FILE"

# Check YAML syntax
echo "Checking YAML syntax..."
kubectl apply --dry-run=client -f "$CONFIG_FILE" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "ERROR: Invalid YAML syntax"
    kubectl apply --dry-run=client -f "$CONFIG_FILE"
    exit 1
fi
echo "✓ YAML syntax is valid"

# Check required fields
echo "Checking required fields..."
if ! grep -q "defaultNamespace:" "$CONFIG_FILE"; then
    echo "ERROR: Missing required field: defaultNamespace"
    exit 1
fi
echo "✓ Required fields present"

# Apply configuration
echo "Applying configuration..."
kubectl apply -f "$CONFIG_FILE"

# Wait for operator to reload
echo "Waiting for configuration reload..."
sleep 5

# Check operator logs
echo "Checking operator logs..."
kubectl logs -n "$NAMESPACE" deployment/virtualservice-operator --tail=20 | grep -i "config\|error"

echo "Configuration validation complete"
```

### Configuration Backup and Restore

#### Backup Configuration
```bash
#!/bin/bash
# backup-config.sh

NAMESPACE="virtualservice-operator-system"
CONFIGMAP_NAME="virtualservice-operator-config"
BACKUP_DIR="./config-backups"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"

kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/config-$TIMESTAMP.yaml"
echo "Configuration backed up to: $BACKUP_DIR/config-$TIMESTAMP.yaml"
```

#### Restore Configuration
```bash
#!/bin/bash
# restore-config.sh

BACKUP_FILE="$1"
NAMESPACE="virtualservice-operator-system"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file>"
    exit 1
fi

echo "Restoring configuration from: $BACKUP_FILE"
kubectl apply -f "$BACKUP_FILE"

echo "Restarting operator to reload configuration..."
kubectl rollout restart deployment/virtualservice-operator -n "$NAMESPACE"

echo "Configuration restored successfully"
```

This comprehensive configuration reference provides all the information needed to configure and manage the VirtualService Operator effectively across different environments and use cases.