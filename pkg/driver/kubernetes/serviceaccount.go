package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubernetesDriver) createServiceAccount(ctx context.Context, id, serviceAccount string) error {
	// try to find pvc
	_, err := k.client.Client().CoreV1().ServiceAccounts(k.namespace).Get(ctx, serviceAccount, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("get service account: %w", err)
	} else if kerrors.IsNotFound(err) {
		// create service account if it does not exist
		k.Log.Infof("Create Service Account '%s'", serviceAccount)
		_, err := k.client.Client().CoreV1().ServiceAccounts(k.namespace).Create(ctx, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:   serviceAccount,
				Labels: ExtraDevPodLabels,
			},
		}, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("create service account: %w", err)
		}
	}

	// try to find role binding
	if k.options.ClusterRole != "" {
		_, err := k.client.Client().RbacV1().RoleBindings(k.namespace).Get(ctx, id, metav1.GetOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("get role binding: %w", err)
		} else if kerrors.IsNotFound(err) {
			// create role binding
			k.Log.Infof("Create Role Binding '%s'", serviceAccount)
			_, err := k.client.Client().RbacV1().RoleBindings(k.namespace).Create(ctx, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:   id,
					Labels: ExtraDevPodLabels,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind: "ServiceAccount",
						Name: serviceAccount,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     "ClusterRole",
					Name:     k.options.ClusterRole,
				},
			}, metav1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				return fmt.Errorf("create role binding: %w", err)
			}
		}
	}

	return nil
}
