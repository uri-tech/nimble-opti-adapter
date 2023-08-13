package utils

// // generateIngressRules creates a list of Ingress rules for the provided paths.
// func generateIngressRules(paths []string) []networkingv1.IngressRule {
// 	rulesIn := []networkingv1.IngressRule{}

// 	for _, path := range paths {
// 		rule := networkingv1.IngressRule{
// 			IngressRuleValue: networkingv1.IngressRuleValue{
// 				HTTP: &networkingv1.HTTPIngressRuleValue{
// 					Paths: []networkingv1.HTTPIngressPath{
// 						{
// 							Path: path,
// 						},
// 					},
// 				},
// 			},
// 		}
// 		rulesIn = append(rulesIn, rule)
// 	}

// 	return rulesIn
// }

// // generateIngress creates an Ingress object with the given name, namespace, labels, paths, and annotations.
// func generateIngress(name, namespace string, labels map[string]string, paths []string, annotations map[string]string) *networkingv1.Ingress {
// 	ingressRules := generateIngressRules(paths)

// 	return &networkingv1.Ingress{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:        name,
// 			Namespace:   namespace,
// 			Labels:      labels,
// 			Annotations: annotations,
// 		},
// 		Spec: networkingv1.IngressSpec{
// 			Rules: ingressRules,
// 		},
// 	}
// }

// // generateTestCert creates a self-signed X.509 certificate with the provided expiration time.
// func generateTestCert(expiration time.Time) ([]byte, error) {
// 	// Generate an ECDSA private key.
// 	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate private key: %v", err)
// 	}

// 	// Define the template for the certificate.
// 	template := x509.Certificate{
// 		SerialNumber: big.NewInt(1),
// 		Subject: pkix.Name{
// 			Organization: []string{"Test Org"},
// 			Country:      []string{"US"},
// 			Province:     []string{"California"},
// 			Locality:     []string{"San Francisco"},
// 		},
// 		NotBefore:             time.Now(),
// 		NotAfter:              expiration,
// 		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
// 		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
// 		BasicConstraintsValid: true,
// 		IsCA:                  false,
// 		DNSNames:              []string{"localhost"},
// 	}

// 	// Create the self-signed certificate.
// 	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create certificate: %v", err)
// 	}

// 	return certDER, nil
// }
