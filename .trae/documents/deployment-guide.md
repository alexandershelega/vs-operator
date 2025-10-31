# VirtualService Operator - Deployment Guide

## Overview

This guide provides comprehensive deployment instructions for the VirtualService Operator across different environments, from development to production. The operator manages Istio VirtualServices automatically based on Kubernetes Services with header-based traffic routing.

## Prerequisites

### Kubernetes Cluster Requirements

#### Minimum Requirements
- **Kubernetes Version**: 1.20+
- **CPU**: 2 cores minimum, 4 cores recommended
- **Memory**: 4GB minimum, 8GB recommended
- **Storage**: 20GB minimum for system components
- **Network**: CNI plugin installed (Calico, Flannel, etc.)

#### Supported Platforms
- **Cloud Providers**: AWS EKS, Google GKE, Azure AKS
- **On-Premises**: kubeadm, kops, Rancher
- **Local Development**: kind, minikube, k3s
- **Managed Services**: OpenShift, VMware Tanzu

### Istio Service Mesh

#### Istio Installation
The operator requires Istio to be installed and configured in the cluster.

```bash
# Download Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-1.19.0
export PATH=$PWD/bin:$PATH

# Install Istio with default configuration
istioctl install --set values.defaultRevision=default -y

# Verify installation
kubectl get pods -n istio-system
```

#### Istio Configuration
```bash
# Enable sidecar injection for relevant namespaces
kubectl label namespace default istio-injection=enabled
kubectl label namespace production istio-injection=enabled

# Verify Istio components
istioctl proxy-status
istioctl analyze
```

### RBAC Permissions

The operator requires specific permissions to manage Services, VirtualServices, and ConfigMaps:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: virtualservice-operator-role
rules:
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["networking.istio.io"]
  resources: ["virtualservices"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Quick Start Deployment

### 1. Deploy Operator
```bash
# Apply the deployment manifest
kubectl apply -f https://raw.githubusercontent.com/your-org/vs-operator/main/deployments/deployment.yaml

# Verify deployment
kubectl get pods -n virtualservice-operator-system
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

### 2. Configure Operator
```bash
# Check default configuration
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml

# The operator will start managing services immediately with default settings
```

### 3. Test Deployment
```bash
# Create a test service
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
spec:
  selector:
    app: test-app
  ports:
  - port: 80
    targetPort: 8080
EOF

# Verify VirtualService creation
kubectl get virtualservice test-service-virtual-service -o yaml
```

## Environment-Specific Deployments

### Development Environment

#### Local Development with kind
```bash
# Create kind cluster
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
  - containerPort: 443
    hostPort: 443
EOF

kind create cluster --config kind-config.yaml --name vs-operator-dev
```

#### Development Configuration
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
      - "staging"
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
        - route:
          - destination:
              host: "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
```

#### Development Deployment
```bash
# Deploy with development image
kubectl set image deployment/virtualservice-operator \
  manager=harbor.intent.ai/library/virtualservice-operator:dev \
  -n virtualservice-operator-system

# Enable debug logging
kubectl patch deployment virtualservice-operator \
  -n virtualservice-operator-system \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "-zap-log-level=debug"}]'
```

### Staging Environment

#### Staging Configuration
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
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          app.kubernetes.io/managed-by: "virtualservice-operator"
          environment: "staging"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}-staging.example.com"
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

#### Staging Deployment
```bash
# Deploy staging version
kubectl apply -f deployments/deployment.yaml

# Update image to staging tag
kubectl set image deployment/virtualservice-operator \
  manager=harbor.intent.ai/library/virtualservice-operator:v1.3.0 \
  -n virtualservice-operator-system

# Configure resource limits for staging
kubectl patch deployment virtualservice-operator \
  -n virtualservice-operator-system \
  --type='json' \
  -p='[
    {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/cpu", "value": "500m"},
    {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "512Mi"},
    {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/cpu", "value": "200m"},
    {"op": "replace", "path": "/spec/template/spec/containers/0/resources/requests/memory", "value": "256Mi"}
  ]'
```

### Production Environment

#### Production Prerequisites
```bash
# Verify cluster readiness
kubectl cluster-info
kubectl get nodes
kubectl get pods -n istio-system

# Check resource availability
kubectl top nodes
kubectl describe nodes | grep -A 5 "Allocated resources"

# Verify RBAC
kubectl auth can-i create virtualservices --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator
```

#### Production Configuration
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
          version: "v1"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}.example.com"
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

#### Production Deployment
```yaml
# production-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: virtualservice-operator
  namespace: virtualservice-operator-system
  labels:
    app: virtualservice-operator
    environment: production
spec:
  replicas: 2  # High availability
  selector:
    matchLabels:
      app: virtualservice-operator
  template:
    metadata:
      labels:
        app: virtualservice-operator
    spec:
      serviceAccountName: virtualservice-operator
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        fsGroup: 65532
      containers:
      - name: manager
        image: harbor.intent.ai/library/virtualservice-operator:v1.3.0
        args:
        - -config-map-name=virtualservice-operator-config
        - -config-map-namespace=virtualservice-operator-system
        - -metrics-bind-address=:8080
        - -health-probe-bind-address=:8081
        - -zap-log-level=info
        ports:
        - containerPort: 8080
          name: metrics
          protocol: TCP
        - containerPort: 8081
          name: health
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        resources:
          limits:
            cpu: 1000m
            memory: 1Gi
          requests:
            cpu: 500m
            memory: 512Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
      terminationGracePeriodSeconds: 30
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - virtualservice-operator
              topologyKey: kubernetes.io/hostname
```

