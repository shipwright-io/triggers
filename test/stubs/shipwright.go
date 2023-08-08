// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package stubs

import (
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Namespace             = "default"
	Branch                = "main"
	PipelineNameInTrigger = "pipeline"
)

var (
	// TriggerWhenPushToMain describes a trigger for a github push event on default branch.
	TriggerWhenPushToMain = v1alpha1.TriggerWhen{
		Type: v1alpha1.GitHubWebHookTrigger,
		GitHub: &v1alpha1.WhenGitHub{
			Events: []v1alpha1.GitHubEventName{
				v1alpha1.GitHubPushEvent,
			},
			Branches: []string{Branch},
		},
	}
	// TriggerWhenPipelineSucceeded describes a trigger for Tekton Pipeline on status "succeeded".
	TriggerWhenPipelineSucceeded = v1alpha1.TriggerWhen{
		Type: v1alpha1.PipelineTrigger,
		ObjectRef: &v1alpha1.WhenObjectRef{
			Name:     PipelineNameInTrigger,
			Status:   []string{"Succeeded"},
			Selector: map[string]string{},
		},
	}
)

// ShipwrightBuild returns a Build using informed output image base and name.
func ShipwrightBuild(outputImageBase, name string) *v1alpha1.Build {
	strategyKind := v1alpha1.BuildStrategyKind("ClusterBuildStrategy")
	contextDir := "source-build"

	return &v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: v1alpha1.BuildSpec{
			Strategy: v1alpha1.Strategy{
				Kind: &strategyKind,
				Name: "buildpacks-v3",
			},
			Source: v1alpha1.Source{
				URL:        &RepoURL,
				ContextDir: &contextDir,
			},
			Output: v1alpha1.Image{
				Image: fmt.Sprintf("%s/%s:latest", outputImageBase, name),
			},
		},
	}
}

// ShipwrightBuildWithTriggers creates a Build with optional triggers.
func ShipwrightBuildWithTriggers(
	outputImageBase,
	name string,
	triggers ...v1alpha1.TriggerWhen,
) *v1alpha1.Build {
	b := ShipwrightBuild(outputImageBase, name)
	b.Spec.Trigger = &v1alpha1.Trigger{When: triggers}
	return b
}

// ShipwrightBuildRun returns a empty BuildRun instance using informed name.
func ShipwrightBuildRun(name string) *v1alpha1.BuildRun {
	return &v1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
	}
}
