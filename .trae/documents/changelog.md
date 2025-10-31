# VirtualService Operator - Changelog

## Overview

This document tracks all notable changes to the VirtualService Operator project. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation suite including API reference, operations guide, security guide, development guide, deployment guide, configuration reference, and architecture deep dive
- Enhanced monitoring and observability features
- Performance optimization improvements

### Changed
- Improved error handling and logging throughout the codebase
- Enhanced configuration validation and error reporting

### Security
- Added security hardening guidelines and best practices
- Implemented comprehensive RBAC documentation

## [1.3.0] - 2024-01-15

### Fixed
- **CRITICAL**: Fixed fundamental logic error in `handleServiceDeletion` function that was incorrectly keeping routes in VirtualService when placeholder services were created
- Routes are now always removed from VirtualService when real services are deleted from developer namespaces, regardless of placeholder creation
- Placeholder services no longer get routes in VirtualService, ensuring proper traffic routing

### Changed
- Completely rewritten `handleServiceDeletion` logic to separate placeholder service creation from route management
- Added comprehensive debug logging for service deletion and placeholder creation processes
- Improved error handling and status reporting in service deletion scenarios

### Technical Details
- Modified `handleServiceDeletion` function in `controllers/service_controller.go` (lines 445-530)
- Removed `placeholderCreated` logic that was preventing proper route removal
- Enhanced logging to track service deletion, route removal, and placeholder creation separately
- Ensured placeholder services are created with proper annotations but without VirtualService routes

## [1.2.9] - 2024-01-14

### Changed
- Removed unnecessary labels from placeholder service creation functions
- Placeholder services now use only essential annotations for identification
- Cleaned up `createSinglePlaceholderService` and `createPlaceholderServices` functions

### Removed
- Removed old labels from placeholder services:
  - `app.kubernetes.io/managed-by: virtualservice-operator`
  - `virtualservice-operator/type: placeholder`
  - `managed-by: virtualservice-operator`
  - `placeholder-service: true`

### Technical Details
- Placeholder services now use only these annotations:
  - `virtualservice-operator/placeholder-service: "true"`
  - `virtualservice-operator/source-service: <source-service-fqdn>`
- Maintained `ExternalName` service specification for placeholder services

## [1.2.8] - 2024-01-13

### Added
- Enhanced placeholder service detection using annotations instead of labels
- Improved service identification logic for better reliability

### Changed
- Updated `isPlaceholderService` function to use annotation-based detection
- Modified placeholder service creation to use consistent annotation format

### Technical Details
- Placeholder services now identified by annotation: `virtualservice-operator/placeholder-service: "true"`
- Added source service tracking with annotation: `virtualservice-operator/source-service`

## [1.2.7] - 2024-01-12

### Fixed
- Resolved issue with placeholder services being created with incorrect labels
- Fixed service controller logic for handling placeholder service lifecycle

### Added
- Comprehensive placeholder service management functions
- Enhanced error handling for placeholder service operations

### Technical Details
- Added `createPlaceholderServices` function for batch placeholder creation
- Added `generatePlaceholderService` function for individual placeholder generation
- Improved placeholder service cleanup logic

## [1.2.6] - 2024-01-11

### Added
- Automatic placeholder service management feature
- ConfigMap-based configuration for enabling/disabling placeholder services
- Enhanced service deletion handling with placeholder creation

### Changed
- Updated service controller to handle placeholder service creation when services are deleted
- Modified VirtualService management to work with placeholder services
- Enhanced configuration structure to support placeholder service settings

### Technical Details
- Added `enablePlaceholderServices` configuration option
- Implemented placeholder service creation in `handleServiceDeletion`
- Added placeholder service detection and management functions

## [1.2.5] - 2024-01-10

### Fixed
- Resolved VirtualService route conflicts when multiple developer services exist
- Fixed race condition in service reconciliation logic
- Improved error handling for concurrent VirtualService updates

### Added
- Retry logic with exponential backoff for handling conflicts
- Enhanced logging for debugging VirtualService operations
- Conflict resolution mechanism for concurrent modifications

### Technical Details
- Added conflict detection and resolution in VirtualService updates
- Implemented retry mechanism with configurable backoff
- Enhanced error reporting and logging throughout the controller

## [1.2.4] - 2024-01-09

### Changed
- Improved VirtualService template processing and validation
- Enhanced configuration loading and validation logic
- Updated service filtering to exclude system services more effectively