```bash
# Deploy production configuration
kubectl apply -f production-deployment.yaml

# Verify deployment
kubectl get pods -n virtualservice-operator-system -o wide
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

## Multi-Environment Setup

### Namespace Strategy
```bash
# Create environment namespaces
kubectl create namespace production
kubectl create namespace staging
kubectl create namespace dev-team-1
kubectl create namespace dev-team-2

# Label namespaces for Istio injection
kubectl label namespace production istio-injection=enabled
kubectl label namespace staging istio-injection=enabled
kubectl label namespace dev-team-1 istio-injection=enabled
kubectl label namespace dev-team-2 istio-injection=enabled

# Add environment labels
kubectl label namespace production environment=production
kubectl label namespace staging environment=staging
kubectl label namespace dev-team-1 environment=development team=team-1
kubectl label namespace dev-team-2 environment=development team=team-2
```

### Network Policies
```yaml
# production-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: production-isolation
  namespace: production
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: istio-system
  - from:
    - namespaceSelector:
        matchLabels:
          environment: production
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: istio-system
  - to:
    - namespaceSelector:
        matchLabels:
          environment: production
  - to: []  # Allow external traffic
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
    - protocol: UDP
      port: 53
```

### Resource Quotas
```yaml
# production-resource-quota.yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: production-quota
  namespace: production
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
    persistentvolumeclaims: "10"
    services: "20"
    secrets: "20"
    configmaps: "20"
---
apiVersion: v1
kind: LimitRange
metadata:
  name: production-limits
  namespace: production
spec:
  limits:
  - default:
      cpu: "1"
      memory: "1Gi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
    type: Container
```

## Configuration Management

### ConfigMap Structure
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
    
    # List of developer namespaces
    developerNamespaces:
      - "dev-team-1"
      - "dev-team-2"
      - "staging"
    
    # Enable automatic placeholder service creation
    enablePlaceholderServices: true
    
    # VirtualService template
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          app.kubernetes.io/managed-by: "virtualservice-operator"
          app.kubernetes.io/component: "virtualservice"
          app.kubernetes.io/part-of: "traffic-routing"
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

### Configuration Updates
```bash
# Update configuration
kubectl patch configmap virtualservice-operator-config \
  -n virtualservice-operator-system \
  --type merge \
  -p '{"data":{"config.yaml":"defaultNamespace: \"production\"\ndeveloperNamespaces:\n  - \"dev-team-1\"\n  - \"dev-team-2\"\n  - \"dev-team-3\"\n  - \"staging\"\nenablePlaceholderServices: true"}}'

# Restart operator to reload configuration
kubectl rollout restart deployment/virtualservice-operator -n virtualservice-operator-system

# Verify configuration reload
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep "Configuration loaded"
```

### Environment-Specific Templates
```yaml
# staging-config.yaml
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
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "{{.ServiceName}}-virtual-service"
        namespace: "{{.ServiceNamespace}}"
        labels:
          app.kubernetes.io/managed-by: "virtualservice-operator"
          environment: "staging"
      spec:
        hosts:
        - "{{.ServiceName}}.{{.ServiceNamespace}}.svc.cluster.local"
        - "{{.ServiceName}}-staging.example.com"
        gateways:
        - staging-gateway
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

## Monitoring and Health Checks

### Health Endpoints
```bash
# Check operator health
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8081:8081
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz

# Check metrics
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8080:8080
curl http://localhost:8080/metrics
```

### Monitoring Setup
```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: virtualservice-operator
  namespace: virtualservice-operator-system
  labels:
    app: virtualservice-operator
spec:
  selector:
    matchLabels:
      app: virtualservice-operator
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Alerting Rules
```yaml
# alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: virtualservice-operator-alerts
  namespace: virtualservice-operator-system
