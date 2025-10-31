package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istiov1beta1 "istio.io/api/networking/v1beta1"
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

// handleDefaultNamespaceService creates or updates VirtualService for services in the default namespace
func (r *ServiceReconciler) handleDefaultNamespaceService(ctx context.Context, service *corev1.Service, config *config.OperatorConfig) (ctrl.Result, error) {
	// Skip system services
	if r.isSystemService(service.Name) {
		return ctrl.Result{}, nil
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
			// Regenerate with only default route
			latest.Spec = vs.Spec
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
		
		// Service exists, add to list of namespaces to add routes for
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
	// First, check if a service with the same name exists in the default namespace
	defaultService := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: config.DefaultNamespace}, defaultService)
	if err != nil {
		if errors.IsNotFound(err) {
			// No service with this name exists in default namespace, nothing to do
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
			return ctrl.Result{}, r.Delete(ctx, vs)
		}
	} else {
		// Remove the route for this developer namespace from the VirtualService
		// VirtualService name follows the pattern: serviceName + "-virtual-service"
		vsName := fmt.Sprintf("%s-virtual-service", serviceName)
		vs := &istionetworkingv1beta1.VirtualService{}
		err := r.Get(ctx, types.NamespacedName{Name: vsName, Namespace: config.DefaultNamespace}, vs)
		if err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		if utils.IsManagedByOperator(vs) {
			// Use retry logic to remove routes for this developer namespace
			err := r.retryVirtualServiceUpdate(ctx, vs, func(latest *istionetworkingv1beta1.VirtualService) error {
				var newRoutes []*istiov1beta1.HTTPRoute
				for _, route := range latest.Spec.Http {
					if len(route.Match) > 0 && route.Match[0].Headers != nil {
						if headerMatch, exists := route.Match[0].Headers["x-developer"]; exists {
							if exact := headerMatch.GetExact(); exact == namespace {
								continue // Skip this route
							}
						}
					}
					newRoutes = append(newRoutes, route)
				}
				latest.Spec.Http = newRoutes
				return nil
			})
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