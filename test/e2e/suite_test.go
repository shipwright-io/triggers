package e2e

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shipwright-io/triggers/test/e2e/framework"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kubeClient  client.Client
	testContext context.Context
	cancel      context.CancelFunc
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	rest, err := ctrl.GetConfig()
	Expect(err).NotTo(HaveOccurred())

	kubeClient, err = client.New(rest, client.Options{
		Scheme: scheme,
	})
	Expect(err).NotTo(HaveOccurred())
	testContext, cancel = context.WithCancel(context.Background())
})

var _ = BeforeEach(func() {
	By("waiting for project deployment")
	err := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
		return framework.IsDeploymentReady(testContext, kubeClient)
	})
	Expect(err).NotTo(HaveOccurred())

})

var _ = When("the controller has been deployed", func() {
	It("should be ready", func() {
		ready, err := framework.IsDeploymentReady(testContext, kubeClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).To(BeTrue())
	})
})

var _ = AfterSuite(func() {
	cancel()
})
