// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/emersion/go-message/textproto"

	"github.com/terinjokes/mailpail/pkgs/bitbucket"
	"github.com/terinjokes/mailpail/pkgs/maildir"
)

const millisInSecond = 1000
const nsInSecond = 1000000

// Converts Unix Epoch from milliseconds to time.Time
func FromUnixMilli(ms int64) time.Time {
	return time.Unix(ms/int64(millisInSecond), (ms%int64(millisInSecond))*int64(nsInSecond))
}

func headers(filename string) (textproto.Header, error) {
	f, err := os.Open(filename)
	if err != nil {
		return textproto.Header{}, nil
	}
	defer f.Close()

	return textproto.ReadHeader(bufio.NewReader(f))
}

func createMailForItem(item bitbucket.InboxPullRequest, diff []byte) ([]byte, error) {
	var message bytes.Buffer

	to := &mail.Address{
		Name:    item.Author.User.DisplayName,
		Address: item.Author.User.EmailAddress,
	}

	var h textproto.Header
	h.Set("From", to.String())
	h.Set("Subject", fmt.Sprintf("[%s/%s #%d] %s", item.To.Repository.Project.Key, item.To.Repository.Slug, item.Id, item.Title))
	h.Set("Date", FromUnixMilli(item.CreatedDate).Format(time.RFC1123Z))
	h.Set("Message-Id", fmt.Sprintf("<%d.%s.%s@bitbucket.cfdata.org>", item.Id, item.To.Repository.Project.Key, item.To.Repository.Slug))
	h.Set("Content-Location", item.Links["self"][0].Href)
	h.Set("X-Bitbucket-Version", strconv.FormatInt(int64(item.Version), 10))
	h.Set("Content-Type", "text/plain")

	if err := textproto.WriteHeader(&message, h); err != nil {
		return nil, err
	}

	message.Write([]byte(item.Description))
	message.Write([]byte("\n\n---\n\n"))
	message.Write(diff)
	message.Write([]byte("-- \n"))

	return message.Bytes(), nil
}

func createMailForComment(item bitbucket.InboxPullRequest, activity bitbucket.Activity) ([]byte, error) {
	var message bytes.Buffer

	to := &mail.Address{
		Name:    activity.Comment.Author.DisplayName,
		Address: activity.Comment.Author.EmailAddress,
	}

	var h textproto.Header
	h.Set("From", to.String())
	h.Set("Subject", fmt.Sprintf("Re: [%s/%s #%d] %s", item.To.Repository.Project.Key, item.To.Repository.Slug, item.Id, item.Title))
	h.Set("Date", FromUnixMilli(activity.CreatedDate).Format(time.RFC1123Z))
	h.Set("References", fmt.Sprintf("<%d.%s.%s@bitbucket.cfdata.org>", item.Id, item.To.Repository.Project.Key, item.To.Repository.Slug))
	h.Set("In-Reply-To", fmt.Sprintf("<%d.%s.%s@bitbucket.cfdata.org>", item.Id, item.To.Repository.Project.Key, item.To.Repository.Slug))
	h.Set("Message-Id", fmt.Sprintf("<%d.%d.%s.%s@bitbucket.cfdata.org>", activity.Id, item.Id, item.To.Repository.Project.Key, item.To.Repository.Slug))
	h.Set("X-Bitbucket-Version", strconv.FormatInt(int64(activity.Comment.Version), 10))
	h.Set("Content-Type", "text/plain")

	if err := textproto.WriteHeader(&message, h); err != nil {
		return nil, err
	}

	message.Write([]byte(activity.Comment.Text))

	return message.Bytes(), nil
}

func inboxItemKeyFunc(item bitbucket.InboxPullRequest) string {
	return fmt.Sprintf("%s.%s.pr.%d", item.To.Repository.Project.Key, item.To.Repository.Slug, item.Id)
}

