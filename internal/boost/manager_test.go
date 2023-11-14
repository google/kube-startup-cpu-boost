package boost_test

import (
	"context"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	cpuboost "github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Manager", func() {
	var manager cpuboost.Manager
	Describe("Registers startup-cpu-boost", func() {
		var (
			spec  *autoscaling.StartupCPUBoost
			boost cpuboost.StartupCPUBoost
			err   error
		)
		BeforeEach(func() {
			spec = specTemplate.DeepCopy()
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManager(nil)
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec)
			Expect(err).ToNot(HaveOccurred())
		})
		When("startup-cpu-boost exists", func() {
			JustBeforeEach(func() {
				err = manager.AddStartupCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
				err = manager.AddStartupCPUBoost(context.TODO(), boost)
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("startup-cpu-boost does not exist", func() {
			JustBeforeEach(func() {
				err = manager.AddStartupCPUBoost(context.TODO(), boost)
			})
			It("does not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("stores the startup-cpu-boost", func() {
				stored, ok := manager.StartupCPUBoost(spec.Namespace, spec.Name)
				Expect(ok).To(BeTrue())
				Expect(stored.Name()).To(Equal(spec.Name))
				Expect(stored.Namespace()).To(Equal(spec.Namespace))
			})
		})

	})
	Describe("De-registers startup-cpu-boost", func() {
		var (
			spec  *autoscaling.StartupCPUBoost
			boost cpuboost.StartupCPUBoost
			err   error
		)
		BeforeEach(func() {
			spec = specTemplate.DeepCopy()
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManager(nil)
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec)
			Expect(err).ToNot(HaveOccurred())
		})
		When("startup-cpu-boost exists", func() {
			JustBeforeEach(func() {
				err = manager.AddStartupCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
				manager.RemoveStartupCPUBoost(context.TODO(), boost.Namespace(), boost.Name())
			})
			It("removes the startup-cpu-boost", func() {
				_, ok := manager.StartupCPUBoost(spec.Namespace, spec.Name)
				Expect(ok).To(BeFalse())
			})
		})
	})
	Describe("retrieves startup-cpu-boost for a POD", func() {
		var (
			pod               *corev1.Pod
			podNameLabel      string
			podNameLabelValue string
			boost             cpuboost.StartupCPUBoost
			found             bool
		)
		BeforeEach(func() {
			podNameLabel = "app.kubernetes.io/name"
			podNameLabelValue = "app-001"
			pod = podTemplate.DeepCopy()
			pod.Labels[podNameLabel] = podNameLabelValue
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManager(nil)
		})
		When("matching startup-cpu-boost does not exist", func() {
			JustBeforeEach(func() {
				boost, found = manager.StartupCPUBoostForPod(context.TODO(), pod)
			})
			It("returns false", func() {
				Expect(found).To(BeFalse())
			})
			It("return nil", func() {
				Expect(boost).To(BeNil())
			})
		})
		When("matching startup-cpu-boost exists", func() {
			var (
				spec *autoscaling.StartupCPUBoost
				err  error
			)
			BeforeEach(func() {
				spec = specTemplate.DeepCopy()
				spec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{}, podNameLabel, podNameLabelValue)
			})
			JustBeforeEach(func() {
				boost, err = cpuboost.NewStartupCPUBoost(nil, spec)
				Expect(err).NotTo(HaveOccurred())
				err = manager.AddStartupCPUBoost(context.TODO(), boost)
				Expect(err).NotTo(HaveOccurred())
				boost, found = manager.StartupCPUBoostForPod(context.TODO(), pod)
			})
			It("returns true", func() {
				Expect(found).To(BeTrue())
			})
			It("returns valid boost", func() {
				Expect(boost).NotTo(BeNil())
				Expect(boost.Name()).To(Equal(spec.Name))
				Expect(boost.Namespace()).To(Equal(spec.Namespace))
			})
		})
	})
	Describe("Runs on a time tick", func() {
		var (
			mockCtrl   *gomock.Controller
			mockTicker *mock.MockTimeTicker
			ctx        context.Context
			cancel     context.CancelFunc
			err        error
			done       chan int
		)
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockTicker = mock.NewMockTimeTicker(mockCtrl)
			ctx, cancel = context.WithCancel(context.TODO())
			done = make(chan int)
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManagerWithTicker(nil, mockTicker)
			go func() {
				defer GinkgoRecover()
				err = manager.Start(ctx)
				done <- 1
			}()
		})
		When("There are no startup-cpu-boosts with fixed duration policy", func() {
			var c chan time.Time
			BeforeEach(func() {
				c = make(chan time.Time, 1)
				mockTicker.EXPECT().Tick().MinTimes(1).Return(c)
				mockTicker.EXPECT().Stop().Return()
			})
			JustBeforeEach(func() {
				c <- time.Now()
				time.Sleep(500 * time.Millisecond)
				cancel()
				<-done
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("There are startup-cpu-boosts with fixed duration policy", func() {
			var (
				spec       *autoscaling.StartupCPUBoost
				boost      cpuboost.StartupCPUBoost
				pod        *corev1.Pod
				mockClient *mock.MockClient
				c          chan time.Time
			)
			BeforeEach(func() {
				spec = specTemplate.DeepCopy()
				var seconds int64 = 60
				spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
					Unit:  autoscaling.FixedDurationPolicyUnitSec,
					Value: seconds,
				}
				pod = podTemplate.DeepCopy()
				creationTimestamp := time.Now().Add(-1 * time.Duration(seconds) * time.Second).Add(-1 * time.Minute)
				pod.CreationTimestamp = metav1.NewTime(creationTimestamp)
				mockClient = mock.NewMockClient(mockCtrl)

				c = make(chan time.Time, 1)
				mockTicker.EXPECT().Tick().MinTimes(1).Return(c)
				mockTicker.EXPECT().Stop().Return()
				mockClient.EXPECT().Update(gomock.Any(), gomock.Eq(pod)).MinTimes(1).Return(nil)
			})
			JustBeforeEach(func() {
				boost, err = cpuboost.NewStartupCPUBoost(mockClient, spec)
				Expect(err).ShouldNot(HaveOccurred())
				err = boost.UpsertPod(ctx, pod)
				Expect(err).ShouldNot(HaveOccurred())
				err = manager.AddStartupCPUBoost(context.TODO(), boost)
				Expect(err).ShouldNot(HaveOccurred())

				c <- time.Now()
				time.Sleep(500 * time.Millisecond)
				cancel()
				<-done
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
