# VirtualService Operator - Operations Guide

## Overview

This guide provides comprehensive operational procedures for deploying, monitoring, maintaining, and troubleshooting the VirtualService Operator in production environments.

## Deployment

### Prerequisites Checklist

Before deploying the operator, ensure the following prerequisites are met:

- [ ] Kubernetes cluster version 1.25 or higher
- [ ] Istio service mesh installed and configured
- [ ] `kubectl` configured with cluster admin access
- [ ] Container registry access to `harbor.intent.ai/library/virtualservice-operator`
- [ ] Sufficient RBAC permissions for operator deployment

### Production Deployment

#### 1. Namespace Preparation
```bash
# Create operator namespace
kubectl create namespace virtualservice-operator-system

# Label namespace for monitoring (optional)
kubectl label namespace virtualservice-operator-system monitoring=enabled
```

#### 2. Configuration Setup
```bash
# Create ConfigMap with your environment-specific settings
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "staging"
      - "dev-team-a"
      - "dev-team-b"
    enablePlaceholderServices: true
    virtualServiceTemplate: |
      apiVersion: networking.istio.io/v1beta1
      kind: VirtualService
      metadata:
        name: "\${serviceName}-virtual-service"
        namespace: "\${defaultNamespace}"
        labels:
          managed-by: virtualservice-operator
      spec:
        hosts:
        - \${serviceName}
        http:
        - route:
          - destination:
              host: \${serviceName}.\${defaultNamespace}.svc.cluster.local
EOF
```

#### 3. Deploy Operator
```bash
# Deploy the operator
kubectl apply -f deployments/deployment.yaml

# Verify deployment
kubectl get pods -n virtualservice-operator-system
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

#### 4. Verify Installation
```bash
# Check operator status
kubectl get deployment virtualservice-operator -n virtualservice-operator-system

# Verify RBAC permissions
kubectl auth can-i create virtualservices --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator

# Test configuration loading
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i config
```

### Multi-Environment Deployment

#### Staging Environment
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
      - "dev-integration"
      - "qa-testing"
    enablePlaceholderServices: true
```

#### Production Environment
```yaml
# production-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: virtualservice-operator-config
  namespace: virtualservice-operator-system
data:
  config.yaml: |
    defaultNamespace: "production"
    developerNamespaces:
      - "staging"
      - "canary"
    enablePlaceholderServices: false  # Disable in production for security
```

## Monitoring

### Health Checks

#### Liveness and Readiness Probes
The operator exposes health endpoints that are automatically monitored by Kubernetes:

```bash
# Check health endpoints manually
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8081:8081

# Liveness probe
curl http://localhost:8081/healthz

# Readiness probe  
curl http://localhost:8081/readyz
```

#### Health Check Troubleshooting
```bash
# If health checks fail, check logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator --tail=100

# Check resource usage
kubectl top pod -n virtualservice-operator-system

# Verify ConfigMap accessibility
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system
```

### Metrics and Observability

#### Prometheus Metrics
The operator exposes metrics on port 8080:

```bash
# Access metrics endpoint
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8080:8080
curl http://localhost:8080/metrics
```

#### Key Metrics to Monitor

**Controller Metrics:**
- `controller_runtime_reconcile_total`: Total reconciliations
- `controller_runtime_reconcile_errors_total`: Reconciliation errors
- `controller_runtime_reconcile_time_seconds`: Reconciliation duration

**Work Queue Metrics:**
- `workqueue_adds_total`: Items added to queue
- `workqueue_depth`: Current queue depth
- `workqueue_longest_running_processor_seconds`: Longest running processor

**Resource Metrics:**
- `process_resident_memory_bytes`: Memory usage
- `process_cpu_seconds_total`: CPU usage
- `go_goroutines`: Number of goroutines

#### Grafana Dashboard Example
```json
{
  "dashboard": {
    "title": "VirtualService Operator",
    "panels": [
      {
        "title": "Reconciliation Rate",
        "targets": [
          {
            "expr": "rate(controller_runtime_reconcile_total[5m])"
          }
        ]
      },
      {
        "title": "Error Rate", 
        "targets": [
          {
            "expr": "rate(controller_runtime_reconcile_errors_total[5m])"
          }
        ]
      }
    ]
  }
}
```

### Logging

#### Log Levels
Configure logging verbosity in deployment:

```yaml
# In deployment.yaml container args
args:
- -zap-log-level=info  # Options: debug, info, error
- -zap-development=false  # Set to true for development
```

#### Log Analysis
```bash
# View recent logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator --tail=100

# Follow logs in real-time
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator -f

# Filter for errors
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i error

# Filter for specific service
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep "my-service"
```