### Added
- Template validation during configuration loading
- Enhanced error messages for configuration issues
- Improved service namespace filtering

### Technical Details
- Added template syntax validation in configuration loader
- Enhanced service filtering logic to exclude Kubernetes system services
- Improved error handling in template processing

## [1.2.3] - 2024-01-08

### Fixed
- Fixed issue with VirtualService not being updated when developer services change
- Resolved template rendering issues with special characters in service names
- Fixed namespace filtering logic for developer services

### Added
- Enhanced template variable validation
- Improved service name sanitization
- Better error reporting for template processing failures

### Technical Details
- Updated template processing to handle edge cases in service names
- Enhanced namespace validation and filtering
- Improved error handling in VirtualService generation

## [1.2.2] - 2024-01-07

### Added
- Support for custom VirtualService templates via ConfigMap
- Dynamic configuration reloading without operator restart
- Enhanced logging and debugging capabilities

### Changed
- Moved from hardcoded VirtualService generation to template-based approach
- Improved configuration management with hot-reload capability
- Enhanced error handling and recovery mechanisms

### Technical Details
- Implemented Go template engine for VirtualService generation
- Added configuration watcher for dynamic updates
- Enhanced logging with structured logging format

## [1.2.1] - 2024-01-06

### Fixed
- Fixed issue with services in system namespaces being processed
- Resolved memory leak in service watching logic
- Fixed VirtualService ownership and garbage collection

### Added
- System namespace filtering to exclude kube-system, istio-system, etc.
- Enhanced garbage collection for orphaned VirtualServices
- Improved resource cleanup on operator shutdown

### Technical Details
- Added system namespace exclusion list
- Implemented proper owner references for VirtualServices
- Enhanced resource cleanup and garbage collection

## [1.2.0] - 2024-01-05

### Added
- Multi-namespace support for developer environments
- Header-based traffic routing using `x-developer` header
- Configurable namespace mapping for development teams
- Support for multiple developer namespaces per service

### Changed
- Enhanced service controller to handle multiple developer namespaces
- Improved VirtualService generation with dynamic routing rules
- Updated configuration structure to support multiple developer namespaces

### Technical Details
- Added support for multiple developer namespaces in configuration
- Implemented header-based routing logic in VirtualService generation
- Enhanced service reconciliation logic for multi-namespace scenarios

## [1.1.2] - 2024-01-04

### Fixed
- Fixed VirtualService creation for services with multiple ports
- Resolved issue with service updates not triggering VirtualService updates
- Fixed resource cleanup when services are deleted

### Added
- Enhanced port handling in VirtualService generation
- Improved service update detection and processing
- Better resource cleanup and garbage collection

### Technical Details
- Updated VirtualService generation to handle multiple service ports
- Enhanced service change detection logic
- Improved resource cleanup on service deletion

## [1.1.1] - 2024-01-03

### Fixed
- Fixed RBAC permissions for VirtualService management
- Resolved issue with operator not starting due to missing permissions
- Fixed service discovery in non-default namespaces

### Added
- Comprehensive RBAC configuration for all required resources
- Enhanced error handling for permission-related issues
- Improved service discovery across namespaces

### Technical Details
- Updated ClusterRole with all necessary permissions
- Enhanced error handling for RBAC-related failures
- Improved service watching across multiple namespaces

## [1.1.0] - 2024-01-02

### Added
- ConfigMap-based configuration management
- Support for custom default and developer namespaces
- Dynamic configuration updates without restart
- Enhanced logging and monitoring capabilities

### Changed
- Moved from environment variable configuration to ConfigMap-based configuration
- Improved operator startup and configuration loading
- Enhanced error handling and recovery

### Technical Details
- Implemented ConfigMap-based configuration loader
- Added configuration validation and error handling
- Enhanced logging with configurable log levels

## [1.0.1] - 2024-01-01

### Fixed
- Fixed issue with VirtualService not being created for existing services
- Resolved startup issues in some Kubernetes environments
- Fixed service controller registration and event handling

### Added
- Enhanced startup logging and error reporting
- Improved service discovery and initial reconciliation
- Better error handling for edge cases

### Technical Details
- Fixed service controller setup and registration
- Enhanced initial reconciliation logic
- Improved error handling and logging

## [1.0.0] - 2023-12-31

### Added
- Initial release of VirtualService Operator
- Automatic VirtualService creation for Kubernetes Services
- Support for Istio service mesh integration
- Basic header-based traffic routing
- Service lifecycle management
- Controller-runtime based implementation

