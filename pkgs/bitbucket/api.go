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
	"net/url"
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

func (a *API) PullRequests(ctx context.Context, state string) ([]PullRequest, error) {
	q := url.Values{}
	q.Set("state", state)
	u, err := url.Parse(a.api + "/dashboard/pull-requests")
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
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
		pullRequests []PullRequest
	)

	if err := json.NewDecoder(resp.Body).Decode(&bbresp); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bbresp.Values, &pullRequests); err != nil {
		return nil, err
	}

	return pullRequests, nil
}

func (a *API) PullRequestActivities(ctx context.Context, proj, slug string, id int) ([]PullRequestActivity, error) {
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
		activities []PullRequestActivity
	)

	if err := json.NewDecoder(resp.Body).Decode(&bbresp); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bbresp.Values, &activities); err != nil {
		return nil, err
	}

	return activities, nil
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
