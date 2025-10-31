# Placeholder Service Management - Implementation Guide

## 1. Overview

This document provides detailed implementation guidance for adding automatic placeholder service management to the VirtualService operator. The implementation will extend the existing controller to automatically create and manage ExternalName services in developer namespaces.

## 2. Configuration Changes

### 2.1 Update OperatorConfig Structure

**File: `internal/config/config.go`**

Add the new configuration field:

```go
type OperatorConfig struct {
    DefaultNamespace          string   `yaml:"defaultNamespace"`
    DeveloperNamespaces      []string `yaml:"developerNamespaces"`
    EnablePlaceholderServices bool     `yaml:"enablePlaceholderServices"` // New field
    VirtualServiceTemplate   string   `yaml:"virtualServiceTemplate"`
}
```

### 2.2 Update ConfigMap

**File: `deployments/deployment.yaml`**

Add the new configuration option to the ConfigMap:

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
      - "default-test"
      - "default-test2"
      - "default-staging"
    enablePlaceholderServices: true  # New configuration
    virtualServiceTemplate: |
      # ... existing template content
```

## 3. Core Implementation Changes

### 3.1 Service Controller Enhancements

**File: `controllers/service_controller.go`**

#### 3.1.1 Add Placeholder Service Constants

```go
const (
    PlaceholderServiceLabel = "placeholder-service"
    ManagedByLabel         = "managed-by"
    SourceNamespaceLabel   = "source-namespace"
    SourceServiceLabel     = "source-service"
    
    SourceServiceAnnotation = "virtualservice-operator/source-service"
    CreatedAtAnnotation    = "virtualservice-operator/created-at"
    VersionAnnotation      = "virtualservice-operator/version"
    
    OperatorName = "virtualservice-operator"
    CurrentVersion = "v1.2.4"
)
```

#### 3.1.2 Add Placeholder Service Detection

```go
// isPlaceholderService checks if a service is a placeholder service created by the operator
func (r *ServiceReconciler) isPlaceholderService(service *corev1.Service) bool {
    if service.Labels == nil {
        return false
    }
    
    placeholderLabel, exists := service.Labels[PlaceholderServiceLabel]
    if !exists {
        return false
    }
    
    managedByLabel, managedExists := service.Labels[ManagedByLabel]
    return placeholderLabel == "true" && managedExists && managedByLabel == OperatorName
}
```

#### 3.1.3 Modify handleDefaultNamespaceService

```go
// handleDefaultNamespaceService creates or updates VirtualService for services in the default namespace
func (r *ServiceReconciler) handleDefaultNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error) {
    // Skip system services
    if r.isSystemService(service.Name) {
        return ctrl.Result{}, nil
    }

    // Create placeholder services if feature is enabled
    if config.EnablePlaceholderServices {
        if err := r.createPlaceholderServices(ctx, service, config); err != nil {
            return ctrl.Result{}, fmt.Errorf("failed to create placeholder services: %w", err)
        }
    }

    // Continue with existing VirtualService logic...
    // [Rest of existing implementation remains the same]
}
```

#### 3.1.4 Modify handleDeveloperNamespaceService

```go
// handleDeveloperNamespaceService updates existing VirtualService for services in developer namespaces
func (r *ServiceReconciler) handleDeveloperNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error) {
    // Skip placeholder services - they should not create VirtualService routes
    if r.isPlaceholderService(service) {
        return ctrl.Result{}, nil
    }

    // Continue with existing logic for regular services...
    // [Rest of existing implementation remains the same]
}
```

#### 3.1.5 Modify handleServiceDeletion

```go
// handleServiceDeletion handles cleanup when a service is deleted
func (r *ServiceReconciler) handleServiceDeletion(ctx context.Context, serviceName, namespace string, config *config.OperatorConfig) (ctrl.Result, error) {
    // If service was deleted from default namespace, cleanup placeholder services
    if namespace == config.DefaultNamespace && config.EnablePlaceholderServices {
        if err := r.cleanupPlaceholderServices(ctx, serviceName, config); err != nil {
            return ctrl.Result{}, fmt.Errorf("failed to cleanup placeholder services: %w", err)
        }
    }

    // Continue with existing VirtualService cleanup logic...
    // [Rest of existing implementation remains the same]
}
```

### 3.2 New Placeholder Service Management Functions

**File: `controllers/service_controller.go`**

#### 3.2.1 Create Placeholder Services

```go
// createPlaceholderServices creates ExternalName services in developer namespaces
func (r *ServiceReconciler) createPlaceholderServices(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) error {
    for _, devNamespace := range config.DeveloperNamespaces {
        if devNamespace == config.DefaultNamespace {
            continue // Skip if developer namespace is same as default
        }
        
        // Check if placeholder service already exists
        existingService := &corev1.Service{}
        err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: devNamespace}, existingService)
        if err == nil {
            // Service already exists, check if it's our placeholder
            if r.isPlaceholderService(existingService) {
                // Our placeholder already exists, skip
                continue
            } else {
                // Different service exists, skip to avoid conflicts
                continue
            }
        } else if !errors.IsNotFound(err) {
            return fmt.Errorf("failed to check existing service %s in namespace %s: %w", service.Name, devNamespace, err)
        }
        
        // Create placeholder service
        placeholderService := r.generatePlaceholderService(service, devNamespace, config.DefaultNamespace)
        if err := r.Create(ctx, placeholderService); err != nil {
            return fmt.Errorf("failed to create placeholder service %s in namespace %s: %w", service.Name, devNamespace, err)
        }
    }
    
    return nil
}
```

#### 3.2.2 Generate Placeholder Service

```go
// generatePlaceholderService creates a placeholder ExternalName service
func (r *ServiceReconciler) generatePlaceholderService(sourceService *corev1.Service, targetNamespace, sourceNamespace string) *corev1.Service {
    externalName := fmt.Sprintf("%s.%s.svc.cluster.local", sourceService.Name, sourceNamespace)
    
    return &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      sourceService.Name,
            Namespace: targetNamespace,
            Labels: map[string]string{
                ManagedByLabel:       OperatorName,
                PlaceholderServiceLabel: "true",
                SourceNamespaceLabel: sourceNamespace,
                SourceServiceLabel:   sourceService.Name,
            },
            Annotations: map[string]string{
                SourceServiceAnnotation: externalName,
                CreatedAtAnnotation:    time.Now().Format(time.RFC3339),
                VersionAnnotation:      CurrentVersion,
            },
        },
        Spec: corev1.ServiceSpec{
            Type:         corev1.ServiceTypeExternalName,
            ExternalName: externalName,
        },
    }
}
```

#### 3.2.3 Cleanup Placeholder Services

```go
// cleanupPlaceholderServices removes placeholder services when source service is deleted
func (r *ServiceReconciler) cleanupPlaceholderServices(ctx context.Context, serviceName string, config *config.OperatorConfig) error {
    for _, devNamespace := range config.DeveloperNamespaces {
        if devNamespace == config.DefaultNamespace {
            continue
        }
        
        // Get the service in developer namespace
        service := &corev1.Service{}
        err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: devNamespace}, service)
        if err != nil {
            if errors.IsNotFound(err) {
                // Service doesn't exist, nothing to cleanup
                continue
            }
            return fmt.Errorf("failed to get service %s in namespace %s: %w", serviceName, devNamespace, err)
        }
        
        // Only delete if it's our placeholder service
        if r.isPlaceholderService(service) {
            if err := r.Delete(ctx, service); err != nil {
                return fmt.Errorf("failed to delete placeholder service %s in namespace %s: %w", serviceName, devNamespace, err)
            }
        }
    }
    
    return nil
}
```

## 4. RBAC Updates

**File: `deployments/deployment.yaml`**

The existing RBAC permissions should be sufficient since the operator already has permissions to manage services. However, verify the ClusterRole includes:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: virtualservice-operator
rules:
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
# ... other existing rules
```

