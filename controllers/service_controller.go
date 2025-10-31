package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	istiov1beta1 "istio.io/api/networking/v1beta1"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"virtualservice-operator/internal/config"
	"virtualservice-operator/internal/utils"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ConfigManager *config.ConfigManager
}

// Reconcile handles Service events and manages VirtualServices
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get operator configuration
	config, err := r.ConfigManager.GetConfig(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get operator config: %w", err)
	}

	// Check if this namespace should be watched
	watchedNamespaces, err := r.ConfigManager.GetWatchedNamespaces(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get watched namespaces: %w", err)
	}
	shouldWatch := false
	for _, ns := range watchedNamespaces {
		if ns == req.Namespace {
			shouldWatch = true
			break
		}
	}

	if !shouldWatch {
		// Ignore services in non-watched namespaces
		return ctrl.Result{}, nil
	}

	// Get the Service
	var service corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			// Service was deleted, handle cleanup
			return r.handleServiceDeletion(ctx, req.Name, req.Namespace, config)
		}
		return ctrl.Result{}, err
	}

	// Handle service creation/update
	if req.Namespace == config.DefaultNamespace {
		return r.handleDefaultNamespaceService(ctx, &service, config)
	} else {
		return r.handleDeveloperNamespaceService(ctx, &service, config)
	}
}

// isSystemService checks if a service is a system service that should be excluded from VirtualService creation
func (r *ServiceReconciler) isSystemService(serviceName string) bool {
	systemServices := []string{
		"kubernetes",
		"kube-dns",
		"kube-system",
	}

	for _, sysService := range systemServices {
		if serviceName == sysService {
			return true
		}
	}
	return false
}

// isPlaceholderService checks if a service is a placeholder service created by the operator
// Uses annotations as primary detection method with fallback to service type and external name pattern
func (r *ServiceReconciler) isPlaceholderService(service *corev1.Service) bool {
	// Primary detection: Check for placeholder annotation
	if service.Annotations != nil {
		if placeholderAnnotation, exists := service.Annotations["virtualservice-operator/placeholder-service"]; exists && placeholderAnnotation == "true" {
			fmt.Printf("DEBUG: Service %s/%s identified as placeholder via annotation\n", service.Namespace, service.Name)
			return true
		}
	}

	// Fallback detection: Check if it's an ExternalName service pointing to default namespace
	if service.Spec.Type == corev1.ServiceTypeExternalName && service.Spec.ExternalName != "" {
		// Check if external name matches pattern *.default.svc.cluster.local
		if strings.HasSuffix(service.Spec.ExternalName, ".default.svc.cluster.local") {
			fmt.Printf("DEBUG: Service %s/%s identified as placeholder via ExternalName pattern: %s\n", service.Namespace, service.Name, service.Spec.ExternalName)
			return true
		}
	}

	// Legacy detection: Check for old label-based identification
	if service.Labels != nil {
		if placeholderLabel, exists := service.Labels["placeholder-service"]; exists && placeholderLabel == "true" {
			fmt.Printf("DEBUG: Service %s/%s identified as placeholder via legacy label\n", service.Namespace, service.Name)
			return true
		}
	}

	return false
}

// createPlaceholderServices creates placeholder ExternalName services in all developer namespaces
// createSinglePlaceholderService creates a placeholder service in a specific namespace
func (r *ServiceReconciler) createSinglePlaceholderService(ctx context.Context, sourceService *corev1.Service, targetNamespace string, config *config.OperatorConfig) error {
	if !config.EnablePlaceholderServices {
		return nil
	}

	// Skip system services
	if r.isSystemService(sourceService.Name) {
		return nil
	}

	fmt.Printf("DEBUG: Creating placeholder service %s in namespace %s\n", sourceService.Name, targetNamespace)

	// Check if service already exists in the target namespace
	existingService := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: sourceService.Name, Namespace: targetNamespace}, existingService)
	if err == nil {
		// Service already exists, skip creation
		fmt.Printf("DEBUG: Service %s already exists in namespace %s, skipping placeholder creation\n", sourceService.Name, targetNamespace)
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing service: %w", err)
	}

	// Create placeholder service
	placeholderService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceService.Name,
			Namespace: targetNamespace,
			Annotations: map[string]string{
				"virtualservice-operator/placeholder-service": "true",
				"virtualservice-operator/source-service":      fmt.Sprintf("%s.%s.svc.cluster.local", sourceService.Name, config.DefaultNamespace),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", sourceService.Name, config.DefaultNamespace),
		},
	}

	if err := r.Create(ctx, placeholderService); err != nil {
		return fmt.Errorf("failed to create placeholder service %s in namespace %s: %w", sourceService.Name, targetNamespace, err)
	}

	fmt.Printf("DEBUG: Successfully created placeholder service %s in namespace %s\n", sourceService.Name, targetNamespace)
	return nil
}

