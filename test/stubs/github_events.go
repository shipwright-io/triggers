// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package stubs

import (
	"time"

	"github.com/google/go-github/v53/github"
)

var RepoURL = "https://github.com/shipwright-io/sample-nodejs"

const (
	RepoFullName         = "shipwright-io/sample-nodejs"
	HeadCommitID         = "commit-id"
	HeadCommitMsg        = "commit message"
	HeadCommitAuthorName = "Author's Name"
	BeforeCommitID       = "before-commit-id"
	GitRef               = "refs/heads/main"
)

func GitHubPingEvent() github.PingEvent {
	return github.PingEvent{
		Zen:          github.String("zen"),
		HookID:       github.Int64(0),
		Installation: &github.Installation{},
	}
}

func GitHubPushEvent() github.PushEvent {
	return github.PushEvent{
		Repo: &github.PushEventRepository{
			HTMLURL:  github.String(RepoURL),
			FullName: github.String(RepoFullName),
		},
		HeadCommit: &github.HeadCommit{
			ID:      github.String(HeadCommitID),
			Message: github.String(HeadCommitMsg),
			Timestamp: &github.Timestamp{
				Time: time.Date(2022, time.March, 1, 0, 0, 0, 0, time.Local),
			},
			Author: &github.CommitAuthor{
				Name: github.String(HeadCommitAuthorName),
			},
		},
		Before: github.String(BeforeCommitID),
		Ref:    github.String(GitRef),
	}
}
