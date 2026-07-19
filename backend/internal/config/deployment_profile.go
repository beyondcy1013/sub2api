package config

import "strings"

const (
	DeploymentProfileMain = "main"
	DeploymentProfileFree = "free"
)

type DeploymentConfig struct {
	Profile string `mapstructure:"profile"`
}

type RuntimeCapabilities struct {
	BalanceCheck              bool
	StickySessionReassignment bool
	BrandedErrors             bool
}

func normalizeDeploymentProfile(profile string) string {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile == "" {
		return DeploymentProfileMain
	}
	return profile
}

func (c *Config) DeploymentProfile() string {
	if c == nil {
		return DeploymentProfileMain
	}
	return normalizeDeploymentProfile(c.Deployment.Profile)
}

func (c *Config) RuntimeCapabilities() RuntimeCapabilities {
	if c.DeploymentProfile() == DeploymentProfileFree {
		return RuntimeCapabilities{
			BalanceCheck:              true,
			StickySessionReassignment: false,
			BrandedErrors:             true,
		}
	}
	return RuntimeCapabilities{
		BalanceCheck:              false,
		StickySessionReassignment: true,
		BrandedErrors:             false,
	}
}
