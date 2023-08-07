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

// internal/controller/suite_test.go

package controller

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	adapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

// TestControllers function starts the suite of tests. This is an entry point to run the tests
func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	// RunSpecs runs all registered Specs. It's Ginkgo's equivalent to a testing package's Test function.
	// The description is used when printing out specs, it does not need to be unique
	RunSpecs(t, "Controller Suite")
}

// BeforeSuite is a Ginkgo function to execute some setup code before the suite of tests runs
var _ = BeforeSuite(func() {
	// Sets the logger to use Zap, which logs to standard output and uses development mode (pretty colored output).
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Bootstraps the test environment
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	// Starts the test environment
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Adds the schema of the NimbleOpti custom resource to the Scheme.
	// The scheme defines the mapping between Golang structs and Kubernetes API Groups, Versions, and Kinds.
	err = adapterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Initialize the client, it's used to interact with the Kubernetes API
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

// AfterSuite is a Ginkgo function to execute some cleanup code after the suite of tests has been run
var _ = AfterSuite(func() {
	// Tearing down the test environment
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
