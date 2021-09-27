// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package maildir

import (
	"os"
	"path/filepath"
)

type Article struct {
	file     *os.File
	filename string
	d        Maildir
}

func (a Article) Write(p []byte) (int, error) {
	return a.file.Write(p)
}

func (a Article) Close() error {
	err := a.file.Close()
	if err != nil {
		return err
	}

	var (
		t = filepath.Join(string(a.d), "tmp", a.filename)
		n = filepath.Join(string(a.d), "new", a.filename)
	)

	if err := os.Link(t, n); err != nil {
		return err
	}

	if err := os.Remove(t); err != nil {
		return err
	}

	return nil
}

func (a Article) Abort() error {
	if err := a.file.Close(); err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(string(a.d), "tmp", a.filename)); err != nil {
		return err
	}

	return nil
}
