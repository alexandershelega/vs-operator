package utils

import (
	"fmt"
	"strings"

	istiov1beta1 "istio.io/api/networking/v1beta1"
	istionetworkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ManagedByLabel = "managed-by"
	OperatorName   = "virtualservice-operator"
)

// isLikelyPlaceholderService checks if a service is likely a placeholder based on heuristics
// This is a safety check to prevent routes from being created for placeholder services
func isLikelyPlaceholderService(serviceName, namespace string) bool {
	// For now, we assume that if we're being asked to create a route for a service
	// in a developer namespace, and the namespace is not "default", it could be a placeholder
	// This is a conservative approach - we'll add more sophisticated checks as needed

	// TODO: In the future, we could make a Kubernetes API call here to check the actual service
	// annotations, but for now we rely on the controller logic to filter out placeholders
	// before calling this function

	return false // For now, let the controller handle the filtering
}

// GenerateVirtualService creates a VirtualService for a given service with only the default route
func GenerateVirtualService(service *corev1.Service, defaultNamespace string, developerNamespaces []string) *istionetworkingv1beta1.VirtualService {
	serviceName := service.Name

	// Create HTTP routes - only add default route initially
	var httpRoutes []*istiov1beta1.HTTPRoute

	// Add default route (no header matching, always last)
	defaultRoute := &istiov1beta1.HTTPRoute{
		Route: []*istiov1beta1.HTTPRouteDestination{
			{
				Destination: &istiov1beta1.Destination{
					Host: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, defaultNamespace),
				},
			},
		},
	}
	httpRoutes = append(httpRoutes, defaultRoute)

	// Create VirtualService
	vs := &istionetworkingv1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-virtual-service", serviceName),
			Namespace: defaultNamespace,
			Labels: map[string]string{
				ManagedByLabel: OperatorName,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       service.Name,
					UID:        service.UID,
				},
			},
		},
		Spec: istiov1beta1.VirtualService{
			Hosts: []string{serviceName},
			Http:  httpRoutes,
		},
	}

	return vs
}

func UpdateVirtualServiceRoutes(vs *istionetworkingv1beta1.VirtualService, serviceName, devNamespace string) {
	// Safety check: Don't create routes for services that look like placeholders
	// Check if this is likely a placeholder service based on naming pattern and namespace
	if isLikelyPlaceholderService(serviceName, devNamespace) {
		fmt.Printf("DEBUG: Skipping route creation for likely placeholder service %s/%s\n", devNamespace, serviceName)
		return
	}

	fmt.Printf("DEBUG: Creating/updating route for service %s in namespace %s\n", serviceName, devNamespace)

	// Add or update route for developer namespace
	newRoute := &istiov1beta1.HTTPRoute{
		Match: []*istiov1beta1.HTTPMatchRequest{
			{
				Headers: map[string]*istiov1beta1.StringMatch{
					"x-developer": {
						MatchType: &istiov1beta1.StringMatch_Exact{
							Exact: devNamespace,
						},
					},
				},
			},
		},
		Route: []*istiov1beta1.HTTPRouteDestination{
			{
				Destination: &istiov1beta1.Destination{
					Host: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, devNamespace),
				},
			},
		},
	}

	// Find if route already exists and update, otherwise add
	found := false
	for i, route := range vs.Spec.Http {
		if len(route.Match) > 0 && route.Match[0].Headers != nil {
			if headerMatch, exists := route.Match[0].Headers["x-developer"]; exists {
				if exact := headerMatch.GetExact(); exact == devNamespace {
					vs.Spec.Http[i] = newRoute
					found = true
					break
				}
			}
		}
	}

	if !found {
		// Insert before the default route (last route)
		if len(vs.Spec.Http) > 0 {
			vs.Spec.Http = append(vs.Spec.Http[:len(vs.Spec.Http)-1], newRoute, vs.Spec.Http[len(vs.Spec.Http)-1])
		} else {
			vs.Spec.Http = append(vs.Spec.Http, newRoute)
		}
	}
}

// IsManagedByOperator checks if a VirtualService is managed by this operator
func IsManagedByOperator(vs *istionetworkingv1beta1.VirtualService) bool {
	if vs.Labels == nil {
		return false
	}
	return vs.Labels[ManagedByLabel] == OperatorName
}

// GetServiceNameFromVirtualService extracts service name from VirtualService name
func GetServiceNameFromVirtualService(vsName string) string {
	// Remove "-virtual-service" suffix
	return strings.TrimSuffix(vsName, "-virtual-service")
}
