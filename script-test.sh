#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e

echo "Applying Ingress resource..."

kubectl label ingress argocd-server -n argocd nimble.opti.adapter/enabled=true

cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server
  namespace: argocd
  labels:
    nimble.opti.adapter/enabled: "true"
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    acme.cert-manager.io/http01-edit-in-place: 'true'
    kubernetes.io/tls-acme: 'true'
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - argo.127.0.0.1.nip.io
      secretName: letsencrypt-argo-example
  rules:
    - host: argo.127.0.0.1.nip.io
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: argocd-server
                port:
                  number: 443
EOF

cat <<EOF | kubectl apply -f -
kind: Service
apiVersion: v1
metadata:
  name: svc-externalname-dashboard
  namespace: default
spec:
  type: ExternalName
  externalName: kubernetes-dashboard.kubernetes-dashboard.svc.cluster.local
EOF

cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  namespace: default
  labels:
    nimble.opti.adapter/enabled: "true"
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    acme.cert-manager.io/http01-edit-in-place: 'true' # allows the cert-manager to edit the Ingress resource in place to solve the challenge, rather than creating additional resources.
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - ex.tech-ua.com
      secretName: letsencrypt-example
  rules:
    - host: ex.tech-ua.com
      http:
        paths:
          - path: /
            pathType: ImplementationSpecific
            backend:
              service:
                name: svc-externalname-dashboard
                port:
                  number: 80
EOF

echo "Ingress resource applied successfully."

echo "Applying NimbleOpti resource..."

cat <<EOF | kubectl apply -f -
apiVersion: adapter.uri-tech.github.io/v1
kind: NimbleOpti
metadata:
  name: default
  namespace: default
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 10
EOF

echo "NimbleOpti resource applied successfully."

cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
name: letsencrypt-prod
spec:
acme:
server: https://acme-v02.api.letsencrypt.org/directory
email: uri.al.1500@gmail.com
privateKeySecretRef:
name: letsencrypt-prod
solvers:
- dns01:
clouddns:
project: your-gcp-project-id
serviceAccountSecretRef:
name: clouddns-dns01-solver-svc-acct
key: key.json
EOF

cat <<EOF | kubectl apply -f -
kind: Service
apiVersion: v1
metadata:
  name: metrics-svc
  namespace: nimble-opti-adapter-system
  labels:
  annotations:
spec:
  ports:
    - name: https
      protocol: TCP
      port: 8080
      targetPort: 8080
  selector:
    control-plane: controller-manager
  type: ClusterIP
EOF
