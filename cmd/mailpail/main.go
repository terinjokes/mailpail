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

func pullRequestCommentKeyFunc(pr bitbucket.PullRequest, comment bitbucket.PullRequestComment) string {
	return fmt.Sprintf("%s.%s.pr.%d.comment.%d",
		pr.ToRef.Repository.Project.Key,
		pr.ToRef.Repository.Slug,
		pr.ID,
		comment.ID,
	)
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

func articleForPullRequestComment(pr bitbucket.PullRequest, activity bitbucket.PullRequestActivity) ([]byte, error) {
	var message bytes.Buffer

	from := &mail.Address{
		Name:    activity.User.DisplayName,
		Address: activity.User.EmailAddress,
	}

	var h textproto.Header
	h.Set("From", from.String())
	h.Set("Subject", fmt.Sprintf("Re: [%s/%s #%d] %s", pr.ToRef.Repository.Project.Key, pr.ToRef.Repository.Slug, pr.ID, pr.Title))
	h.Set("Date", FromUnixMilli(pr.CreatedDate).Format(time.RFC1123Z))
	h.Set("Message-Id", fmt.Sprintf("<%s@bitbucket.cfdata.org>", pullRequestCommentKeyFunc(pr, activity.Comment)))
	h.Set("In-Reply-To", fmt.Sprintf("<%s@bitbucket.cfdata.org>", pullRequestItemKeyFunc(pr)))
	h.Set("References", fmt.Sprintf("<%s@bitbucket.cfdata.org>", pullRequestItemKeyFunc(pr)))
	h.Set("Content-Type", "text/plain")

	if err := textproto.WriteHeader(&message, h); err != nil {
		return nil, err
	}

	message.Write([]byte(activity.Comment.Text))

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

	deliveryDB, err := initDB(conf.Database)
	if err != nil {
		fmt.Printf("unable to create database: %s\n", err)
		os.Exit(-1)
	}

	c := &http.Client{
		Transport: &UATransport{rt: http.DefaultTransport},
	}
	api := bitbucket.New(c, conf.API.Endpoint, token)

	md := maildir.Maildir(conf.Maildir)
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

		exists, err := deliveryDB.HasPullRequest(ctx, proj, repo, prID)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}

		if !exists {
			diff, err := api.Diff(ctx, pullRequest.ToRef.Repository.Project.Key, pullRequest.ToRef.Repository.Slug, pullRequest.ID)
			if err != nil {
				fmt.Printf("err fetching diff: %s\n", err)
				os.Exit(-1)
			}

			article, _ := articleForPullRequest(pullRequest, diff)

			art, err := md.NewArticle()
			if err != nil {
				fmt.Printf("err: %s\n", err)
				os.Exit(-1)
			}
			defer art.Close()

			if _, err := art.Write(article); err != nil {
				fmt.Printf("err: %s\n", err)
				os.Exit(-1)
			}

			if err := deliveryDB.UpsertPullRequest(ctx, proj, repo, prID, 0); err != nil {
				fmt.Printf("err: %s\n", err)
				os.Exit(-1)
			}
		}

		lastActivity, err := deliveryDB.LastActivity(ctx, proj, repo, prID)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}

		fmt.Printf("lastActivity: %d\n", lastActivity)

		activities, err := api.PullRequestActivities(ctx, proj, repo, prID)
		if err != nil {
			fmt.Printf("err: %s\n", err)
			os.Exit(-1)
		}

		for _, activity := range activities {
			switch activity.Action {
			case "COMMENTED":
				if activity.ID > lastActivity {
					article, _ := articleForPullRequestComment(pullRequest, activity)
					art, err := md.NewArticle()
					if err != nil {
						fmt.Printf("err: %s\n", err)
						os.Exit(-1)
					}

					if _, err := art.Write(article); err != nil {
						fmt.Printf("err: %s\n", err)
						os.Exit(-1)
					}

					if err := deliveryDB.UpsertPullRequest(ctx, proj, repo, prID, activity.ID); err != nil {
						fmt.Printf("err: %s\n", err)
						os.Exit(-1)
					}

					art.Close()
				}

				// TODO: check recursively for new comments under activity.Comments.Comments
			default:
				fmt.Printf("skipping unknown action: %s\n", activity.Action)
			}
		}
	}
}

func initDB(file string) (*db.DB, error) {
	d, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", file))
	if err != nil {
		fmt.Printf("unable to open database: %s\n", err)
		os.Exit(-1)
	}

	d.Exec(`
CREATE TABLE IF NOT EXISTS pulls (
  key TEXT NOT NULL PRIMARY KEY,
  last_activity INTEGER
);
`)

	return db.New(d), nil
}
