package boost_test

import (
	"context"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	cpuboost "github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/boost/policy"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("StartupCPUBoost", func() {
	var (
		spec  *autoscaling.StartupCPUBoost
		boost cpuboost.StartupCPUBoost
		err   error
		pod   *corev1.Pod
	)
	BeforeEach(func() {
		pod = podTemplate.DeepCopy()
		spec = specTemplate.DeepCopy()
	})
	Describe("Instantiates from the API specification", func() {
		JustBeforeEach(func() {
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("returns valid name", func() {
			Expect(boost.Name()).To(Equal(spec.Name))
		})
		It("returns valid namespace", func() {
			Expect(boost.Namespace()).To(Equal(spec.Namespace))
		})
		It("returns valid boost percent", func() {
			Expect(boost.BoostPercent()).To(Equal(spec.Spec.BoostPercent))
		})
		When("the spec has fixed duration policy", func() {
			BeforeEach(func() {
				spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
					Unit:  autoscaling.FixedDurationPolicyUnitSec,
					Value: 123,
				}
			})
			It("returns fixed duration policy implementation", func() {
				Expect(boost.DurationPolicies()).To(HaveKey(policy.FixedDurationPolicyName))
			})
			It("returned fixed duration policy implementation is valid", func() {
				p := boost.DurationPolicies()[policy.FixedDurationPolicyName]
				fixedP, ok := p.(*policy.FixedDurationPolicy)
				Expect(ok).To(BeTrue())
				expDuration := time.Duration(spec.Spec.DurationPolicy.Fixed.Value) * time.Second
				Expect(fixedP.Duration()).To(Equal(expDuration))
			})
		})
		When("the spec has pod condition duration policy", func() {
			BeforeEach(func() {
				spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
					Unit:  autoscaling.FixedDurationPolicyUnitSec,
					Value: 123,
				}
				spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				}
			})
			It("returns pod condition duration policy implementation", func() {
				Expect(boost.DurationPolicies()).To(HaveKey(policy.PodConditionPolicyName))
			})
			It("returned pod condition duration policy implementation is valid", func() {
				p := boost.DurationPolicies()[policy.PodConditionPolicyName]
				podCondP, ok := p.(*policy.PodConditionPolicy)
				Expect(ok).To(BeTrue())
				Expect(podCondP.Condition()).To(Equal(spec.Spec.DurationPolicy.PodCondition.Type))
				Expect(podCondP.Status()).To(Equal(spec.Spec.DurationPolicy.PodCondition.Status))
			})
		})
	})

	Describe("Upserts a POD", func() {
		var (
			mockCtrl   *gomock.Controller
			mockClient *mock.MockClient
		)
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
		})
		JustBeforeEach(func() {
			boost, err = cpuboost.NewStartupCPUBoost(mockClient, spec)
			Expect(err).ShouldNot(HaveOccurred())
		})
		When("POD does not exist", func() {
			JustBeforeEach(func() {
				err = boost.UpsertPod(context.TODO(), pod)
			})
			It("doesn't error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("stores a POD", func() {
				p, ok := boost.Pod(pod.Name)
				Expect(ok).To(BeTrue())
				Expect(p.Name).To(Equal(pod.Name))
			})
		})
		When("POD exists", func() {
			var existingPod *corev1.Pod
			var createTimestamp metav1.Time
			BeforeEach(func() {
				existingPod = podTemplate.DeepCopy()
				createTimestamp = metav1.NewTime(time.Now())
				pod.CreationTimestamp = createTimestamp
			})
			JustBeforeEach(func() {
				err = boost.UpsertPod(context.TODO(), existingPod)
				Expect(err).ShouldNot(HaveOccurred())
				err = boost.UpsertPod(context.TODO(), pod)
			})
			It("doesn't error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("stores an updated POD", func() {
				p, found := boost.Pod(pod.Name)
				Expect(found).To(BeTrue())
				Expect(p.Name).To(Equal(pod.Name))
				Expect(p.CreationTimestamp).To(Equal(createTimestamp))
			})
			When("boost spec has pod condition policy", func() {
				BeforeEach(func() {
					spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					}
				})
				When("POD condition matches spec policy", func() {
					BeforeEach(func() {
						pod.Status.Conditions = []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}}
						mockClient.EXPECT().
							Update(gomock.Any(), gomock.Eq(pod)).
							Return(nil)
					})
					It("doesn't error", func() {
						Expect(err).NotTo(HaveOccurred())
					})
				})
				When("POD condition does not match spec policy", func() {
					BeforeEach(func() {
						pod.Status.Conditions = []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						}}
					})
					It("doesn't error", func() {
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
		})
	})

	Describe("Deletes a pod", func() {
		JustBeforeEach(func() {
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec)
			Expect(err).ShouldNot(HaveOccurred())
		})
		When("Pod exists", func() {
			JustBeforeEach(func() {
				err = boost.UpsertPod(context.TODO(), pod)
				Expect(err).ShouldNot(HaveOccurred())
				err = boost.DeletePod(context.TODO(), pod)
			})
			It("removes stored pod", func() {
				_, found := boost.Pod(pod.Name)
				Expect(found).To(BeFalse())
			})
		})
	})
})
