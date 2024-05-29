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
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	interviewv1alpha1 "gitlab.com/rfashwal/dummy-controller/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = interviewv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func TestDummyReconciler(t *testing.T) {
	// init a dummy resource
	dummy := &interviewv1alpha1.Dummy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy-test",
			Namespace: "default",
		},
		Spec: interviewv1alpha1.DummySpec{
			//Message: "Hello, k8s operators!",
		},
	}

	objs := []client.Object{dummy}

	// register dummy schema
	s := scheme.Scheme
	s.AddKnownTypes(interviewv1alpha1.SchemeBuilder.GroupVersion, dummy)

	// build a fake client to mock api
	builder := fake.NewClientBuilder()
	cl := builder.WithObjects(objs...).Build()

	// init a dummy reconciler with the fake client and scheme
	r := &DummyReconciler{Client: cl, Scheme: s}

	// a dummy request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      dummy.Name,
			Namespace: dummy.Namespace,
		},
	}

	// reconcile the Dummy resource
	ctx := context.TODO()
	_, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Errorf("reconcile error: %v", err)
	}

	// check the request is successful
	err = cl.Get(ctx, types.NamespacedName{Name: dummy.Name, Namespace: dummy.Namespace}, dummy)
	if err != nil {
		t.Fatalf("failed to get Dummy resource: %v", err)
	}

	// assert message and echo are equal
	if dummy.Status.AtProvider.SpecEcho != dummy.Spec.ForProvider.Message {
		t.Errorf("unexpected SpecEcho value: want=%s, got=%s", dummy.Spec.ForProvider.Message, dummy.Status.AtProvider.SpecEcho)
	}

	// validate pod creation
	createdPod := &corev1.Pod{}
	err = cl.Get(ctx, types.NamespacedName{Namespace: dummy.Namespace, Name: dummy.Name}, createdPod)
	if err != nil {
		t.Fatalf("failed to get created Pod: %v", err)
	}

	// assert pod status matches dummy podStatus
	if string(createdPod.Status.Phase) != dummy.Status.AtProvider.PodStatus {
		t.Errorf("unexpected Pod status: want=%s, got=%s", dummy.Status.AtProvider.PodStatus, string(createdPod.Status.Phase))
	}
}
