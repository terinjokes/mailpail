// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type API struct {
	client Doer
	token  string
	api    string
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

func New(client Doer, endpoint, token string) *API {
	return &API{
		client: client,
		token:  token,
		api:    endpoint,
	}
}

func (a *API) Inbox(ctx context.Context) ([]InboxPullRequest, error) {
	req, err := http.NewRequest("GET", a.api+"/inbox/pull-requests", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)

	req = req.WithContext(ctx)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var (
		bbresp       Response
		pullRequests []InboxPullRequest
	)

	if err := json.NewDecoder(resp.Body).Decode(&bbresp); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bbresp.Values, &pullRequests); err != nil {
		return nil, err
	}

	return pullRequests, nil
}

func (a *API) Diff(ctx context.Context, proj, slug string, id int) ([]byte, error) {
	req, err := http.NewRequest("GET", a.api+fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d/diff", proj, slug, id), nil)

	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Accept", "text/plain")

	req = req.WithContext(ctx)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (a *API) Activities(ctx context.Context, proj, slug string, id int) ([]Activity, error) {
	req, err := http.NewRequest("GET", a.api+fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d/activities", proj, slug, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)

	req = req.WithContext(ctx)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var (
		bbresp     Response
		activities []Activity
	)

	if err := json.NewDecoder(resp.Body).Decode(&bbresp); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bbresp.Values, &activities); err != nil {
		return nil, err
	}

	return activities, nil
}
