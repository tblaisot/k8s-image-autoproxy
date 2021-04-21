apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: k8s-image-autoproxy
  namespace: k8s-image-autoproxy
  labels:
    app: k8s-image-autoproxy
    kind: mutator
webhooks:
  - name: k8s-image-autoproxy.k8s-image-autoproxy.svc.cluster.local
    # Avoid chicken-egg problem with our webhook deployment.
    objectSelector:
      matchExpressions:
      - key: app
        operator: NotIn
        values: ["k8s-image-autoproxy"]
    admissionReviewVersions: ["v1"]
    sideEffects: None
    timeoutSeconds: 5
    reinvocationPolicy: Never
    failurePolicy: Ignore
    clientConfig:
      service:
        name: k8s-image-autoproxy
        namespace: k8s-image-autoproxy
        path: /mutate
        port: 443
      caBundle: CA_BUNDLE
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["*"]
        apiVersions: ["*"]
        resources: ["pods","replicationcontrollers","containers","deployments","replicasets","daemonsets","statefulsets","cronjobs","jobs"]