spec:
  groups:
  - name: virtualservice-operator
    rules:
    - alert: VirtualServiceOperatorDown
      expr: up{job="virtualservice-operator"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "VirtualService Operator is down"
        description: "VirtualService Operator has been down for more than 5 minutes"
    
    - alert: VirtualServiceOperatorHighErrorRate
      expr: rate(controller_runtime_reconcile_errors_total[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High error rate in VirtualService Operator"
        description: "VirtualService Operator error rate is {{ $value }} errors per second"
```

## Backup and Recovery

### Backup Strategy
```bash
# Backup operator configuration
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml > operator-config-backup.yaml

# Backup all VirtualServices managed by operator
kubectl get virtualservices -A -l app.kubernetes.io/managed-by=virtualservice-operator -o yaml > virtualservices-backup.yaml

# Backup operator deployment
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o yaml > operator-deployment-backup.yaml
```

### Recovery Procedures
```bash
# Restore operator configuration
kubectl apply -f operator-config-backup.yaml

# Restore operator deployment
kubectl apply -f operator-deployment-backup.yaml

# Force reconciliation of all services
kubectl get services -A -o name | xargs -I {} kubectl annotate {} reconcile.virtualservice-operator/trigger="$(date)"

# Verify recovery
kubectl get virtualservices -A -l app.kubernetes.io/managed-by=virtualservice-operator
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

## Troubleshooting Deployment Issues

### Common Deployment Problems

#### 1. Operator Pod Not Starting
```bash
# Check pod status
kubectl get pods -n virtualservice-operator-system

# Check events
kubectl get events -n virtualservice-operator-system --sort-by='.lastTimestamp'

# Check pod logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator

# Common causes:
# - Image pull errors
# - RBAC permissions
# - ConfigMap not found
# - Resource constraints
```

#### 2. RBAC Permission Issues
```bash
# Test permissions
kubectl auth can-i create virtualservices --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator
kubectl auth can-i get services --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator

# Check ClusterRoleBinding
kubectl get clusterrolebinding virtualservice-operator-binding -o yaml

# Fix permissions if needed
kubectl apply -f deployments/deployment.yaml
```

#### 3. ConfigMap Issues
```bash
# Check ConfigMap exists
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system

# Validate ConfigMap content
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml

# Check operator logs for configuration errors
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i config
```

#### 4. VirtualServices Not Created
```bash
# Check if services exist in monitored namespaces
kubectl get services -n production
kubectl get services -n dev-team-1

# Check operator logs for service processing
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep "Processing service"

# Force reconciliation
kubectl annotate service my-service reconcile.virtualservice-operator/trigger="$(date)" -n production
```

### Performance Troubleshooting
```bash
# Check resource usage
kubectl top pod -n virtualservice-operator-system

# Check for memory leaks
kubectl exec -n virtualservice-operator-system deployment/virtualservice-operator -- ps aux

# Enable profiling (if built with pprof)
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 6060:6060
go tool pprof http://localhost:6060/debug/pprof/heap
```

### Network Troubleshooting
```bash
# Check if operator can reach Kubernetes API
kubectl exec -n virtualservice-operator-system deployment/virtualservice-operator -- nslookup kubernetes.default.svc.cluster.local

# Check Istio connectivity
kubectl exec -n virtualservice-operator-system deployment/virtualservice-operator -- nslookup istiod.istio-system.svc.cluster.local

# Test VirtualService creation manually
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: test-vs
  namespace: default
spec:
  hosts:
  - test-service
  http:
  - route:
    - destination:
        host: test-service
EOF
```

## Upgrade Procedures

### Pre-Upgrade Checklist
```bash
# Backup current configuration
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml > config-backup.yaml

# Backup current deployment
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o yaml > deployment-backup.yaml

# Check current version
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Verify cluster health
kubectl get nodes
kubectl get pods -n virtualservice-operator-system
```

### Upgrade Process
```bash
# Update to new version
kubectl set image deployment/virtualservice-operator \
  manager=harbor.intent.ai/library/virtualservice-operator:v1.4.0 \
  -n virtualservice-operator-system

# Monitor rollout
kubectl rollout status deployment/virtualservice-operator -n virtualservice-operator-system

# Verify new version
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check logs for any issues
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

### Post-Upgrade Verification
```bash
# Test operator functionality
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: upgrade-test-service
  namespace: default
spec:
  selector:
    app: upgrade-test
  ports:
  - port: 80
EOF

# Verify VirtualService creation
kubectl get virtualservice upgrade-test-service-virtual-service -o yaml

# Clean up test resources
kubectl delete service upgrade-test-service
kubectl delete virtualservice upgrade-test-service-virtual-service
```

### Rollback Procedures
```bash
# Rollback to previous version
kubectl rollout undo deployment/virtualservice-operator -n virtualservice-operator-system

# Or rollback to specific revision
kubectl rollout history deployment/virtualservice-operator -n virtualservice-operator-system
kubectl rollout undo deployment/virtualservice-operator --to-revision=2 -n virtualservice-operator-system

# Verify rollback
kubectl rollout status deployment/virtualservice-operator -n virtualservice-operator-system
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

## Security Considerations

### Pod Security Standards
```yaml
# pod-security-policy.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: virtualservice-operator-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### Network Policies
```yaml
# operator-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: virtualservice-operator-netpol
  namespace: virtualservice-operator-system
spec:
  podSelector:
    matchLabels:
      app: virtualservice-operator
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080  # metrics
    - protocol: TCP
      port: 8081  # health
  egress:
  - to: []  # Allow all egress for Kubernetes API access
```

### Image Security
```bash
# Scan images for vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image harbor.intent.ai/library/virtualservice-operator:v1.3.0

# Use specific image digests in production
kubectl patch deployment virtualservice-operator \
  -n virtualservice-operator-system \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "harbor.intent.ai/library/virtualservice-operator@sha256:abc123..."}]'
```

This comprehensive deployment guide covers all aspects of deploying the VirtualService Operator from development to production environments, including configuration management, monitoring, troubleshooting, and security considerations.