// ensurePlaceholderServicesForNamespace ensures all necessary placeholder services exist in a specific namespace
func (r *ServiceReconciler) ensurePlaceholderServicesForNamespace(ctx context.Context, targetNamespace string, config *config.OperatorConfig) error {
	if !config.EnablePlaceholderServices {
		return nil
	}

	// Get all services in the default namespace
	serviceList := &corev1.ServiceList{}
	err := r.List(ctx, serviceList, client.InNamespace(config.DefaultNamespace))
	if err != nil {
		return fmt.Errorf("failed to list services in default namespace: %w", err)
	}

	// For each service in default namespace, ensure a placeholder exists in the target namespace
	for _, defaultService := range serviceList.Items {
		if r.isSystemService(defaultService.Name) {
			continue
		}

		err := r.createSinglePlaceholderService(ctx, &defaultService, targetNamespace, config)
		if err != nil {
			return fmt.Errorf("failed to create placeholder service %s: %w", defaultService.Name, err)
		}
	}

	return nil
}

func (r *ServiceReconciler) createPlaceholderServices(ctx context.Context, sourceService *corev1.Service, config *config.OperatorConfig) error {
	log := ctrl.LoggerFrom(ctx)

	if !config.EnablePlaceholderServices {
		log.Info("Placeholder services feature is disabled")
		return nil // Feature is disabled
	}

	log.Info("Creating placeholder services", "sourceService", sourceService.Name, "sourceNamespace", sourceService.Namespace, "developerNamespaces", config.DeveloperNamespaces)

	for _, devNamespace := range config.DeveloperNamespaces {
		if devNamespace == config.DefaultNamespace {
			log.V(1).Info("Skipping placeholder creation in same namespace as source", "namespace", devNamespace)
			continue // Skip creating placeholder in the same namespace as source
		}

		log.Info("Checking for existing service", "serviceName", sourceService.Name, "namespace", devNamespace)

		// Check if placeholder service already exists
		existingService := &corev1.Service{}
		err := r.Get(ctx, types.NamespacedName{Name: sourceService.Name, Namespace: devNamespace}, existingService)
		if err == nil {
			log.Info("Service already exists, skipping placeholder creation", "serviceName", sourceService.Name, "namespace", devNamespace, "serviceType", existingService.Spec.Type)
			// Service already exists, don't modify it
			continue
		}
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check existing service", "serviceName", sourceService.Name, "namespace", devNamespace)
			return fmt.Errorf("failed to check existing service %s in namespace %s: %w", sourceService.Name, devNamespace, err)
		}

		log.Info("No existing service found, creating placeholder", "serviceName", sourceService.Name, "namespace", devNamespace)

		// Create placeholder service
		placeholderService := &corev1.Service{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      sourceService.Name,
				Namespace: devNamespace,
				Annotations: map[string]string{
					"virtualservice-operator/placeholder-service": "true",
					"virtualservice-operator/source-service":      fmt.Sprintf("%s.%s.svc.cluster.local", sourceService.Name, config.DefaultNamespace),
				},
			},
			Spec: corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", sourceService.Name, config.DefaultNamespace),
			},
		}

		if err := r.Create(ctx, placeholderService); err != nil {
			log.Error(err, "Failed to create placeholder service", "serviceName", sourceService.Name, "namespace", devNamespace)
			return fmt.Errorf("failed to create placeholder service %s in namespace %s: %w", sourceService.Name, devNamespace, err)
		}

		log.Info("Successfully created placeholder service", "serviceName", sourceService.Name, "namespace", devNamespace)
	}

	log.Info("Finished creating placeholder services", "sourceService", sourceService.Name)
	return nil
}

