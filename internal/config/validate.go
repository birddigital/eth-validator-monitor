package config

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Validate checks that all required configuration is present and valid
func (c *Config) Validate() error {
	var errors []string

	// Validate Server
	if err := c.validateServer(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate Database
	if err := c.validateDatabase(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate Redis
	if err := c.validateRedis(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate Beacon Chain
	if err := c.validateBeaconChain(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate Monitoring
	if err := c.validateMonitoring(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate JWT (optional - only if secret key is set)
	if err := c.validateJWT(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate Session (optional - only if secret key is set)
	if err := c.validateSession(); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation errors:\n  - %s",
			strings.Join(errors, "\n  - "))
	}

	return nil
}

func (c *Config) validateServer() error {
	// Validate HTTP_PORT
	if c.Server.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT is required")
	}
	if _, err := strconv.Atoi(c.Server.HTTPPort); err != nil {
		return fmt.Errorf("HTTP_PORT must be a valid port number: %s", c.Server.HTTPPort)
	}

	// Validate GIN_MODE
	validModes := map[string]bool{"debug": true, "release": true, "test": true}
	if !validModes[c.Server.GinMode] {
		return fmt.Errorf("GIN_MODE must be 'debug', 'release', or 'test', got: %s",
			c.Server.GinMode)
	}

	return nil
}

func (c *Config) validateDatabase() error {
	var errors []string

	// Required fields
	if c.Database.User == "" {
		errors = append(errors, "DB_USER is required")
	}
	if c.Database.Password == "" {
		errors = append(errors, "DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		errors = append(errors, "DB_NAME is required")
	}

	// Validate port
	if _, err := strconv.Atoi(c.Database.Port); err != nil {
		errors = append(errors, fmt.Sprintf("DB_PORT must be a valid port number: %s",
			c.Database.Port))
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{
		"disable": true, "require": true, "verify-ca": true, "verify-full": true,
	}
	if !validSSLModes[c.Database.SSLMode] {
		errors = append(errors, fmt.Sprintf("DB_SSL_MODE must be one of: disable, require, verify-ca, verify-full, got: %s",
			c.Database.SSLMode))
	}

	if len(errors) > 0 {
		return fmt.Errorf("database config errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (c *Config) validateRedis() error {
	// Redis password is optional
	// Validate Redis DB number
	if c.Redis.DB < 0 || c.Redis.DB > 15 {
		return fmt.Errorf("REDIS_DB must be between 0 and 15, got: %d", c.Redis.DB)
	}

	// Validate address format (should be host:port)
	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	parts := strings.Split(c.Redis.Addr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("REDIS_ADDR must be in format host:port, got: %s", c.Redis.Addr)
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return fmt.Errorf("REDIS_ADDR port must be valid number in %s", c.Redis.Addr)
	}

	return nil
}

func (c *Config) validateBeaconChain() error {
	if c.BeaconChain.NodeURL == "" {
		return fmt.Errorf("BEACON_NODE_URL is required")
	}

	// Validate URL format
	parsedURL, err := url.Parse(c.BeaconChain.NodeURL)
	if err != nil {
		return fmt.Errorf("BEACON_NODE_URL must be a valid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("BEACON_NODE_URL must use http or https scheme, got: %s",
			parsedURL.Scheme)
	}

	return nil
}

func (c *Config) validateMonitoring() error {
	if c.Monitoring.PrometheusPort == "" {
		return fmt.Errorf("PROMETHEUS_PORT is required")
	}

	if _, err := strconv.Atoi(c.Monitoring.PrometheusPort); err != nil {
		return fmt.Errorf("PROMETHEUS_PORT must be a valid port number: %s",
			c.Monitoring.PrometheusPort)
	}

	return nil
}

func (c *Config) validateJWT() error {
	// JWT is optional - if no secret key, authentication is disabled
	if c.JWT.SecretKey == "" {
		return nil
	}

	// Validate JWT secret strength
	if err := validateJWTSecret(c.JWT.SecretKey); err != nil {
		return err
	}

	// Validate durations are positive
	if c.JWT.AccessTokenDuration <= 0 {
		return fmt.Errorf("JWT_ACCESS_TOKEN_DURATION must be positive, got: %v", c.JWT.AccessTokenDuration)
	}

	if c.JWT.RefreshTokenDuration <= 0 {
		return fmt.Errorf("JWT_REFRESH_TOKEN_DURATION must be positive, got: %v", c.JWT.RefreshTokenDuration)
	}

	// Validate refresh is longer than access
	if c.JWT.RefreshTokenDuration <= c.JWT.AccessTokenDuration {
		return fmt.Errorf("JWT_REFRESH_TOKEN_DURATION must be longer than JWT_ACCESS_TOKEN_DURATION")
	}

	return nil
}

// validateJWTSecret ensures the JWT secret meets security requirements
func validateJWTSecret(secret string) error {
	const minJWTSecretLength = 32 // 256 bits minimum

	if secret == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required but not set")
	}

	// Check minimum length
	if len(secret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET_KEY must be at least %d characters for security (got %d)",
			minJWTSecretLength, len(secret))
	}

	// Check for common weak patterns
	weakPatterns := []struct {
		pattern string
		message string
	}{
		{`^(?i)(secret|password|test|demo|admin|12345|changeme|your-secret)`, "uses a common weak word"},
		{`^(.)\1+$`, "uses repeated characters"},
		{`^[0-9]+$`, "uses only numbers"},
		{`^[a-zA-Z]+$`, "uses only letters"},
	}

	for _, wp := range weakPatterns {
		matched, _ := regexp.MatchString(wp.pattern, secret)
		if matched {
			return fmt.Errorf("JWT_SECRET_KEY %s; use a cryptographically secure random value (e.g., openssl rand -base64 32)", wp.message)
		}
	}

	// Calculate Shannon entropy to detect low-randomness secrets
	entropy := calculateEntropy(secret)
	minExpectedEntropy := 4.5 // bits per character (indicates good randomness)

	if entropy < minExpectedEntropy {
		return fmt.Errorf("JWT_SECRET_KEY has low entropy (%.2f bits/char); expected >= %.2f bits/char for cryptographic strength. Generate with: openssl rand -base64 32",
			entropy, minExpectedEntropy)
	}

	return nil
}

// calculateEntropy computes Shannon entropy in bits per character
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]int)
	for _, c := range s {
		freq[c]++
	}

	var entropy float64
	length := float64(len(s))

	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (c *Config) validateSession() error {
	// Session config is optional - if no secret key, session auth is disabled
	if c.Session.SecretKey == "" {
		return nil
	}

	// Validate session secret strength (use same validation as JWT)
	const minSessionSecretLength = 32 // 256 bits minimum
	if len(c.Session.SecretKey) < minSessionSecretLength {
		return fmt.Errorf("SESSION_SECRET_KEY must be at least %d characters for security (got %d)",
			minSessionSecretLength, len(c.Session.SecretKey))
	}

	// Check entropy
	entropy := calculateEntropy(c.Session.SecretKey)
	minExpectedEntropy := 4.5
	if entropy < minExpectedEntropy {
		return fmt.Errorf("SESSION_SECRET_KEY has low entropy (%.2f bits/char); expected >= %.2f bits/char. Generate with: openssl rand -base64 32",
			entropy, minExpectedEntropy)
	}

	// Validate MaxAge is positive
	if c.Session.MaxAge <= 0 {
		return fmt.Errorf("SESSION_MAX_AGE must be positive, got: %v", c.Session.MaxAge)
	}

	// Validate SameSite value
	validSameSite := map[string]bool{"Strict": true, "Lax": true, "None": true}
	if !validSameSite[c.Session.SameSite] {
		return fmt.Errorf("SESSION_SAME_SITE must be 'Strict', 'Lax', or 'None', got: %s",
			c.Session.SameSite)
	}

	// If SameSite=None, Secure must be true
	if c.Session.SameSite == "None" && !c.Session.Secure {
		return fmt.Errorf("SESSION_SECURE must be true when SESSION_SAME_SITE is 'None'")
	}

	return nil
}
