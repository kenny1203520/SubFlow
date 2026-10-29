package config

import "os"

// Runtime contains the small set of process settings needed before PocketBase
// can load its persistent application settings.
type Runtime struct {
	HTTPPort      string
	Environment   string
	EncryptionKey string
}

func LoadRuntime() Runtime {
	environment := os.Getenv("SUBFLOW_ENV")
	if environment == "" {
		environment = "production"
	}
	return Runtime{HTTPPort: os.Getenv("SUBFLOW_HTTP_PORT"), Environment: environment, EncryptionKey: os.Getenv("SUBFLOW_SETTINGS_ENCRYPTION_KEY")}
}

func DeprecatedEnvironmentNames() []string {
	deprecated := []string{"SUBFLOW_DATA_DRIVER", "SUBFLOW_PUBLISH_PORT", "SUBFLOW_APP_URL", "SUBFLOW_SMTP_HOST", "SUBFLOW_SMTP_PORT", "SUBFLOW_SMTP_USER", "SUBFLOW_SMTP_PASSWORD", "SUBFLOW_SMTP_FROM", "SUBFLOW_AUDIT_HMAC_KEY", "SUBFLOW_SHARE_SESSION_HMAC_KEY", "VITE_BACKEND_URL"}
	var present []string
	for _, name := range deprecated {
		if _, ok := os.LookupEnv(name); ok {
			present = append(present, name)
		}
	}
	return present
}
