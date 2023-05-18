/*
Copyright 2018 The Kubernetes Authors.
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

package serviceaccount

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	// Injected bound service account token expiration which triggers monitoring of its time-bound feature.
	WarnOnlyBoundTokenExpirationSeconds = 60*60 + 7

	// Extended expiration for those modified tokens involved in safe rollout if time-bound feature.
	ExpirationExtensionSeconds = 24 * 365 * 60 * 60
)

// time.Now stubbed out to allow testing
var now = time.Now

type privateClaims struct {
	Kubernetes kubernetes `json:"kubernetes.io,omitempty"`
}

type kubernetes struct {
	Namespace string          `json:"namespace,omitempty"`
	Svcacct   ref             `json:"serviceaccount,omitempty"`
	Pod       *ref            `json:"pod,omitempty"`
	Secret    *ref            `json:"secret,omitempty"`
	WarnAfter jwt.NumericDate `json:"warnafter,omitempty"`
}

type ref struct {
	Name string `json:"name,omitempty"`
	UID  string `json:"uid,omitempty"`
}

func Claims(sa corev1.ServiceAccount, pod *corev1.Pod, secret *corev1.Secret, expirationSeconds, warnafter int64, audience []string) (*jwt.Claims, interface{}) {
	now := now()
	sc := &jwt.Claims{
		Subject:   "system:serviceaccount:" + sa.Namespace + ":" + sa.Name,
		Audience:  jwt.Audience(audience),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Expiry:    jwt.NewNumericDate(now.Add(time.Duration(expirationSeconds) * time.Second)),
	}
	pc := &privateClaims{
		Kubernetes: kubernetes{
			Namespace: sa.Namespace,
			Svcacct: ref{
				Name: sa.Name,
				UID:  string(sa.UID),
			},
		},
	}
	switch {
	case pod != nil:
		pc.Kubernetes.Pod = &ref{
			Name: pod.Name,
			UID:  string(pod.UID),
		}
	case secret != nil:
		pc.Kubernetes.Secret = &ref{
			Name: secret.Name,
			UID:  string(secret.UID),
		}
	}

	if warnafter != 0 {
		pc.Kubernetes.WarnAfter = *jwt.NewNumericDate(now.Add(time.Duration(warnafter) * time.Second))
	}

	return sc, pc
}