#### Structured Log Fields
The operator uses structured logging with these key fields:
- `timestamp`: Event timestamp
- `level`: Log level (info, error, debug)
- `msg`: Log message
- `service`: Service name being processed
- `namespace`: Namespace of the service
- `virtualservice`: VirtualService name
- `error`: Error details (if applicable)

## Maintenance

### Configuration Updates

#### Hot Configuration Reload
The operator automatically reloads configuration when the ConfigMap is updated:

```bash
# Update ConfigMap
kubectl edit configmap virtualservice-operator-config -n virtualservice-operator-system

# Verify reload in logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i "config reloaded"
```

#### Configuration Validation
```bash
# Validate configuration before applying
kubectl apply --dry-run=client -f new-config.yaml

# Test configuration with temporary ConfigMap
kubectl create configmap test-config --from-file=config.yaml=new-config.yaml -n virtualservice-operator-system --dry-run=client -o yaml
```

### Operator Updates

#### Rolling Update Process
```bash
# Update operator image
kubectl set image deployment/virtualservice-operator manager=harbor.intent.ai/library/virtualservice-operator:v1.3.1 -n virtualservice-operator-system

# Monitor rollout
kubectl rollout status deployment/virtualservice-operator -n virtualservice-operator-system

# Verify new version
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o jsonpath='{.spec.template.spec.containers[0].image}'
```

#### Rollback Procedure
```bash
# Check rollout history
kubectl rollout history deployment/virtualservice-operator -n virtualservice-operator-system

# Rollback to previous version
kubectl rollout undo deployment/virtualservice-operator -n virtualservice-operator-system

# Rollback to specific revision
kubectl rollout undo deployment/virtualservice-operator --to-revision=2 -n virtualservice-operator-system
```

### Backup and Recovery

#### Configuration Backup
```bash
# Backup ConfigMap
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml > config-backup.yaml

# Backup deployment
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o yaml > deployment-backup.yaml

# Backup RBAC
kubectl get clusterrole virtualservice-operator-role -o yaml > rbac-backup.yaml
kubectl get clusterrolebinding virtualservice-operator-rolebinding -o yaml >> rbac-backup.yaml
```

#### Disaster Recovery
```bash
# Restore from backup
kubectl apply -f config-backup.yaml
kubectl apply -f deployment-backup.yaml
kubectl apply -f rbac-backup.yaml

# Verify restoration
kubectl get pods -n virtualservice-operator-system
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

### Resource Management

#### Resource Monitoring
```bash
# Check resource usage
kubectl top pod -n virtualservice-operator-system

# Check resource limits
kubectl describe deployment virtualservice-operator -n virtualservice-operator-system | grep -A 10 "Limits\|Requests"

# Monitor resource trends
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/virtualservice-operator-system/pods | jq '.items[].containers[].usage'
```

#### Resource Optimization
```yaml
# Adjust resource limits based on monitoring
resources:
  limits:
    cpu: 500m      # Adjust based on CPU usage patterns
    memory: 256Mi  # Adjust based on memory usage patterns
  requests:
    cpu: 50m       # Minimum guaranteed CPU
    memory: 128Mi  # Minimum guaranteed memory
```

## Troubleshooting

### Common Issues

#### 1. Operator Not Starting
**Symptoms:**
- Pod in CrashLoopBackOff state
- Health checks failing

**Diagnosis:**
```bash
# Check pod status
kubectl get pods -n virtualservice-operator-system

# Check events
kubectl get events -n virtualservice-operator-system --sort-by='.lastTimestamp'

# Check logs
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

**Common Causes:**
- Invalid ConfigMap configuration
- RBAC permission issues
- Resource constraints
- Image pull failures

**Solutions:**
```bash
# Fix ConfigMap syntax
kubectl edit configmap virtualservice-operator-config -n virtualservice-operator-system

# Verify RBAC
kubectl auth can-i create virtualservices --as=system:serviceaccount:virtualservice-operator-system:virtualservice-operator

# Check resource availability
kubectl describe node | grep -A 5 "Allocated resources"
```

#### 2. VirtualServices Not Created
**Symptoms:**
- Services exist but no VirtualServices created
- No routes in existing VirtualServices

**Diagnosis:**
```bash
# Check if services are in monitored namespaces
kubectl get services -A | grep my-service

# Check operator logs for service processing
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep my-service

# Verify service is not a system service
kubectl get service my-service -o yaml | grep labels
```

**Solutions:**
```bash
# Force reconciliation by restarting operator
kubectl rollout restart deployment/virtualservice-operator -n virtualservice-operator-system

# Check service labels for exclusion
kubectl label service my-service app.kubernetes.io/name=my-service

# Verify namespace configuration
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml
```

#### 3. Placeholder Services Issues
**Symptoms:**
- Placeholder services not created
- Routes created for placeholder services
- Placeholder services not cleaned up

