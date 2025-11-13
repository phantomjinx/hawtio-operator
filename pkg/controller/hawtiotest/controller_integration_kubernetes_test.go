//go:build integration

package hawtiotest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hawtiov2 "github.com/hawtio/hawtio-operator/pkg/apis/hawtio/v2"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Testing the Hawtio Controller", func() {
	var mgrState *ManagerState

	Context("on Kubernetes", func() {

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

		It("Should handle empty type Hawtio CR", func() {
			performEmptyTypeHawtioCR(mgrState.ctx)
		})

		It("Should create expected common resources", func() {
			performCommonResourceTest(mgrState.ctx, "Kubernetes")
		})

		It("Should create ingress", func() {
			By("Creating a new Hawtio CR")
			hawtioKey := types.NamespacedName{Name: HawtioName, Namespace: HawtioNamespace}
			hawtio := &hawtiov2.Hawtio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hawtioKey.Name,
					Namespace: hawtioKey.Namespace,
				},
				Spec: hawtiov2.HawtioSpec{
					Type:    hawtiov2.NamespaceHawtioDeploymentType,
					Version: "latest",
				},
			}
			Expect(testTools.k8sClient.Create(mgrState.ctx, hawtio)).To(Succeed())

			By("Waiting for Ingress to be created")
			ingressKey := types.NamespacedName{Name: HawtioName, Namespace: HawtioNamespace}
			Eventually(func(g Gomega) {
				ingress := &networkingv1.Ingress{}
				g.Expect(testTools.k8sClient.Get(mgrState.ctx, ingressKey, ingress)).To(Succeed())
			}, timeout, interval).Should(Succeed())
		})
	})
})
