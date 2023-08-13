package ingresswatcher

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	// _ "github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// waitForChallengeAbsence waits for the absence of the ACME challenge path in the Ingress or until a timeout is reached.
// Returns the time it took to renew(timeout*2 when it failed) or there is an error.
func (iw *IngressWatcher) waitForChallengeAbsence(ctx context.Context, timeout time.Duration, ingNamespace, ingName string) (time.Duration, error) {
	logger.Debug("waitForChallengeAbsence")

	// Capture the start time
	startTime := time.Now()

	// Create a child context with the specified timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel() // Ensure resources are cleaned up after timeout or successful completion

	for {
		select {
		case <-timeoutCtx.Done():
			logger.Info("Timeout reached or context cancelled. Stopping.")
			return timeout * 2, nil
		default:
			logger.Debug("Checking Ingress")

			// Get the Ingress
			ingress := &networkingv1.Ingress{}
			if err := iw.ClientObj.Get(timeoutCtx, client.ObjectKey{Name: ingName, Namespace: ingNamespace}, ingress); err != nil {
				logger.Errorf("Error fetching ingress, %v", err)
				elapsedTime := time.Since(startTime)
				return elapsedTime, err
			}

			// Check all paths of the Ingress for the ACME challenge path
			pathFound := false
			for _, rule := range ingress.Spec.Rules {
				for _, pathType := range rule.HTTP.Paths {
					if strings.Contains(pathType.Path, ".well-known/acme-challenge") {
						logger.Debug("ACME challenge path found")
						pathFound = true
						break
					}
				}
				if pathFound {
					break
				}
			}

			if !pathFound {
				logger.Info("ACME challenge path not found. Stopping.")

				// If we reach here, the ACME challenge path was not found in any rule
				elapsedTime := time.Since(startTime)
				return elapsedTime, nil // Return the elapsed time on success
			}

			// Introduce a short delay to prevent high CPU usage
			time.Sleep(1 * time.Second)
		}
	}
}

// isContainsAcmeChallenge checks if the given ingress contains any ACME challenge paths.
func isContainsAcmeChallenge(ctx context.Context, ing *networkingv1.Ingress) bool {
	logger.Debug("isContainsAcmeChallenge")

	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if isAcmeChallengePath(ctx, path.Path) {
				logger.Debugf("Found %s in path %s", ".well-known/acme-challenge", path.Path)
				return true
			}
		}
	}
	return false
}

// isAcmeChallengePath checks if the given path contains the ACME challenge string.
func isAcmeChallengePath(ctx context.Context, p string) bool {
	logger.Debug("isAcmeChallengePath")

	const acmeChallengePath = ".well-known/acme-challenge"

	return strings.Contains(p, acmeChallengePath)
}

// timeRemainingCertificateUpToRenewal return the certificate time remaining for renewal and the secret name.
func (iw *IngressWatcher) timeRemainingCertificateUpToRenewal(ctx context.Context, ing *networkingv1.Ingress) (time.Duration, string, error) {
	logger.Debug("timeRemainingCertificateUpToRenewal")

	// Iterate over spec.tls[] to fetch associated secrets
	for _, tlsSpec := range ing.Spec.TLS {
		secretName := tlsSpec.SecretName

		// Fetch the secret
		secret := &corev1.Secret{}
		err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ing.Namespace}, secret)
		if err != nil {
			logger.Errorf("Failed to fetch secret %s: %v", secretName, err)
			return 0, secretName, err
		}

		// Extract the certificate from the secret. Assuming it's stored under the key "tls.crt"
		certData, ok := secret.Data["tls.crt"]
		if !ok {
			logger.Errorf("Secret %s does not have tls.crt", secretName)
			return 0, secretName, errors.New("missing tls.crt in secret")
		}

		// Check if the certificate is in PEM or DER format
		var certDER []byte
		if strings.Contains(string(certData), "-----BEGIN CERTIFICATE-----") {
			logger.Debug("renewValidCertificateIfNecessary - PEM format")

			// Decode PEM to get the DER-encoded certificate
			block, _ := pem.Decode(certData)
			if block == nil || block.Type != "CERTIFICATE" {
				logger.Errorf("Failed to decode PEM block from secret %s", secretName)
				return 0, secretName, errors.New("failed to decode PEM block")
			}
			certDER = block.Bytes
		} else {
			logger.Debug("renewValidCertificateIfNecessary - DER format")

			// Assume it's DER format
			certDER = certData
		}

		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			logger.Errorf("Failed to parse certificate from secret %s: %v", secretName, err)
			return 0, secretName, err
		}

		// Calculate remaining duration until certificate expiry
		timeRemaining := cert.NotAfter.Sub(time.Now())
		logger.Infof("timeRemaining: %v", timeRemaining)

		return timeRemaining, secretName, nil
	}

	return 0, "", nil
}
