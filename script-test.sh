#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e

echo "Applying Ingress resource..."

cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: minimal-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  labels:
    nimble.opti.adapter/enabled: "true"
spec:
  ingressClassName: nginx-example
  rules:
  - http:
      paths:
      - path: /testpath
        pathType: Prefix
        backend:
          service:
            name: test
            port:
              number: 80 
EOF

echo "Ingress resource applied successfully."

echo "Applying NimbleOpti resource..."

cat <<EOF | kubectl apply -f -
apiVersion: adapter.uri-tech.github.io/v1
kind: NimbleOpti
metadata:
  name: example-nimbleopti
spec:
  # Add any fields defined in the NimbleOpti spec here
  # For instance:
  # someField: someValue
EOF

echo "NimbleOpti resource applied successfully."
