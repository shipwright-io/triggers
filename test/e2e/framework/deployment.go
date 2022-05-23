package framework

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsDeploymentReady(ctx context.Context, c client.Client) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Namespace: "shipwright-build", Name: "shipwright-triggers-controller"}, deployment)
	if err != nil {
		return false, err
	}
	for _, condition := range deployment.Status.Conditions {
		if condition.Type != appsv1.DeploymentAvailable {
			continue
		}
		return condition.Status == corev1.ConditionTrue, nil
	}
	return false, nil
}
