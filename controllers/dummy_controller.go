/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	interviewv1alpha1 "gitlab.com/rfashwal/dummy-controller/api/v1alpha1"
)

// DummyReconciler reconciles a Dummy object
type DummyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=interview.interview.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=interview.interview.com,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=interview.interview.com,resources=dummies/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dummy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dummy interviewv1alpha1.Dummy
	if err := r.Get(ctx, req.NamespacedName, &dummy); err != nil {
		if kerrors.IsNotFound(err) {
			logger.Info("Dummy resource not found.", "name", dummy.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch Dummy", "name", dummy.Name)
		return ctrl.Result{}, err
	}

	logger.Info("Processing Dummy", "name", dummy.Name, "namespace", dummy.Namespace, "message", dummy.Spec.ForProvider.Message)

	// Copy spec.message to status.specEcho and update sttus
	dummy.Status.AtProvider.SpecEcho = dummy.Spec.ForProvider.Message
	err := r.Status().Update(ctx, &dummy)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check pod if exists
	pod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Namespace: dummy.Namespace, Name: dummy.Name}, pod)
	if kerrors.IsNotFound(err) {
		// create a new Pod when pod not found
		logger.Info("Creating Pod", "name", pod.Name, "namespace", pod.Namespace)
		pod, err = r.createPod(ctx, &dummy)
		if err != nil {
			return ctrl.Result{}, err
		}

		// update the Dummy status with the podStatus
		dummy.Status.AtProvider.PodStatus = string(pod.Status.Phase)
		err = r.Status().Update(ctx, &dummy)
		if err != nil {
			return ctrl.Result{}, err
		}
		// pod is created succesfully, return and requeue request
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	podStatus := string(pod.Status.Phase)
	logger.Info("Pod fetched successfully", "name", pod.Name, "namespace", pod.Namespace, "status", podStatus)

	// pod already exists, check its status and update the dummy podStatus
	dummy.Status.AtProvider.PodStatus = podStatus
	err = r.Status().Update(ctx, &dummy)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&interviewv1alpha1.Dummy{}).
		//set up dummy controller to watch for changes
		//in the pod that is owned by its resource.
		Owns(&corev1.Pod{}).
		Complete(r)
}

// createPod create a new pod with nginx image, and associate it with the dummy object
func (r *DummyReconciler) createPod(ctx context.Context, dummy *interviewv1alpha1.Dummy) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dummy.Name,
			Namespace: dummy.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(dummy, pod, r.Scheme); err != nil {
		return nil, err
	}

	err := r.Create(ctx, pod)
	if err != nil {
		return nil, err
	}

	return pod, nil
}
