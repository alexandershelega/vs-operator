package config

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// OperatorConfig represents the operator configuration
type OperatorConfig struct {
	DefaultNamespace    string   `yaml:"defaultNamespace"`
	DeveloperNamespaces []string `yaml:"developerNamespaces"`
	VirtualServiceTemplate string `yaml:"virtualServiceTemplate"`
}

// ConfigManager manages operator configuration
type ConfigManager struct {
	client    client.Client
	namespace string
	configMapName string
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(client client.Client, namespace, configMapName string) *ConfigManager {
	return &ConfigManager{
		client:        client,
		namespace:     namespace,
		configMapName: configMapName,
	}
}

// GetConfig retrieves the operator configuration from ConfigMap
func (cm *ConfigManager) GetConfig(ctx context.Context) (*OperatorConfig, error) {
	configMap := &corev1.ConfigMap{}
	err := cm.client.Get(ctx, types.NamespacedName{
		Name:      cm.configMapName,
		Namespace: cm.namespace,
	}, configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap %s/%s: %w", cm.namespace, cm.configMapName, err)
	}

	configData, exists := configMap.Data["config.yaml"]
	if !exists {
		return nil, fmt.Errorf("config.yaml not found in ConfigMap %s/%s", cm.namespace, cm.configMapName)
	}

	var config OperatorConfig
	if err := yaml.Unmarshal([]byte(configData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default values if not provided
	if config.DefaultNamespace == "" {
		config.DefaultNamespace = "default"
	}

	return &config, nil
}

// GetWatchedNamespaces returns all namespaces that should be watched
func (cm *ConfigManager) GetWatchedNamespaces(ctx context.Context) ([]string, error) {
	config, err := cm.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	namespaces := []string{config.DefaultNamespace}
	namespaces = append(namespaces, config.DeveloperNamespaces...)
	
	return namespaces, nil
}