## 5. Testing Strategy

### 5.1 Unit Tests

Create test cases for:
- `isPlaceholderService()` function
- `generatePlaceholderService()` function
- `createPlaceholderServices()` function
- `cleanupPlaceholderServices()` function

### 5.2 Integration Tests

Test scenarios:
1. Service creation in default namespace creates placeholder services
2. Service deletion in default namespace removes placeholder services
3. Existing services in developer namespaces are not overwritten
4. Placeholder services don't create VirtualService routes
5. Feature can be disabled via configuration

### 5.3 Manual Testing

1. Deploy operator with `enablePlaceholderServices: true`
2. Create a service in default namespace
3. Verify placeholder services are created in developer namespaces
4. Verify VirtualService routes are not created for placeholder services
5. Delete service from default namespace
6. Verify placeholder services are cleaned up

## 6. Deployment Steps

1. **Update Configuration**: Modify the ConfigMap to include `enablePlaceholderServices: true`
2. **Build New Image**: Build and push new operator image with version `v1.2.4`
3. **Update Deployment**: Update deployment.yaml to use new image tag
4. **Deploy**: Apply the updated deployment
5. **Verify**: Test the functionality with sample services

## 7. Monitoring and Observability

### 7.1 Logging

Add structured logging for:
- Placeholder service creation events
- Placeholder service deletion events
- Configuration changes
- Error conditions

### 7.2 Metrics (Future Enhancement)

Consider adding Prometheus metrics for:
- Number of placeholder services created
- Number of placeholder services deleted
- Configuration status
- Error rates

## 8. Rollback Strategy

If issues occur:
1. Set `enablePlaceholderServices: false` in ConfigMap
2. Manually clean up existing placeholder services if needed
3. Rollback to previous operator version if necessary

The feature is designed to be backward compatible and can be disabled without affecting existing functionality.