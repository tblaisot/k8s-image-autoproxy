package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Proxy   string
	Verbose bool
}

// Mutate mutates
func Mutate(body []byte, config Config) ([]byte, error) {
	if config.Verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := admission.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error

	responseBody := []byte{}
	ar := admReview.Request
	resp := admission.AdmissionResponse{}

	// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
	// tell K8S how it should modifiy it
	p := []map[string]string{}

	if ar != nil {
		switch ar.Kind.Kind {
		case "Pod":
			var pod *core.Pod
			if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
				return nil, fmt.Errorf("unable unmarshal Pod json object %v", err)
			}
			p = proxyPodSpec(pod.Spec, config.Proxy, "/spec", p)
		case "ReplicationController":
			var replicationController *core.ReplicationController
			if err := json.Unmarshal(ar.Object.Raw, &replicationController); err != nil {
				return nil, fmt.Errorf("unable unmarshal ReplicationController json object %v", err)
			}
			p = proxyPodSpec(replicationController.Spec.Template.Spec, config.Proxy, "/spec/template/spec", p)
		case "Container":
			var container *core.Container
			if err := json.Unmarshal(ar.Object.Raw, &container); err != nil {
				return nil, fmt.Errorf("unable unmarshal Container json object %v", err)
			}
		case "Deployment":
			var deployment *apps.Deployment
			if err := json.Unmarshal(ar.Object.Raw, &deployment); err != nil {
				return nil, fmt.Errorf("unable unmarshal Deployment json object %v", err)
			}
			p = proxyPodSpec(deployment.Spec.Template.Spec, config.Proxy, "/spec/template/spec", p)
		case "ReplicaSet":
			var replicaSet *apps.ReplicaSet
			if err := json.Unmarshal(ar.Object.Raw, &replicaSet); err != nil {
				return nil, fmt.Errorf("unable unmarshal ReplicaSet json object %v", err)
			}
			p = proxyPodSpec(replicaSet.Spec.Template.Spec, config.Proxy, "/spec/template/spec", p)
		case "DaemonSet":
			var daemonSet *apps.DaemonSet
			if err := json.Unmarshal(ar.Object.Raw, &daemonSet); err != nil {
				return nil, fmt.Errorf("unable unmarshal DaemonSet json object %v", err)
			}
			p = proxyPodSpec(daemonSet.Spec.Template.Spec, config.Proxy, "/spec/template/spec", p)
		case "StatefulSet":
			var statefulSet *apps.StatefulSet
			if err := json.Unmarshal(ar.Object.Raw, &statefulSet); err != nil {
				return nil, fmt.Errorf("unable unmarshal StatefulSet json object %v", err)
			}
			p = proxyPodSpec(statefulSet.Spec.Template.Spec, config.Proxy, "/spec/template/spec", p)
		case "CronJob":
			var cronJob *batchv1beta1.CronJob
			if err := json.Unmarshal(ar.Object.Raw, &cronJob); err != nil {
				return nil, fmt.Errorf("unable unmarshal CronJob json object %v", err)
			}
			p = proxyPodSpec(cronJob.Spec.JobTemplate.Spec.Template.Spec, config.Proxy, "/spec/jobTemplate/spec/template/spec", p)
		case "Job":
			var job *batchv1.Job
			if err := json.Unmarshal(ar.Object.Raw, &job); err != nil {
				return nil, fmt.Errorf("unable unmarshal Job json object %v", err)
			}
			p = proxyPodSpec(job.Spec.Template.Spec, config.Proxy, "/spec/template/spec", p)
		default:
			return nil, fmt.Errorf("object not supported: %s", ar.Kind.Kind)
		}

		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := admission.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"k8s-image-autoproxy": "true",
		}

		// parse the []map into JSON
		resp.Patch, err = json.Marshal(p)

		if err != nil {
			return nil, err // untested section
		}

		// Success, of course ;)
		resp.Result = &meta.Status{
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

	if config.Verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
