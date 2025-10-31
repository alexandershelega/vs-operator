# VirtualService Operator - API Reference

## Overview

This document provides comprehensive API reference for the VirtualService Operator, including all key functions, structs, interfaces, and configuration options.

## Core Structures

### OperatorConfig

The main configuration structure for the operator.

```go
type OperatorConfig struct {
    DefaultNamespace          string   `yaml:"defaultNamespace"`
    DeveloperNamespaces      []string `yaml:"developerNamespaces"`
    EnablePlaceholderServices bool     `yaml:"enablePlaceholderServices"`
    VirtualServiceTemplate   string   `yaml:"virtualServiceTemplate"`
}
```

**Fields:**
- `DefaultNamespace`: The primary namespace where production services reside
- `DeveloperNamespaces`: List of developer/staging namespaces to monitor
- `EnablePlaceholderServices`: Feature flag to enable automatic placeholder service creation
- `VirtualServiceTemplate`: Template for generating VirtualServices

### ServiceReconciler

Main controller struct that handles service reconciliation.

```go
type ServiceReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Config *config.OperatorConfig
}
```

## Core Functions

### Service Controller Functions

#### Reconcile
```go
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```
Main reconciliation loop that handles service events.

**Parameters:**
- `ctx`: Request context
- `req`: Reconciliation request containing namespace and name

**Returns:**
- `ctrl.Result`: Reconciliation result with requeue information
- `error`: Any error that occurred during reconciliation

#### handleDefaultNamespaceService
```go
func (r *ServiceReconciler) handleDefaultNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error)
```
Handles services created/updated in the default namespace.

**Parameters:**
- `ctx`: Request context
- `service`: The service object from default namespace
- `config`: Operator configuration

**Behavior:**
- Creates/updates VirtualService with routes to developer namespaces
- Creates placeholder services if feature is enabled
- Skips system services automatically

#### handleDeveloperNamespaceService
```go
func (r *ServiceReconciler) handleDeveloperNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error)
```
Handles services created/updated in developer namespaces.

**Parameters:**
- `ctx`: Request context
- `service`: The service object from developer namespace
- `config`: Operator configuration

**Behavior:**
- Adds developer route to existing VirtualService
- Skips placeholder services
- Updates VirtualService with retry logic

#### handleServiceDeletion
```go
func (r *ServiceReconciler) handleServiceDeletion(ctx context.Context, serviceName, namespace string, config *config.OperatorConfig) (ctrl.Result, error)
```
Handles cleanup when services are deleted.

**Parameters:**
- `ctx`: Request context
- `serviceName`: Name of the deleted service
- `namespace`: Namespace where service was deleted
- `config`: Operator configuration

**Behavior:**
- Removes routes from VirtualService
- Cleans up placeholder services
- Deletes entire VirtualService if no routes remain

### Placeholder Service Management

#### createPlaceholderServices
```go
func (r *ServiceReconciler) createPlaceholderServices(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) error
```
Creates ExternalName services in developer namespaces.

**Parameters:**
- `ctx`: Request context
- `service`: Source service from default namespace
- `config`: Operator configuration

**Behavior:**
- Creates ExternalName services pointing to default namespace service
- Skips creation if service already exists
- Sets appropriate labels and annotations

#### createSinglePlaceholderService
```go
func (r *ServiceReconciler) createSinglePlaceholderService(ctx context.Context, service *corev1.Service, targetNamespace, sourceNamespace string) error
```
Creates a single placeholder service in specified namespace.

#### ensurePlaceholderServicesForNamespace
```go
func (r *ServiceReconciler) ensurePlaceholderServicesForNamespace(ctx context.Context, targetNamespace, sourceNamespace string, config *config.OperatorConfig) error
```
Ensures all services from source namespace have placeholders in target namespace.

#### deletePlaceholderServices
```go
func (r *ServiceReconciler) deletePlaceholderServices(ctx context.Context, serviceName string, config *config.OperatorConfig) error
```
Deletes placeholder services when source service is removed.

### Utility Functions

#### isPlaceholderService
```go
func (r *ServiceReconciler) isPlaceholderService(service *corev1.Service) bool
```
Checks if a service is a placeholder service created by the operator.

**Detection Logic:**
- Checks for annotation: `virtualservice-operator/placeholder-service: "true"`
- Validates service type is ExternalName
- Verifies external name pattern matches expected format

