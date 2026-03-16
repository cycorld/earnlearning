package config

import "os"

type Config struct {
	Port            string
	DBPath          string
	UploadPath      string
	JWTSecret       string
	AdminEmail      string
	AdminPassword   string
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
	// SES Email
	SESRegion          string
	SESAccessKeyID     string
	SESSecretAccessKey string
	SESFromEmail       string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		DBPath:          getEnv("DB_PATH", "./data/earnlearning.db"),
		UploadPath:      getEnv("UPLOAD_PATH", "./data/uploads"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		AdminEmail:      getEnv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:   getEnv("ADMIN_PASSWORD", "change-this"),
		VAPIDPublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:       getEnv("VAPID_SUBJECT", "mailto:admin@example.com"),
		SESRegion:          getEnv("SES_REGION", "ap-northeast-2"),
		SESAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		SESSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		SESFromEmail:       getEnv("SES_FROM_EMAIL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
