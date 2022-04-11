/*
Copyright 2022 The Kubernetes Authors.

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

package config

import (
	"context"

	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client"

	nrtv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"

	nropv1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
)

type E2EConfig struct {
	Fixture       *e2efixture.Fixture
	NRTList       nrtv1alpha1.NodeResourceTopologyList
	NROOperObj    *nropv1alpha1.NUMAResourcesOperator
	NROSchedObj   *nropv1alpha1.NUMAResourcesScheduler
	SchedulerName string
}

func (cfg *E2EConfig) Ready() bool {
	if cfg == nil {
		return false
	}
	if cfg.Fixture == nil || cfg.NROOperObj == nil || cfg.NROSchedObj == nil {
		return false
	}
	if cfg.SchedulerName == "" {
		return false
	}
	return true
}

var Config *E2EConfig

func SetupFixture() error {
	var err error
	Config, err = NewFixtureWithOptions("e2e-test-infra", e2efixture.OptionRandomizeName)
	return err
}

func TeardownFixture() error {
	return e2efixture.Teardown(Config.Fixture)
}

func NewFixtureWithOptions(nsName string, options e2efixture.Options) (*E2EConfig, error) {
	var err error
	cfg := E2EConfig{
		NROOperObj:  &nropv1alpha1.NUMAResourcesOperator{},
		NROSchedObj: &nropv1alpha1.NUMAResourcesScheduler{},
	}

	cfg.Fixture, err = e2efixture.SetupWithOptions(nsName, options)
	if err != nil {
		return nil, err
	}

	err = cfg.Fixture.Client.List(context.TODO(), &cfg.NRTList)
	if err != nil {
		return nil, err
	}

	err = cfg.Fixture.Client.Get(context.TODO(), client.ObjectKey{Name: objects.NROName()}, cfg.NROOperObj)
	if err != nil {
		return nil, err
	}

	err = cfg.Fixture.Client.Get(context.TODO(), client.ObjectKey{Name: nrosched.NROSchedObjectName}, cfg.NROSchedObj)
	if err != nil {
		return nil, err
	}

	cfg.SchedulerName = cfg.NROSchedObj.Status.SchedulerName
	klog.Infof("detected scheduler name: %q", cfg.SchedulerName)

	return &cfg, nil
}
