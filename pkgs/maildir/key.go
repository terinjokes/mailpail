// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package maildir

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"strconv"
	"sync/atomic"
)

var id int64 = 10000

func keySuffix() (string, error) {
	var key string

	bs := make([]byte, 10)
	if _, err := io.ReadFull(rand.Reader, bs); err != nil {
		return "", err
	}

	key += strconv.FormatInt(int64(os.Getpid()), 10)
	key += strconv.FormatInt(id, 10)
	atomic.AddInt64(&id, 1)
	key += hex.EncodeToString(bs)

	return key, nil
}
