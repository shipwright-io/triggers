<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
Components
----------

This document describes the major components present in this application and organized in as packages in the [`pkg`](../pkg) directory.

<p align="center">
	<img alt="Shipwright Triggers components" src="./assets/components.drawio.png" />
</p>

# Build Inventory

The inventory is the central component of Shipwright Triggers, it stores all the Build instances organized in a way that allows searching for types of triggers, depending on the Inventory client.

For example, the WebHook Handler will always search for Builds based on the Git repository URL, the type of event (Push or PullRequest), and the branch names. In other hand, the other Controllers will query the inventory based on the `.objectRef` attribute instead.

As you can see on the diagram above, almost all components are interacting with the Inventory using the specialized query methods `SearchForGit` and `SearchForObjectRef`.

# WebHook Handler

The WebHook handler is a simple HTTP server implementation which receives requests from the outside, and after processing the event, searches over Builds that should be activated. The search on the inventory happens in the same fashion as the controllers, however uses `SearchForGit` method.

This type of `SearchForGit` is meant to match the repository URL, the type of event and the branches affected. For instance, the WebHook event can have different types, like Push or PullRequest and plus the branch affected.

# Kubernetes Controllers

## Shipwright Build Controller

The Builds are added or removed from the Inventory through the Build Controller, responsible to reflect all Shipwright Build resources into the Inventory. On adding new entries, the Build is prepared for the subsequent queries.

## Tekton Run Controller

Watches for Tekton Run instances referencing Shipwright Builds, when a new instance is created it creates a new BuildRun. The controller also watches over the BuildRun instance, in order to reflect the status back to the Tekton Run parent.

The Tekton Run instances are part of the Custom-Tasks workflow, everytime Tekton finds a TaskRef resource outside of Tekton's scope, it creates a Run instance with the coordinates. In other words, to extend Tekton's functionality third party applications must watch and interact with those objects.

## Tekton PipelineRun Controller

The controller for PipelineRun instances is meant to react when a Pipeline reaches the desired status, so upon changes on the resource the controller checks on the inventory if there are triggers configured for the specific resource in question, in the desired status.

Upon the creation of a BuildRun instance, the PipelineRun object is annotated to avoid reprocessing.
