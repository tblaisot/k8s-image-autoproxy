package mutate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	AnnotationEnabled = "k8s-image-autoproxy.enabled"
	AnnotationProxy   = "k8s-image-autoproxy.proxy"
)

type Config struct {
	DefaultProxy  string
	DefaultEnable bool
	Verbose       bool
}

type imageProxyMutator struct {
	config    Config
	clientset kubernetes.Interface
}

type ImageProxyMutator interface {
	Mutate(body []byte) ([]byte, error)
}

func NewImageProxyMutator(config Config) ImageProxyMutator {
	log.Println(config)
	// creates the in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		panic(err.Error())
	}

	return imageProxyMutator{config: config, clientset: clientset}
}

type Options struct {
	enabled bool
	proxy   string
}

func (m imageProxyMutator) GetNamespaceOptions(namespaceName string) (options Options, err error) {
	if m.config.Verbose {
		log.Printf("GetNamespaceOptions: %s", namespaceName)
	}
	namespace, err := m.clientset.CoreV1().Namespaces().Get(context.TODO(), namespaceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	annotationEnabled := m.config.DefaultEnable
	if val, ok := namespace.GetAnnotations()["k8s-image-autoproxy.enabled"]; ok {
		annotationEnabled = val != "false"
	}
	annotationProxy := m.config.DefaultProxy
	if val, ok := namespace.GetAnnotations()["k8s-image-autoproxy.proxy"]; ok && val != "" {
		annotationProxy = val
	}
	if m.config.Verbose {
		log.Printf("Namespace: %s, enabled: %t, proxy: %s\n", namespaceName, annotationEnabled, annotationProxy) // untested section
	}
	options = Options{enabled: annotationEnabled, proxy: annotationProxy}
	return
}

// Mutate mutates
func (m imageProxyMutator) Mutate(body []byte) ([]byte, error) {

	if m.config.Verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := admissionv1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	responseBody := []byte{}
	ar := admReview.Request
	resp := admissionv1beta1.AdmissionResponse{}

	// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
	// tell K8S how it should modifiy it
	p := []map[string]string{}

	if ar != nil {
		switch ar.Kind.Kind {
		case "Pod":
			var pod *corev1.Pod
			if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
				return nil, fmt.Errorf("unable unmarshal Pod json object %v", err)
			}
			options, err := m.GetNamespaceOptions(pod.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, pod.Spec, "/spec", p)
		case "ReplicationController":
			var replicationController *corev1.ReplicationController
			if err := json.Unmarshal(ar.Object.Raw, &replicationController); err != nil {
				return nil, fmt.Errorf("unable unmarshal ReplicationController json object %v", err)
			}
			options, err := m.GetNamespaceOptions(replicationController.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, replicationController.Spec.Template.Spec, "/spec/template/spec", p)
		case "Deployment":
			var deployment *appsv1.Deployment
			if err := json.Unmarshal(ar.Object.Raw, &deployment); err != nil {
				return nil, fmt.Errorf("unable unmarshal Deployment json object %v", err)
			}
			options, err := m.GetNamespaceOptions(deployment.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, deployment.Spec.Template.Spec, "/spec/template/spec", p)
		case "ReplicaSet":
			var replicaSet *appsv1.ReplicaSet
			if err := json.Unmarshal(ar.Object.Raw, &replicaSet); err != nil {
				return nil, fmt.Errorf("unable unmarshal ReplicaSet json object %v", err)
			}
			options, err := m.GetNamespaceOptions(replicaSet.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, replicaSet.Spec.Template.Spec, "/spec/template/spec", p)
		case "DaemonSet":
			var daemonSet *appsv1.DaemonSet
			if err := json.Unmarshal(ar.Object.Raw, &daemonSet); err != nil {
				return nil, fmt.Errorf("unable unmarshal DaemonSet json object %v", err)
			}
			options, err := m.GetNamespaceOptions(daemonSet.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, daemonSet.Spec.Template.Spec, "/spec/template/spec", p)
		case "StatefulSet":
			var statefulSet *appsv1.StatefulSet
			if err := json.Unmarshal(ar.Object.Raw, &statefulSet); err != nil {
				return nil, fmt.Errorf("unable unmarshal StatefulSet json object %v", err)
			}
			options, err := m.GetNamespaceOptions(statefulSet.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, statefulSet.Spec.Template.Spec, "/spec/template/spec", p)
		case "CronJob":
			var cronJob *batchv1beta1.CronJob
			if err := json.Unmarshal(ar.Object.Raw, &cronJob); err != nil {
				return nil, fmt.Errorf("unable unmarshal CronJob json object %v", err)
			}
			options, err := m.GetNamespaceOptions(cronJob.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, cronJob.Spec.JobTemplate.Spec.Template.Spec, "/spec/jobTemplate/spec/template/spec", p)
		case "Job":
			var job *batchv1.Job
			if err := json.Unmarshal(ar.Object.Raw, &job); err != nil {
				return nil, fmt.Errorf("unable unmarshal Job json object %v", err)
			}
			options, err := m.GetNamespaceOptions(job.Namespace)
			if err != nil {
				return nil, err
			}
			p = proxyPodSpec(options, job.Spec.Template.Spec, "/spec/template/spec", p)
		default:
			return nil, fmt.Errorf("object not supported: %s", ar.Kind.Kind)
		}

		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := admissionv1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"k8s-image-autoproxy": "true",
		}

		// parse the []map into JSON
		var err error
		resp.Patch, err = json.Marshal(p)

		if err != nil {
			return nil, err // untested section
		}

		// Success, of course ;)
		resp.Result = &metav1.Status{
			Status: "Success",
		}

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err // untested section
		}
	}

	if m.config.Verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
