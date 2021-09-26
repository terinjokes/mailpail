// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitbucket

import (
	"encoding/json"
)

type Response struct {
	Size       int             `json:"size"`
	Start      int             `json:"start"`
	Limit      int             `json:"limit"`
	IsLastPage bool            `json:"isLastPage"`
	Errors     []Error         `json:"errors"`
	Values     json.RawMessage `json:"values"`
}

type Error struct {
	Context       string `json:"context"`
	Message       string `json:"message"`
	ExceptionName string `json:"exceptionName"`
}

type PullRequest struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Open        bool   `json:"open"`
	Closed      bool   `json:"closed"`
	Locked      bool   `json:"locked"`
	CreatedDate int64  `json:"createdDate"`
	UpdatedDate int64  `json:"updatedDate"`
	ClosedDate  int64  `json:"closedDate"`

	FromRef      PullRequestReference     `json:"fromRef"`
	ToRef        PullRequestReference     `json:"toRef"`
	Author       PullRequestParticipant   `json:"author"`
	Reviewers    []PullRequestParticipant `json:"reviewers"`
	Participants []PullRequestParticipant `json:"participants"`
	Properties   map[string]interface{}   `json:"properties"`
	Links        RelatedLinks             `json:"links"`
}

type PullRequestParticipant struct {
	User               User   `json:"user"`
	LastReviewedCommit string `json:"lastReviewedCommit"`
	Role               string `json:"role"`
	Approved           bool   `json:"approved"`
	Status             string `json:"status"`
}

type PullRequestReference struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Type         string     `json:"type"`
	Repository   Repository `json:"repository"`
}

type Repository struct {
	Slug          string `json:"slug"`
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	HierarchyID   string `json:"hierarchyId"`
	ScmID         string `json:"scmId"`
	State         string `json:"state"`
	StatusMessage string `json:"statusMessage"`
	Forkable      bool   `json:"forkable"`
	// Origin        Repository `json:"origin"`
	Project Project      `json:"project"`
	Public  bool         `json:"public"`
	Links   RelatedLinks `json:"links"`
}

type Project struct {
	Namespace string       `json:"namespace"`
	Key       string       `json:"key"`
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Public    bool         `json:"public"`
	Type      string       `json:"type"`
	Links     RelatedLinks `json:"links"`
}

type User struct {
	Name         string       `json:"name"`
	EmailAddress string       `json:"emailAddress"`
	ID           int          `json:"id"`
	DisplayName  string       `json:"displayName"`
	Active       bool         `json:"active"`
	Slug         string       `json:"slug"`
	Type         string       `json:"type"`
	Links        RelatedLinks `json:"links"`
}

type RelatedLinks struct {
	Self        []Link `json:"self"`
	Clone       []Link `json:"clone"`
	MirrorClone []Link `json:"mirrorClone"`
}

type Link struct {
	Href string `json:"href"`
	Name string `json:"name"`
}
