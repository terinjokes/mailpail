// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package maildir

import (
	"os"
	"path/filepath"
)

type Maildir string

func (d Maildir) NewArticle() (*Article, error) {
	fn, err := uniqueFilename()
	if err != nil {
		return nil, err
	}

	art := &Article{}
	file, err := os.Create(filepath.Join(string(d), "tmp", fn))
	if err != nil {
		return nil, err
	}

	art.file = file
	art.filename = fn
	art.d = d

	return art, nil
}
