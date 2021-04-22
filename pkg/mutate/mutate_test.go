package mutate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	v1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var clientset kubernetes.Interface = fake.NewSimpleClientset(
	&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Annotations: map[string]string{
				AnnotationEnabled: "true",
			},
		},
	},
	&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "proxied",
			Annotations: map[string]string{
				AnnotationEnabled: "true",
				AnnotationProxy:   "proxy2.io",
			},
		},
	},
	&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "disabled",
			Annotations: map[string]string{
				AnnotationEnabled: "false",
			},
		},
	},
)

var mutator = imageProxyMutator{config: Config{DefaultProxy: "proxy.io", DefaultEnable: false, Verbose: true}, clientset: clientset}

func TestMutatesValidRequest(t *testing.T) {
	rawJSON := `{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1beta1",
		"request": {
			"uid": "7f0b2891-916f-4ed6-b7cd-27bff1815a8c",
			"kind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"resource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"requestKind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"requestResource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"namespace": "default",
			"operation": "CREATE",
			"userInfo": {
				"username": "kubernetes-admin",
				"groups": [
					"system:masters",
					"system:authenticated"
				]
			},
			"object": {
				"kind": "Pod",
				"apiVersion": "v1",
				"metadata": {
					"name": "c7m",
					"namespace": "default",
					"creationTimestamp": null,
					"labels": {
						"name": "c7m"
					},
					"annotations": {
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"annotations\":{},\"labels\":{\"name\":\"c7m\"},\"name\":\"c7m\",\"namespace\":\"default\"},\"spec\":{\"containers\":[{\"args\":[\"-c\",\"trap \\\"killall sleep\\\" TERM; trap \\\"kill -9 sleep\\\" KILL; sleep infinity\"],\"command\":[\"/bin/bash\"],\"image\":\"centos:7\",\"name\":\"c7m\"}]}}\n"
					}
				},
				"spec": {
					"volumes": [
						{
							"name": "default-token-5z7xl",
							"secret": {
								"secretName": "default-token-5z7xl"
							}
						}
					],
					"containers": [
						{
							"name": "c7m",
							"image": "centos:7",
							"command": [
								"/bin/bash"
							],
							"args": [
								"-c",
								"trap \"killall sleep\" TERM; trap \"kill -9 sleep\" KILL; sleep infinity"
							],
							"resources": {},
							"volumeMounts": [
								{
									"name": "default-token-5z7xl",
									"readOnly": true,
									"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
								}
							],
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"imagePullPolicy": "IfNotPresent"
						}
					],
					"restartPolicy": "Always",
					"terminationGracePeriodSeconds": 30,
					"dnsPolicy": "ClusterFirst",
					"serviceAccountName": "default",
					"serviceAccount": "default",
					"securityContext": {},
					"schedulerName": "default-scheduler",
					"tolerations": [
						{
							"key": "node.kubernetes.io/not-ready",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						},
						{
							"key": "node.kubernetes.io/unreachable",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						}
					],
					"priority": 0,
					"enableServiceLinks": true
				},
				"status": {}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1"
			}
		}
	}`
	response, err := mutator.Mutate([]byte(rawJSON))
	if err != nil {
		t.Errorf("failed to mutate AdmissionRequest %s with error %s", string(response), err)
	}

	r := v1beta1.AdmissionReview{}
	err = json.Unmarshal(response, &r)
	assert.NoError(t, err, "failed to unmarshal with error %s", err)

	rr := r.Response
	assert.Equal(t, `[{"op":"replace","path":"/spec/containers/0/image","value":"proxy.io/library/centos:7"}]`, string(rr.Patch))
	assert.Contains(t, rr.AuditAnnotations, "k8s-image-autoproxy")
}

