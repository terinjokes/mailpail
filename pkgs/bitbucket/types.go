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
	Message       string `json:"context"`
	ExceptionName string `json:"exceptionName"`
}

type InboxPullRequest struct {
	Id          int    `json:"id"`
	Version     int    `json:"version"`
	CreatedDate int64  `json:"createdDate"`
	UpdatedDate int64  `json:"updatedDate"`
	Title       string `json:"title"`
	Description string `json:"description"`

	Author InboxAuthor       `json:"author"`
	To     InboxReference    `json:"toRef"`
	Links  map[string][]Link `json:"links"`
}

type InboxAuthor struct {
	User User `json:"user"`
}

type User struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

type InboxReference struct {
	Repository Repository `json:"repository"`
}

type Repository struct {
	Slug    string  `json:"slug"`
	Project Project `json:"project"`
}

type Project struct {
	Key string `json:"key"`
}

type Link struct {
	Href string `json:"href"`
}

type Activity struct {
	Id          int     `json:"id"`
	CreatedDate int64   `json:"createdDate"`
	Action      string  `json:"action"`
	Comment     Comment `json:"comment"`
}

type Comment struct {
	Version int    `json:"version"`
	Text    string `json:"text"`
	Author  User   `json:"author"`
}
