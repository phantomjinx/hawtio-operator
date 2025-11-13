//go:build integration

package hawtiotest

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/hawtio/hawtio-operator/pkg/capabilities"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	kclient "k8s.io/client-go/kubernetes"
)

var _ = Describe("Testing the Hawtio Controller", Ordered, func() {
	var mgrState *ManagerState

	fakeCV := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}

	ocPublicNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "openshift-config-managed"},
	}

	fakeConsoleConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "console-public",
			Namespace: ocPublicNS.Name,
		},
		Data: map[string]string{
			"consoleURL": "https://console-openshift-console.apps-crc.testing",
		},
	}

	BeforeAll(func() {
		By("Creating fake ClusterVersion to enable OpenShift mode")
		// Create the cluster version to mark the cluster as Openshift
		Expect(testTools.k8sClient.Create(context.Background(), fakeCV)).To(Succeed())

		createdCV := &configv1.ClusterVersion{}
		Expect(testTools.k8sClient.Get(context.Background(), types.NamespacedName{Name: "version"}, createdCV)).To(Succeed())

		// 3. Set the Status field on the *fetched* object
		createdCV.Status.History = []configv1.UpdateHistory{
			{
				State:       configv1.CompletedUpdate,
				Version:     "4.19.0",
				StartedTime: metav1.Now(),
			},
		}

		// Use the .Status().Update() client to apply the status
		By("Updating fake ClusterVersion's status")
		Expect(testTools.k8sClient.Status().Update(context.Background(), createdCV)).To(Succeed())

		By("Creating fake OpenShift system namespace")
		Expect(testTools.k8sClient.Create(context.Background(), ocPublicNS)).To(Succeed())

		By("Creating fake 'console-public' ConfigMap in system namespace")
		Expect(testTools.k8sClient.Create(context.Background(), fakeConsoleConfig)).To(Succeed())
	})

	AfterAll(func() {
		By("Deleting the ClusterVersion object")
		// Remove the object from the API server's memory
		Expect(testTools.k8sClient.Delete(context.Background(), fakeCV)).To(Succeed())

		By("Cleaning up fake OpenShift system namespace")
		Expect(testTools.k8sClient.Delete(context.Background(), ocPublicNS)).To(Succeed())
	})

	Context("on OpenShift", func() {

		BeforeEach(func() {
			mgrState = startManager(testTools.cfg)

			// Used in preference to AfterEach since
			// DeferCleanup executes right at the end of the test
			// and has a LIFO stack so any other cleanups added
			// in tests will be executed before this one.
			DeferCleanup(func() {
				By("Deleting the Hawtio CR")
				performDeleteHawtioCR(HawtioName, HawtioNamespace)

				By("Stopping the Kubernetes manager")
				mgrState.cancel()  // Signal manager to stop
				mgrState.wg.Wait() // Wait for it to shut down
			})
		})

		It("Should correctly detect an OpenShift cluster", func() {
			By("Manually creating API clients")
			// Create the clients just like your controller's AddToManager does
			configClient, err := configclient.NewForConfig(testTools.cfg)
			Expect(err).NotTo(HaveOccurred())

			apiClient, err := kclient.NewForConfig(testTools.cfg)
			Expect(err).NotTo(HaveOccurred())

			By("Running APICapabilities check")
			// This runs the check *after* the BeforeEach created the fake ClusterVersion
			apiSpec, err := capabilities.APICapabilities(mgrState.ctx, apiClient, configClient)
			Expect(err).NotTo(HaveOccurred())

			By("Asserting OpenShift mode is enabled")
			// This is the direct assertion you wanted
			Expect(apiSpec.IsOpenShift4).To(BeTrue())
			Expect(apiSpec.Routes).To(BeTrue())
		})

		It("Should handle empty type Hawtio CR", func() {
			performEmptyTypeHawtioCR(mgrState.ctx)
		})

		It("Should create expected common resources", func() {
			performCommonResourceTest(mgrState.ctx, "OpenShift")
		})

		It("Should create route", func() {
			// By("Creating a new Hawtio CR")
			// hawtioKey := types.NamespacedName{Name: HawtioName, Namespace: HawtioNamespace}
			// hawtio := &hawtiov2.Hawtio{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Name:      hawtioKey.Name,
			// 		Namespace: hawtioKey.Namespace,
			// 	},
			// 	Spec: hawtiov2.HawtioSpec{
			// 		Type:    hawtiov2.NamespaceHawtioDeploymentType,
			// 		Version: "latest",
			// 	},
			// }
			// Expect(testTools.k8sClient.Create(mgrState.ctx, hawtio)).To(Succeed())
			//
			// By("Waiting for Ingress to be created")
			// ingressKey := types.NamespacedName{Name: HawtioName, Namespace: HawtioNamespace}
			// Eventually(func(g Gomega) {
			// 	ingress := &networkingv1.Ingress{}
			// 	g.Expect(testTools.k8sClient.Get(mgrState.ctx, ingressKey, ingress)).To(Succeed())
			// }, timeout, interval).Should(Succeed())
		})
	})
})