func activityKeyFunc(item bitbucket.InboxPullRequest, activity bitbucket.Activity) string {
	return fmt.Sprintf("%s.%s.activity.%d.%d", item.To.Repository.Project.Key, item.To.Repository.Slug, item.Id, activity.Id)
}

func writeArticle(dir maildir.Maildir, key string, article []byte) error {
	art, err := dir.NewArticle(key)
	if err != nil {
		return err
	}

	n, err := art.Write(article)
	switch {
	case err != nil:
		return err
	case n != len(article):
		return fmt.Errorf("truncated write wrote=%d expected=%d", n, len(article))
	}

	if err := art.Close(); err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()

	client := http.DefaultClient
	endpoint := os.Getenv("BBAPI")
	token := os.Getenv("BBPAT")
	directory := "maildir/bitbucket"

	api := bitbucket.New(client, endpoint, token)

	dir := maildir.New(directory)
	os.MkdirAll(filepath.Join(directory, "tmp"), 0744)
	os.MkdirAll(filepath.Join(directory, "cur"), 0744)
	os.MkdirAll(filepath.Join(directory, "new"), 0744)

	inbox, err := api.Inbox(ctx)
	if err != nil {
		fmt.Printf("error fetching inbox: %s", err)
	}

	for _, item := range inbox {
		diff, err := api.Diff(ctx, item.To.Repository.Project.Key, item.To.Repository.Slug, item.Id)
		if err != nil {
			fmt.Printf("error fetching diff: %s\n", err)
			continue
		}

		article, err := createMailForItem(item, diff)
		if err != nil {
			fmt.Printf("error creating article: %s\n", err)
			continue

		}

		existing, err := dir.Filename(inboxItemKeyFunc(item))
		switch {
		case err != nil && err.(*maildir.KeyError).N == 0:
			if err := writeArticle(dir, inboxItemKeyFunc(item), article); err != nil {
				fmt.Printf("error writing article: %s\n", err)
				continue
			}
		case err != nil:
			fmt.Printf("error finding existing pr mails: %s\n", err)
			continue
		case existing != "":
			hdrs, err := headers(existing)
			if err != nil {
				fmt.Printf("error loading headers: %s\n", err)
			}

			version := hdrs.Get("X-Bitbucket-Version")
			if version != strconv.FormatInt(int64(item.Version), 10) {
				os.Remove(existing)

				if err := writeArticle(dir, inboxItemKeyFunc(item), article); err != nil {
					fmt.Printf("error writing article: %s\n", err)
					continue
				}
			}
		}

		activities, err := api.Activities(ctx, item.To.Repository.Project.Key, item.To.Repository.Slug, item.Id)
		if err != nil {
			fmt.Printf("error fetching activities: %s\n", err)
			continue
		}

		for _, activity := range activities {
			if activity.Action != "COMMENTED" {
				continue
			}

			article, err := createMailForComment(item, activity)
			if err != nil {
				fmt.Printf("error creating article: %s\n", err)
				continue
			}

			existing, err := dir.Filename(activityKeyFunc(item, activity))
			switch {
			case err != nil && err.(*maildir.KeyError).N == 0:
				if err := writeArticle(dir, activityKeyFunc(item, activity), article); err != nil {
					fmt.Printf("error writing article: %s\n", err)
					continue
				}
			case err != nil:
				fmt.Printf("error finding existing activity mails: %s\n", err)
				continue
			case existing != "":
				hdrs, err := headers(existing)
				if err != nil {
					fmt.Printf("error loading headers: %s\n", err)
				}

				version := hdrs.Get("X-Bitbucket-Version")
				if version != strconv.FormatInt(int64(activity.Comment.Version), 10) {
					os.Remove(existing)

					if err := writeArticle(dir, activityKeyFunc(item, activity), article); err != nil {
						fmt.Printf("error writing article: %s\n", err)
						continue
					}
				}
			}
		}
	}
}