#### isSystemService
```go
func (r *ServiceReconciler) isSystemService(serviceName string) bool
```
Determines if a service should be excluded from processing.

**Excluded Services:**
- kubernetes
- kube-dns
- Any service starting with "kube-"
- Any service starting with "istio-"

#### retryVirtualServiceUpdate
```go
func (r *ServiceReconciler) retryVirtualServiceUpdate(ctx context.Context, updateFunc func() error) error
```
Implements retry logic with exponential backoff for VirtualService updates.

**Retry Configuration:**
- Maximum retries: 5
- Base delay: 1 second
- Exponential backoff: 2x multiplier
- Maximum delay: 16 seconds

## VirtualService Utilities

### GenerateVirtualService
```go
func GenerateVirtualService(serviceName, defaultNamespace string, developerNamespaces []string, template string, existingServices map[string]bool) (*v1beta1.VirtualService, error)
```
Generates a VirtualService based on template and existing services.

**Parameters:**
- `serviceName`: Name of the service
- `defaultNamespace`: Default namespace
- `developerNamespaces`: List of developer namespaces
- `template`: VirtualService template
- `existingServices`: Map of existing services by namespace

**Returns:**
- `*v1beta1.VirtualService`: Generated VirtualService object
- `error`: Any error during generation

### UpdateVirtualServiceRoutes
```go
func UpdateVirtualServiceRoutes(vs *v1beta1.VirtualService, serviceName, defaultNamespace string, developerNamespaces []string, existingServices map[string]bool) error
```
Updates routes in an existing VirtualService.

**Behavior:**
- Preserves existing non-managed routes
- Updates developer routes based on existing services
- Maintains default route as fallback

## Configuration Management

### LoadConfig
```go
func LoadConfig(configMapName, configMapNamespace string, client client.Client) (*OperatorConfig, error)
```
Loads operator configuration from ConfigMap.

**Parameters:**
- `configMapName`: Name of the ConfigMap
- `configMapNamespace`: Namespace of the ConfigMap
- `client`: Kubernetes client

**Returns:**
- `*OperatorConfig`: Loaded configuration
- `error`: Any error during loading

### ValidateConfig
```go
func ValidateConfig(config *OperatorConfig) error
```
Validates operator configuration.

**Validation Rules:**
- DefaultNamespace must not be empty
- DeveloperNamespaces must not contain DefaultNamespace
- VirtualServiceTemplate must be valid YAML

## Constants

### Labels and Annotations
```go
const (
    // Placeholder service identification
    PlaceholderServiceAnnotation = "virtualservice-operator/placeholder-service"
    SourceServiceAnnotation     = "virtualservice-operator/source-service"
    
    // VirtualService management
    ManagedByLabel = "managed-by"
    OperatorName   = "virtualservice-operator"
    
    // Versioning
    VersionAnnotation = "virtualservice-operator/version"
    CurrentVersion    = "v1.3.0"
)
```

### System Namespaces
```go
var SystemNamespaces = []string{
    "kube-system",
    "kube-public", 
    "kube-node-lease",
    "istio-system",
    "istio-injection",
}
```

## Error Types

### Common Errors
- `ErrServiceNotFound`: Service not found in specified namespace
- `ErrVirtualServiceConflict`: Conflict during VirtualService update
- `ErrInvalidConfiguration`: Invalid operator configuration
- `ErrTemplateProcessing`: Error processing VirtualService template

## Metrics and Observability

### Exposed Metrics
The operator exposes standard controller-runtime metrics on `:8080/metrics`:

- `controller_runtime_reconcile_total`: Total number of reconciliations
- `controller_runtime_reconcile_errors_total`: Total number of reconciliation errors
- `workqueue_adds_total`: Total number of items added to work queue
- `workqueue_depth`: Current depth of work queue

### Health Endpoints
- `/healthz`: Liveness probe endpoint
- `/readyz`: Readiness probe endpoint

## Event Types

### Service Events
- `ServiceCreated`: New service detected
- `ServiceUpdated`: Existing service modified
- `ServiceDeleted`: Service removed

### VirtualService Events
- `VirtualServiceCreated`: New VirtualService created
- `VirtualServiceUpdated`: Existing VirtualService modified
- `VirtualServiceDeleted`: VirtualService removed

### Placeholder Service Events
- `PlaceholderServiceCreated`: New placeholder service created
- `PlaceholderServiceDeleted`: Placeholder service removed