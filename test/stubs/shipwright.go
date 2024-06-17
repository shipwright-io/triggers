// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package stubs

import (
	"fmt"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Namespace             = "default"
	Branch                = "main"
	PipelineNameInTrigger = "pipeline"
)

var (
	// TriggerWhenPushToMain describes a trigger for a github push event on default branch.
	TriggerWhenPushToMain = buildapi.TriggerWhen{
		Type: buildapi.GitHubWebHookTrigger,
		GitHub: &buildapi.WhenGitHub{
			Events: []buildapi.GitHubEventName{
				buildapi.GitHubPushEvent,
			},
			Branches: []string{Branch},
		},
	}
	// TriggerWhenPipelineSucceeded describes a trigger for Tekton Pipeline on status "succeeded".
	TriggerWhenPipelineSucceeded = buildapi.TriggerWhen{
		Type: buildapi.PipelineTrigger,
		ObjectRef: &buildapi.WhenObjectRef{
			Name:     PipelineNameInTrigger,
			Status:   []string{"Succeeded"},
			Selector: map[string]string{},
		},
	}
)

// ShipwrightBuild returns a Build using informed output image base and name.
func ShipwrightBuild(outputImageBase, name string) *buildapi.Build {
	strategyKind := buildapi.BuildStrategyKind("ClusterBuildStrategy")
	contextDir := "source-build"

	return &buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: buildapi.BuildSpec{
			Strategy: buildapi.Strategy{
				Kind: &strategyKind,
				Name: "buildpacks-v3",
			},
			Source: &buildapi.Source{
				Type: buildapi.GitType,
				Git: &buildapi.Git{
					URL: RepoURL,
				},
				ContextDir: &contextDir,
			},
			Output: buildapi.Image{
				Image: fmt.Sprintf("%s/%s:latest", outputImageBase, name),
			},
		},
	}
}

// ShipwrightBuildWithTriggers creates a Build with optional triggers.
func ShipwrightBuildWithTriggers(
	outputImageBase,
	name string,
	triggers ...buildapi.TriggerWhen,
) *buildapi.Build {
	b := ShipwrightBuild(outputImageBase, name)
	b.Spec.Trigger = &buildapi.Trigger{When: triggers}
	return b
}

// ShipwrightBuildRun returns a empty BuildRun instance using informed name.
func ShipwrightBuildRun(name string) *buildapi.BuildRun {
	return &buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
	}
}
