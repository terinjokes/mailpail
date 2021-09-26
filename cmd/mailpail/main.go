// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"time"

	"github.com/emersion/go-message/textproto"
	_ "github.com/mattn/go-sqlite3"
	"github.com/terinjokes/mailpail/pkgs/bitbucket"
	"github.com/terinjokes/mailpail/pkgs/db"
	"github.com/terinjokes/mailpail/pkgs/maildir"
)

const millisInSecond = 1000
const nsInSecond = 1000000

// Converts Unix Epoch from milliseconds to time.Time
func FromUnixMilli(ms int64) time.Time {
	return time.Unix(ms/int64(millisInSecond), (ms%int64(millisInSecond))*int64(nsInSecond))
}

type UATransport struct {
	rt http.RoundTripper
}

func (u *UATransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "mailpail (github.com/terinjokes/mailpail)")
	return u.rt.RoundTrip(req)
}

func pullRequestItemKeyFunc(pr bitbucket.PullRequest) string {
	return fmt.Sprintf("%s.%s.pr.%d", pr.ToRef.Repository.Project.Key, pr.ToRef.Repository.Slug, pr.ID)
}

func articleForPullRequest(pr bitbucket.PullRequest, diff []byte) ([]byte, error) {
	var message bytes.Buffer

	to := &mail.Address{
		Name:    pr.Author.User.DisplayName,
		Address: pr.Author.User.EmailAddress,
	}

	var h textproto.Header
	h.Set("From", to.String())
	h.Set("Subject", fmt.Sprintf("[%s/%s #%d] %s", pr.ToRef.Repository.Project.Key, pr.ToRef.Repository.Slug, pr.ID, pr.Title))
	h.Set("Date", FromUnixMilli(pr.CreatedDate).Format(time.RFC1123Z))
	h.Set("Message-Id", fmt.Sprintf("<%s@bitbucket.cfdata.org>", pullRequestItemKeyFunc(pr)))
	h.Set("Content-Location", pr.Links.Self[0].Href)
	h.Set("Content-Type", "text/plain")

	if err := textproto.WriteHeader(&message, h); err != nil {
		return nil, err
	}

	// TODO: implement flow=reflow
	message.Write([]byte(pr.Description))
	message.Write([]byte("\n\n---\n\n"))
	message.Write(diff)
	message.Write([]byte("-- \n"))

	return message.Bytes(), nil
}

func main() {
	ctx := context.Background()

	// TODO: make the location of this config file a flag option.
	conf, err := LoadUserConfig()
	if err != nil {
		fmt.Printf("unable to load config file: %s\n", err)
		os.Exit(1)
	}

	token, err := conf.Token()
	if err != nil {
		fmt.Printf("unable to load token: %s\n", err)
		os.Exit(1)
	}

	sqldb, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", conf.Database))
	if err != nil {
		fmt.Printf("unable to open database: %s\n", err)
		os.Exit(-1)
	}

	// | id (PK) | project | repo | pr_id | last_activity |
	sqldb.Exec("create table pulls (id integer not null primary key, project text, repo text, pr_id integer, last_activity integer);")

	d := db.New(sqldb)

	c := &http.Client{
		Transport: &UATransport{rt: http.DefaultTransport},
	}
	api := bitbucket.New(c, conf.API.Endpoint, token)

	md := maildir.New(conf.Maildir)
	os.MkdirAll(filepath.Join(conf.Maildir, "tmp"), 0744)
	os.MkdirAll(filepath.Join(conf.Maildir, "cur"), 0744)
	os.MkdirAll(filepath.Join(conf.Maildir, "new"), 0744)

	pullRequests, err := api.PullRequests(ctx, "open")
	if err != nil {
		fmt.Printf("error fetch pull requests: %s\n", err)
	}

	for _, pullRequest := range pullRequests {
		var (
			proj = pullRequest.ToRef.Repository.Project.Key
			repo = pullRequest.ToRef.Repository.Slug
			prID = pullRequest.ID
		)

		exists, err := d.HasPullRequest(ctx, proj, repo, prID)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}

		if exists {
			fmt.Printf("skipping existing PR: %s/%s#%d\n", proj, repo, prID)
			continue
		}

		diff, err := api.Diff(ctx, pullRequest.ToRef.Repository.Project.Key, pullRequest.ToRef.Repository.Slug, pullRequest.ID)
		if err != nil {
			fmt.Printf("err fetching diff: %s\n", err)
			os.Exit(-1)
		}

		article, _ := articleForPullRequest(pullRequest, diff)

		art, err := md.NewArticle("t")
		if err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}
		defer art.Close()

		if _, err := art.Write(article); err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}

		if err := d.UpsertPullRequest(ctx, proj, repo, prID, 0); err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}
	}
}
