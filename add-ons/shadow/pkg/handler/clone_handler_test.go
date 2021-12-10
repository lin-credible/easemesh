package handler

import (
	"testing"
)

func TestShadowServiceCloner_Clone(t *testing.T) {

	cloner := &ShadowServiceCloner{
		KubeClient: prepareClientForTest(),
	}

	shadowService := fakeShadowService()
	sourceDeployment := fakeSourceDeployment()

	serviceCloneBlock := ServiceCloneBlock{
		service:   shadowService,
		deployObj: sourceDeployment,
	}
	cloner.Clone(serviceCloneBlock)
}