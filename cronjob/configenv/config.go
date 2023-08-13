package configenv

import (
	"errors"
	"os"
	"strconv"
)

// Config holds all configuration for our program
type ConfigEnv struct {
	RunMode                     string
	CertificateRenewalThreshold int  // in days
	AnnotationRemovalDelay      int  // in seconds
	AdminUserPermission         bool // for reading secrets
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*ConfigEnv, error) {
	cfg := &ConfigEnv{
		RunMode:                     getEnv("RUN_MODE", "dev"),
		CertificateRenewalThreshold: getEnvAsInt("CERTIFICATE_RENEWAL_THRESHOLD", 60),
		AnnotationRemovalDelay:      getEnvAsInt("ANNOTATION_REMOVAL_DELAY", 10),
		AdminUserPermission:         getEnv("ADMIN_USER_PERMISSION", "false") == "true",
	}

	// Validation
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *ConfigEnv) error {
	// Check that RunMode is either 'dev' or 'prod'
	if cfg.RunMode != "dev" && cfg.RunMode != "prod" {
		return errors.New("RUN_MODE must be either 'dev' or 'prod'")
	}

	// Check that CertificateRenewalThreshold is a positive number
	if cfg.CertificateRenewalThreshold <= 0 {
		return errors.New("CERTIFICATE_RENEWAL_THRESHOLD must be a positive number")
	}

	// Check that AnnotationRemovalDelay is a positive number
	if cfg.AnnotationRemovalDelay <= 0 {
		return errors.New("ANNOTATION_REMOVAL_DELAY must be a positive number")
	}

	// Check that AdminUserPermission is a boolean
	if cfg.AdminUserPermission != true && cfg.AdminUserPermission != false {
		return errors.New("ADMIN_USER_PERMISSION must be a boolean")
	}

	return nil
}

// getEnv fetches an environment variable, returning a default value if it's not found
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt fetches an environment variable as an integer, returning a default value if it's not found
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
