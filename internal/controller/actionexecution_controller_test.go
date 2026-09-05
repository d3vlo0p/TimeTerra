/*
Copyright 2024.

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

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	timeterrav1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
)

var _ = Describe("ActionExecution Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		actionexecution := &timeterrav1alpha1.ActionExecution{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ActionExecution")
			err := k8sClient.Get(ctx, typeNamespacedName, actionexecution)
			if err != nil && errors.IsNotFound(err) {
				resource := &timeterrav1alpha1.ActionExecution{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &timeterrav1alpha1.ActionExecution{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				_ = k8sClient.Delete(ctx, resource)
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ActionExecutionReconciler{
				BaseReconciler: BaseReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should timeout and delete the resource when ActionTimeout is exceeded", func() {
			By("Reconciling with an expired timeout")
			controllerReconciler := &ActionExecutionReconciler{
				BaseReconciler: BaseReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				},
				ActionTimeout: 1 * time.Nanosecond,
			}

			time.Sleep(10 * time.Millisecond)

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// The resource should have been deleted due to timeout
			deletedOp := &timeterrav1alpha1.ActionExecution{}
			err = k8sClient.Get(ctx, typeNamespacedName, deletedOp)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