### Features
- **Automatic VirtualService Management**: Creates and manages Istio VirtualServices based on Kubernetes Services
- **Header-Based Routing**: Routes traffic based on `x-developer` header to appropriate developer namespaces
- **Service Lifecycle Tracking**: Automatically creates, updates, and deletes VirtualServices based on Service changes
- **Namespace-Aware**: Supports different behavior for production and developer namespaces
- **Conflict Resolution**: Handles concurrent updates with retry logic
- **Observability**: Provides metrics and health endpoints for monitoring

### Technical Details
- Built with controller-runtime framework
- Supports Kubernetes 1.20+
- Requires Istio service mesh
- Uses Go 1.21+
- Implements standard Kubernetes controller pattern

### Configuration
- Environment variable-based configuration
- Support for custom namespace configuration
- Configurable service filtering and routing rules

### Deployment
- Single deployment with RBAC configuration
- Supports high availability with leader election
- Includes health checks and metrics endpoints
- Container image available at `harbor.intent.ai/library/virtualservice-operator:v1.0.0`

---

## Release Notes Format

Each release follows this format:

### Version Format
- **Major.Minor.Patch** (e.g., 1.2.3)
- Major: Breaking changes or significant new features
- Minor: New features, backward compatible
- Patch: Bug fixes, backward compatible

### Change Categories
- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security-related changes

### Technical Details
Each release includes technical implementation details for developers and operators.

## Migration Guides

### Upgrading from 1.2.x to 1.3.0

The 1.3.0 release includes critical bug fixes for placeholder service handling. No configuration changes are required, but the behavior of service deletion has been improved:

#### What Changed
- Service deletion now properly removes routes from VirtualServices
- Placeholder services are created without routes in VirtualServices
- Enhanced logging provides better visibility into service deletion processes

#### Migration Steps
1. Update the operator image to v1.3.0
2. Monitor logs during the upgrade to ensure proper operation
3. Test service deletion scenarios to verify correct behavior

#### Verification
```bash
# Verify operator version
kubectl get deployment virtualservice-operator -n virtualservice-operator-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Test service deletion behavior
kubectl delete service test-service -n production
kubectl logs -n virtualservice-operator-system deployment/virtualservice-operator | grep "service deletion"
```

### Upgrading from 1.1.x to 1.2.0

The 1.2.0 release introduces multi-namespace support and template-based configuration.

#### Configuration Changes
Update your ConfigMap to include the new template-based configuration:

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

### Upgrading from 1.0.x to 1.1.0

The 1.1.0 release moves from environment variable configuration to ConfigMap-based configuration.

#### Configuration Migration
1. Create the new ConfigMap:
```bash
kubectl apply -f - <<EOF
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
EOF
```

2. Update the deployment to remove environment variables and add ConfigMap reference
3. Restart the operator to load the new configuration

## Support and Compatibility

### Kubernetes Compatibility Matrix

| Operator Version | Kubernetes Version | Istio Version | Go Version |
|------------------|-------------------|---------------|------------|
| 1.3.0            | 1.20+             | 1.17+         | 1.21+      |
| 1.2.x            | 1.20+             | 1.17+         | 1.21+      |
| 1.1.x            | 1.19+             | 1.16+         | 1.20+      |
| 1.0.x            | 1.18+             | 1.15+         | 1.19+      |

### Breaking Changes

#### Version 1.3.0
- No breaking changes, backward compatible

#### Version 1.2.0
- Configuration format changed from environment variables to ConfigMap
- VirtualService template format introduced
- Migration required for existing deployments

#### Version 1.1.0
- RBAC permissions expanded
- Configuration structure changed
- Deployment manifest updates required

## Contributing

### Changelog Maintenance

When contributing to the project:

1. **Add entries to [Unreleased]** section for new changes
2. **Follow the established format** for consistency
3. **Include technical details** for implementation changes
4. **Add migration notes** for breaking changes
5. **Update compatibility matrix** when dependencies change

### Release Process

1. Move changes from [Unreleased] to new version section
2. Update version numbers in all relevant files
3. Create git tag for the release
4. Build and push container images
5. Update deployment manifests
6. Create GitHub release with changelog excerpt

This changelog provides a comprehensive history of the VirtualService Operator project, including detailed technical information, migration guides, and compatibility information to help users understand changes and upgrade safely.