func TestMutatesNamespaceOverride(t *testing.T) {
	rawJSON := `{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1beta1",
		"request": {
			"uid": "7f0b2891-916f-4ed6-b7cd-27bff1815a8c",
			"kind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"resource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"requestKind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"requestResource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"namespace": "proxied",
			"operation": "CREATE",
			"userInfo": {
				"username": "kubernetes-admin",
				"groups": [
					"system:masters",
					"system:authenticated"
				]
			},
			"object": {
				"kind": "Pod",
				"apiVersion": "v1",
				"metadata": {
					"name": "c7m",
					"namespace": "proxied",
					"creationTimestamp": null,
					"labels": {
						"name": "c7m"
					},
					"annotations": {
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"annotations\":{},\"labels\":{\"name\":\"c7m\"},\"name\":\"c7m\",\"namespace\":\"proxied\"},\"spec\":{\"containers\":[{\"args\":[\"-c\",\"trap \\\"killall sleep\\\" TERM; trap \\\"kill -9 sleep\\\" KILL; sleep infinity\"],\"command\":[\"/bin/bash\"],\"image\":\"centos:7\",\"name\":\"c7m\"}]}}\n"
					}
				},
				"spec": {
					"volumes": [
						{
							"name": "default-token-5z7xl",
							"secret": {
								"secretName": "default-token-5z7xl"
							}
						}
					],
					"containers": [
						{
							"name": "c7m",
							"image": "centos:7",
							"command": [
								"/bin/bash"
							],
							"args": [
								"-c",
								"trap \"killall sleep\" TERM; trap \"kill -9 sleep\" KILL; sleep infinity"
							],
							"resources": {},
							"volumeMounts": [
								{
									"name": "default-token-5z7xl",
									"readOnly": true,
									"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
								}
							],
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"imagePullPolicy": "IfNotPresent"
						}
					],
					"restartPolicy": "Always",
					"terminationGracePeriodSeconds": 30,
					"dnsPolicy": "ClusterFirst",
					"serviceAccountName": "default",
					"serviceAccount": "default",
					"securityContext": {},
					"schedulerName": "default-scheduler",
					"tolerations": [
						{
							"key": "node.kubernetes.io/not-ready",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						},
						{
							"key": "node.kubernetes.io/unreachable",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						}
					],
					"priority": 0,
					"enableServiceLinks": true
				},
				"status": {}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1"
			}
		}
	}`
	response, err := mutator.Mutate([]byte(rawJSON))
	if err != nil {
		t.Errorf("failed to mutate AdmissionRequest %s with error %s", string(response), err)
	}

	r := v1beta1.AdmissionReview{}
	err = json.Unmarshal(response, &r)
	assert.NoError(t, err, "failed to unmarshal with error %s", err)

	rr := r.Response
	assert.Equal(t, `[{"op":"replace","path":"/spec/containers/0/image","value":"proxy2.io/library/centos:7"}]`, string(rr.Patch))
	assert.Contains(t, rr.AuditAnnotations, "k8s-image-autoproxy")
}

func TestMutatesNamespaceDisabled(t *testing.T) {
	rawJSON := `{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1beta1",
		"request": {
			"uid": "7f0b2891-916f-4ed6-b7cd-27bff1815a8c",
			"kind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"resource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"requestKind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"requestResource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"namespace": "disabled",
			"operation": "CREATE",
			"userInfo": {
				"username": "kubernetes-admin",
				"groups": [
					"system:masters",
					"system:authenticated"
				]
			},
			"object": {
				"kind": "Pod",
				"apiVersion": "v1",
				"metadata": {
					"name": "c7m",
					"namespace": "disabled",
					"creationTimestamp": null,
					"labels": {
						"name": "c7m"
					},
					"annotations": {
						"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"annotations\":{},\"labels\":{\"name\":\"c7m\"},\"name\":\"c7m\",\"namespace\":\"disabled\"},\"spec\":{\"containers\":[{\"args\":[\"-c\",\"trap \\\"killall sleep\\\" TERM; trap \\\"kill -9 sleep\\\" KILL; sleep infinity\"],\"command\":[\"/bin/bash\"],\"image\":\"centos:7\",\"name\":\"c7m\"}]}}\n"
					}
				},
				"spec": {
					"volumes": [
						{
							"name": "default-token-5z7xl",
							"secret": {
								"secretName": "default-token-5z7xl"
							}
						}
					],
					"containers": [
						{
							"name": "c7m",
							"image": "centos:7",
							"command": [
								"/bin/bash"
							],
							"args": [
								"-c",
								"trap \"killall sleep\" TERM; trap \"kill -9 sleep\" KILL; sleep infinity"
							],
							"resources": {},
							"volumeMounts": [
								{
									"name": "default-token-5z7xl",
									"readOnly": true,
									"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
								}
							],
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"imagePullPolicy": "IfNotPresent"
						}
					],
					"restartPolicy": "Always",
					"terminationGracePeriodSeconds": 30,
					"dnsPolicy": "ClusterFirst",
					"serviceAccountName": "default",
					"serviceAccount": "default",
					"securityContext": {},
					"schedulerName": "default-scheduler",
					"tolerations": [
						{
							"key": "node.kubernetes.io/not-ready",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						},
						{
							"key": "node.kubernetes.io/unreachable",
							"operator": "Exists",
							"effect": "NoExecute",
							"tolerationSeconds": 300
						}
					],
					"priority": 0,
					"enableServiceLinks": true
				},
				"status": {}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1"
			}
		}
	}`
	response, err := mutator.Mutate([]byte(rawJSON))
	if err != nil {
		t.Errorf("failed to mutate AdmissionRequest %s with error %s", string(response), err)
	}

	r := v1beta1.AdmissionReview{}
	err = json.Unmarshal(response, &r)
	assert.NoError(t, err, "failed to unmarshal with error %s", err)

	rr := r.Response
	assert.Equal(t, `[]`, string(rr.Patch))
	assert.Contains(t, rr.AuditAnnotations, "k8s-image-autoproxy")
}

func TestErrorsOnInvalidJson(t *testing.T) {
	rawJSON := `Wut ?`
	_, err := mutator.Mutate([]byte(rawJSON))
	if err == nil {
		t.Error("did not fail when sending invalid json")
	}
}

func TestErrorsOnInvalidPod(t *testing.T) {
	rawJSON := `{
		"request": {
			"object": 111
		}
	}`
	_, err := mutator.Mutate([]byte(rawJSON))
	if err == nil {
		t.Error("did not fail when sending invalid pod")
	}
}
