// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// +build !linux

package maildir

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var (
	deliveries int64 = 10000
	hostname   string
	pid        int
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprint("unable to determine hostname"))
	}
	hostname = strings.ReplaceAll(hostname, "/", "\\057")
	hostname = strings.ReplaceAll(hostname, ":", "\\072")

	pid = os.Getpid()
}

// uniqueFilename implementation on non_linux systems returns filenames in the
// following format:
//
//  {timestamp}.P{pid}Q{delivery}R{random}M{millisecond}.{hostname}
//
// Where `timestamp` is the number of elasped seconds since 1 January 1970 UTC,
// `pid` is the process ID of this mailpail instance, `delivery` is a monotonically
// increasing number representing the number of messages delivered by this process,
// `random` is a 10 byte random number, `millisecond` is the millisecond component
// of the earlier timestamp, and `hostname` is the system hostname (with "/" and ":"
// characters escaped).
func uniqueFilename() (string, error) {
	now := time.Now()

	bs := make([]byte, 10)
	if _, err := io.ReadFull(rand.Reader, bs); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.P%dQ%dR%xM%d.%s",
		now.Unix(),
		pid,
		atomic.AddInt64(&deliveries, 1),
		bs,
		now.UnixNano()/1000%1000,
		hostname,
	), nil
}
