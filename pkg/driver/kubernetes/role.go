package kubernetes

// func (k *kubernetesDriver) createServiceAccount(ctx context.Context, id string) error {
// 	if k.config.ClusterRole == "" {
// 		return nil
// 	}

// 	// create service account
// 	if k.config.ServiceAccount == "" {
// 		serviceAccountRaw, err := json.Marshal(&corev1.ServiceAccount{
// 			TypeMeta: metav1.TypeMeta{
// 				Kind:       "ServiceAccount",
// 				APIVersion: corev1.SchemeGroupVersion.String(),
// 			},
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: id,
// 			},
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		k.Log.Infof("Create Service Account '%s'", id)
// 		buf := &bytes.Buffer{}
// 		err = k.runCommand(ctx, []string{"create", "-f", "-"}, bytes.NewReader(serviceAccountRaw), buf, buf)
// 		if err != nil {
// 			return errors.Wrapf(err, "create service account: %s", buf.String())
// 		}
// 	}

// 	// which service account?
// 	serviceAccount := id
// 	if k.config.ServiceAccount != "" {
// 		serviceAccount = k.config.ServiceAccount
// 	}

// 	// create role binding
// 	roleBindingRaw, err := json.Marshal(&rbacv1.RoleBinding{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "RoleBinding",
// 			APIVersion: rbacv1.SchemeGroupVersion.String(),
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: id,
// 		},
// 		Subjects: []rbacv1.Subject{
// 			{
// 				Kind: "ServiceAccount",
// 				Name: serviceAccount,
// 			},
// 		},
// 		RoleRef: rbacv1.RoleRef{
// 			APIGroup: rbacv1.SchemeGroupVersion.Group,
// 			Kind:     "ClusterRole",
// 			Name:     k.config.ClusterRole,
// 		},
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	k.Log.Infof("Create Role Binding '%s'", id)
// 	buf := &bytes.Buffer{}
// 	err = k.runCommand(ctx, []string{"create", "-f", "-"}, bytes.NewReader(roleBindingRaw), buf, buf)
// 	if err != nil {
// 		return errors.Wrapf(err, "create role binding: %s", buf.String())
// 	}

// 	return nil
// }