// deletePlaceholderServices deletes placeholder services from all developer namespaces
func (r *ServiceReconciler) deletePlaceholderServices(ctx context.Context, serviceName string, config *config.OperatorConfig) error {
	if !config.EnablePlaceholderServices {
		return nil // Feature is disabled
	}

	for _, devNamespace := range config.DeveloperNamespaces {
		if devNamespace == config.DefaultNamespace {
			continue // Skip the default namespace
		}

		// Get the service in the developer namespace
		service := &corev1.Service{}
		err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: devNamespace}, service)
		if err != nil {
			if errors.IsNotFound(err) {
				continue // Service doesn't exist, nothing to delete
			}
			return fmt.Errorf("failed to get service %s in namespace %s: %w", serviceName, devNamespace, err)
		}

		// Only delete if it's a placeholder service managed by us
		if r.isPlaceholderService(service) {
			if err := r.Delete(ctx, service); err != nil {
				return fmt.Errorf("failed to delete placeholder service %s in namespace %s: %w", serviceName, devNamespace, err)
			}
		}
	}

	return nil
}

// handleDefaultNamespaceService creates or updates VirtualService for services in the default namespace
func (r *ServiceReconciler) handleDefaultNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error) {
	// Skip system services
	if r.isSystemService(service.Name) {
		return ctrl.Result{}, nil
	}

	// Create placeholder services in developer namespaces if feature is enabled
	if err := r.createPlaceholderServices(ctx, service, config); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create placeholder services: %w", err)
	}

	// Generate VirtualService with only default route initially
	vs := utils.GenerateVirtualService(service, config.DefaultNamespace, config.DeveloperNamespaces)

	// Set owner reference
	if err := ctrl.SetControllerReference(service, vs, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Check if VirtualService already exists
	existingVS := &istionetworkingv1beta1.VirtualService{}
	err := r.Get(ctx, types.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}, existingVS)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new VirtualService with default route only
			if err := r.Create(ctx, vs); err != nil {
				return ctrl.Result{}, err
			}
			// Now check for existing services in developer namespaces and add routes
			return r.addExistingDeveloperRoutes(ctx, service, vs, config)
		}
		return ctrl.Result{}, err
	}

	// Update existing VirtualService if it's managed by us
	if utils.IsManagedByOperator(existingVS) {
		// Use retry logic to update the VirtualService
		err := r.retryVirtualServiceUpdate(ctx, existingVS, func(latest *istionetworkingv1beta1.VirtualService) error {
			// Regenerate with only default route - copy fields individually to avoid mutex copy
			latest.Spec.Hosts = vs.Spec.Hosts
			latest.Spec.Gateways = vs.Spec.Gateways
			latest.Spec.Http = vs.Spec.Http
			latest.Spec.Tls = vs.Spec.Tls
			latest.Spec.Tcp = vs.Spec.Tcp
			latest.Spec.ExportTo = vs.Spec.ExportTo
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		// Add routes for existing services in developer namespaces
		return r.addExistingDeveloperRoutes(ctx, service, existingVS, config)
	}

	return ctrl.Result{}, nil
}

// addExistingDeveloperRoutes checks each developer namespace for existing services and adds routes
func (r *ServiceReconciler) addExistingDeveloperRoutes(ctx context.Context, service *corev1.Service, vs *istionetworkingv1beta1.VirtualService, config *config.OperatorConfig) (ctrl.Result, error) {
	var namespacesToAdd []string

	for _, devNamespace := range config.DeveloperNamespaces {
		if devNamespace == config.DefaultNamespace {
			continue // Skip if developer namespace is same as default
		}

		// Check if service exists in this developer namespace
		devService := &corev1.Service{}
		err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: devNamespace}, devService)
		if err != nil {
			if errors.IsNotFound(err) {
				// Service doesn't exist in this developer namespace, skip
				continue
			}
			return ctrl.Result{}, err
		}

		// Skip placeholder services - they should not have VirtualService routes
		if r.isPlaceholderService(devService) {
			fmt.Printf("DEBUG: Skipping route addition for placeholder service %s/%s\n", devService.Namespace, devService.Name)
			continue
		}

		fmt.Printf("DEBUG: Adding route for real service %s/%s\n", devService.Namespace, devService.Name)

		// Service exists and is not a placeholder, add to list of namespaces to add routes for
		namespacesToAdd = append(namespacesToAdd, devNamespace)
	}

	// Update VirtualService if we have routes to add
	if len(namespacesToAdd) > 0 {
		err := r.retryVirtualServiceUpdate(ctx, vs, func(latest *istionetworkingv1beta1.VirtualService) error {
			for _, devNamespace := range namespacesToAdd {
				utils.UpdateVirtualServiceRoutes(latest, service.Name, devNamespace)
			}
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// handleDeveloperNamespaceService updates existing VirtualService for services in developer namespaces
func (r *ServiceReconciler) handleDeveloperNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error) {
	// Skip placeholder services - they should not have VirtualService routes
	if r.isPlaceholderService(service) {
		fmt.Printf("DEBUG: Skipping VirtualService route creation for placeholder service %s/%s\n", service.Namespace, service.Name)
		return ctrl.Result{}, nil
	}

	fmt.Printf("DEBUG: Processing developer namespace service %s/%s for VirtualService routes\n", service.Namespace, service.Name)

	// First, check if a service with the same name exists in the default namespace
	defaultService := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: config.DefaultNamespace}, defaultService)
	if err != nil {
		if errors.IsNotFound(err) {
			// No service with this name exists in default namespace
			// But we should check if there are other services in other developer namespaces
			// that might need placeholder services created for this namespace
			if config.EnablePlaceholderServices {
				err := r.ensurePlaceholderServicesForNamespace(ctx, service.Namespace, config)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to ensure placeholder services: %w", err)
				}
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Find the corresponding VirtualService in the default namespace
	// VirtualService name follows the pattern: serviceName + "-virtual-service"
	vsName := fmt.Sprintf("%s-virtual-service", service.Name)
	existingVS := &istionetworkingv1beta1.VirtualService{}
	err = r.Get(ctx, types.NamespacedName{Name: vsName, Namespace: config.DefaultNamespace}, existingVS)
	if err != nil {
		if errors.IsNotFound(err) {
			// VirtualService doesn't exist yet, nothing to update
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Update the VirtualService with new route for this developer namespace
	if utils.IsManagedByOperator(existingVS) {
		err := r.retryVirtualServiceUpdate(ctx, existingVS, func(latest *istionetworkingv1beta1.VirtualService) error {
			utils.UpdateVirtualServiceRoutes(latest, service.Name, service.Namespace)
			return nil
		})
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleServiceDeletion handles cleanup when a service is deleted
func (r *ServiceReconciler) handleServiceDeletion(ctx context.Context, serviceName, namespace string, config *config.OperatorConfig) (ctrl.Result, error) {
	if namespace == config.DefaultNamespace {
		// Delete the VirtualService when the main service is deleted
		// VirtualService name follows the pattern: serviceName + "-virtual-service"
		vsName := fmt.Sprintf("%s-virtual-service", serviceName)
		vs := &istionetworkingv1beta1.VirtualService{}
		err := r.Get(ctx, types.NamespacedName{Name: vsName, Namespace: namespace}, vs)
		if err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		if utils.IsManagedByOperator(vs) {
			if err := r.Delete(ctx, vs); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Delete placeholder services in developer namespaces if feature is enabled
		if err := r.deletePlaceholderServices(ctx, serviceName, config); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete placeholder services: %w", err)
		}
	} else {
		// Handle deletion in developer namespace
		fmt.Printf("DEBUG: Real service %s deleted from developer namespace %s. Always removing route from VirtualService.\n", serviceName, namespace)

		// ALWAYS remove the route from VirtualService when a real service is deleted
		// Placeholder services should NEVER have routes in VirtualService
		vsName := fmt.Sprintf("%s-virtual-service", serviceName)
		vs := &istionetworkingv1beta1.VirtualService{}
		err := r.Get(ctx, types.NamespacedName{Name: vsName, Namespace: config.DefaultNamespace}, vs)
		if err != nil {
			if errors.IsNotFound(err) {
				fmt.Printf("DEBUG: VirtualService %s not found, nothing to update.\n", vsName)
			} else {
				return ctrl.Result{}, err
			}
		} else {
			if utils.IsManagedByOperator(vs) {
				fmt.Printf("DEBUG: Removing route for namespace %s from VirtualService %s.\n", namespace, vsName)
				// Use retry logic to remove routes for this developer namespace
				err := r.retryVirtualServiceUpdate(ctx, vs, func(latest *istionetworkingv1beta1.VirtualService) error {
					var newRoutes []*istiov1beta1.HTTPRoute
					routesRemoved := 0
					for _, route := range latest.Spec.Http {
						if len(route.Match) > 0 && route.Match[0].Headers != nil {
							if headerMatch, exists := route.Match[0].Headers["x-developer"]; exists {
								if exact := headerMatch.GetExact(); exact == namespace {
									routesRemoved++
									continue // Skip this route - REMOVE IT
								}
							}
						}
						newRoutes = append(newRoutes, route)
					}
					latest.Spec.Http = newRoutes
					fmt.Printf("DEBUG: Removed %d routes for namespace %s from VirtualService.\n", routesRemoved, namespace)
					return nil
				})
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		// SEPARATELY handle placeholder service creation (if needed)
		// This is independent of route management - placeholder services don't get routes
		defaultService := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: config.DefaultNamespace}, defaultService)
		if err == nil {
			// Service exists in default namespace, so we should recreate the placeholder service
			// if placeholder services are enabled
			if config.EnablePlaceholderServices {
				fmt.Printf("DEBUG: Service %s exists in default namespace. Creating placeholder service in %s (no route will be added).\n", serviceName, namespace)

				// Create a single placeholder service for this namespace
				err := r.createSinglePlaceholderService(ctx, defaultService, namespace, config)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to recreate placeholder service: %w", err)
				}
				fmt.Printf("DEBUG: Placeholder service created for %s in namespace %s. Placeholder services do NOT get VirtualService routes.\n", serviceName, namespace)
			}
		} else if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// retryVirtualServiceUpdate performs a VirtualService update with retry logic and conflict resolution
func (r *ServiceReconciler) retryVirtualServiceUpdate(ctx context.Context, vs *istionetworkingv1beta1.VirtualService, updateFunc func(*istionetworkingv1beta1.VirtualService) error) error {
	backoff := wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}

	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		// Get the latest version of the VirtualService
		latest := &istionetworkingv1beta1.VirtualService{}
		err := r.Get(ctx, types.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}, latest)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, err // Don't retry if resource is deleted
			}
			return false, nil // Retry on other errors
		}

		// Apply the update function to the latest version
		if err := updateFunc(latest); err != nil {
			return false, err // Don't retry on update function errors
		}

		// Try to update
		err = r.Update(ctx, latest)
		if err != nil {
			if errors.IsConflict(err) {
				return false, nil // Retry on conflict
			}
			return false, err // Don't retry on other errors
		}

		return true, nil // Success
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a predicate that filters services based on watched namespaces
	namespacePredicate := predicate.NewPredicateFuncs(func(object client.Object) bool {
		watchedNamespaces, err := r.ConfigManager.GetWatchedNamespaces(context.Background())
		if err != nil {
			return false
		}

		for _, ns := range watchedNamespaces {
			if ns == object.GetNamespace() {
				return true
			}
		}
		return false
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(namespacePredicate).
		Complete(r)
}