**Diagnosis:**
```bash
# Check if feature is enabled
kubectl get configmap virtualservice-operator-config -n virtualservice-operator-system -o yaml | grep enablePlaceholderServices

# List placeholder services
kubectl get services -A -l "virtualservice-operator/placeholder-service=true"

# Check placeholder service annotations
kubectl get service my-service -n dev-namespace -o yaml | grep annotations -A 5
```

**Solutions:**
```bash
# Enable placeholder services
kubectl patch configmap virtualservice-operator-config -n virtualservice-operator-system --patch '{"data":{"config.yaml":"defaultNamespace: \"default\"\ndeveloperNamespaces:\n  - \"dev\"\nenablePlaceholderServices: true\nvirtualServiceTemplate: |..."}}'

# Clean up orphaned placeholder services
kubectl delete services -l "virtualservice-operator/placeholder-service=true" -A

# Force placeholder service recreation
kubectl rollout restart deployment/virtualservice-operator -n virtualservice-operator-system
```

#### 4. VirtualService Conflicts
**Symptoms:**
- Conflict errors in logs
- VirtualServices not updating
- Retry exhaustion messages

**Diagnosis:**
```bash
# Check for conflict errors
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i conflict

# Check VirtualService resource version
kubectl get virtualservice my-service-virtual-service -o yaml | grep resourceVersion

# Check for multiple operator instances
kubectl get pods -n virtualservice-operator-system
```

**Solutions:**
```bash
# Ensure only one operator instance
kubectl scale deployment virtualservice-operator --replicas=1 -n virtualservice-operator-system

# Manually resolve conflicts by deleting and recreating VirtualService
kubectl delete virtualservice my-service-virtual-service
# Operator will recreate it automatically

# Check leader election
kubectl get lease -n virtualservice-operator-system
```

### Performance Troubleshooting

#### High CPU Usage
```bash
# Check CPU metrics
kubectl top pod -n virtualservice-operator-system

# Profile CPU usage
kubectl exec -n virtualservice-operator-system deployment/virtualservice-operator -- curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze with go tool pprof
go tool pprof cpu.prof
```

#### High Memory Usage
```bash
# Check memory metrics
kubectl top pod -n virtualservice-operator-system

# Profile memory usage
kubectl exec -n virtualservice-operator-system deployment/virtualservice-operator -- curl http://localhost:8080/debug/pprof/heap > heap.prof

# Check for memory leaks
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep -i "out of memory\|oom"
```

#### Slow Reconciliation
```bash
# Check reconciliation metrics
kubectl port-forward -n virtualservice-operator-system deployment/virtualservice-operator 8080:8080
curl http://localhost:8080/metrics | grep reconcile_time

# Check work queue depth
curl http://localhost:8080/metrics | grep workqueue_depth

# Enable debug logging
kubectl patch deployment virtualservice-operator -n virtualservice-operator-system --patch '{"spec":{"template":{"spec":{"containers":[{"name":"manager","args":["-zap-log-level=debug"]}]}}}}'
```

### Emergency Procedures

#### Complete Operator Shutdown
```bash
# Scale down operator
kubectl scale deployment virtualservice-operator --replicas=0 -n virtualservice-operator-system

# Verify shutdown
kubectl get pods -n virtualservice-operator-system
```

#### Emergency VirtualService Cleanup
```bash
# List all managed VirtualServices
kubectl get virtualservices -A -l managed-by=virtualservice-operator

# Remove operator management (allows manual management)
kubectl label virtualservices -A managed-by- -l managed-by=virtualservice-operator

# Delete all managed VirtualServices
kubectl delete virtualservices -A -l managed-by=virtualservice-operator
```

#### Recovery from Corruption
```bash
# Delete operator deployment
kubectl delete deployment virtualservice-operator -n virtualservice-operator-system

# Clean up resources
kubectl delete virtualservices -A -l managed-by=virtualservice-operator
kubectl delete services -A -l "virtualservice-operator/placeholder-service=true"

# Redeploy operator
kubectl apply -f deployments/deployment.yaml

# Verify recovery
kubectl get pods -n virtualservice-operator-system
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator
```

## Security Considerations

### RBAC Review
Regularly review and audit RBAC permissions:

```bash
# Review current permissions
kubectl describe clusterrole virtualservice-operator-role

# Audit service account usage
kubectl get rolebindings,clusterrolebindings -A -o wide | grep virtualservice-operator
```

### Network Policies
Consider implementing network policies to restrict operator traffic:

```yaml
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
  egress:
  - to: []  # Allow all egress for Kubernetes API access
  ingress:
  - from: []  # Allow health check access
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8081
```

### Security Scanning
```bash
# Scan operator image for vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image harbor.intent.ai/library/virtualservice-operator:v1.3.0

# Check for security updates
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o yaml | grep image:
```