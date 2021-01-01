// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package maildir

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/luksen/maildir"
)

type Maildir struct {
	maildir.Dir
}

type KeyError = maildir.KeyError

func New(dir string) Maildir {
	return Maildir{Dir: maildir.Dir(dir)}
}

func (d Maildir) NewArticle(key string) (*Article, error) {
	ks, err := keySuffix()
	if err != nil {
		return nil, err
	}

	key = key + "." + ks

	art := &Article{}
	file, err := os.Create(filepath.Join(string(d.Dir), "tmp", key))
	if err != nil {
		return nil, err
	}

	art.file = file
	art.d = d
	art.key = key

	return art, nil
}

// Filename returns the path to the file corresponding to the key.
func (d Maildir) Filename(key string) (string, error) {
	n := 0
	var matchedFile string
	for _, dir := range []string{"cur", "new"} {
		dirPath := filepath.Join(string(d.Dir), dir)
		f, err := os.Open(dirPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		names, err := f.Readdirnames(-1)
		if err != nil {
			return "", err
		}
		for _, name := range names {
			if strings.HasPrefix(name, key) {
				if n == 0 {
					matchedFile = filepath.Join(dirPath, name)
				}
				n++
			}
		}
	}
	if n != 1 {
		return "", &maildir.KeyError{Key: key, N: n}
	}
	return matchedFile, nil
}
