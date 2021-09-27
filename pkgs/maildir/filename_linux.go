// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package maildir

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var (
	deliveries int64 = 10000
	bootID     uuid.UUID
	machineID  uuid.UUID
	pid        int

	applicationID = [16]byte{
		0x06, 0x0e, 0x97, 0x74, 0x4c, 0x65, 0x49, 0x95,
		0xbf, 0xd6, 0x46, 0xcd, 0x3f, 0x16, 0x07, 0xc1,
	}
)

func init() {
	mid, err := os.ReadFile("/var/lib/dbus/machine-id")
	if err != nil {
		panic(fmt.Sprintf("unable to read machine-id: %s", err))
	}
	machineID = appID(uuid.Must(uuid.ParseBytes(bytes.TrimSpace(mid))), applicationID)

	bid, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		panic(fmt.Sprintf("unable to read boot-id: %s", err))
	}
	bootID = appID(uuid.Must(uuid.ParseBytes(bytes.TrimSpace(bid))), applicationID)

	pid = os.Getpid()
}

// appID takes a base ID (eg, machine-id and boot-id) and returns
// a cooresponding ID that is specific to the provided application
// ID. This allows these IDs to be used as stable identifiers, without
// accidentally exposing to untrused environments.
//
// This should compute the same values as `sd_id128_get_machine_app_specific`
// and `sd_id128_get_boot_app_specific`.
func appID(base [16]byte, app [16]byte) uuid.UUID {
	mac := hmac.New(sha256.New, base[:])
	mac.Write(app[:])
	id := mac.Sum(nil)[:16]

	// Set UUID version to 4 (random generation)
	id[6] = (id[6] & 0x0F) | 0x40
	// Set UUID variant to DCE
	id[8] = (id[8] & 0x3F) | 0x80

	return uuid.Must(uuid.FromBytes(id))
}

// uniqueFilename on Linux systems returns filenames in the following format:
//
//   {timestamp}.X{boot-id}P{pid}Q{delivery}R{random}M{millisecond}.D{machine-id}
//
// Where `timestamp` is the number of elapsed seconds since 1 January 1970 UTC,
// `boot-id` is a random number computed by Linux at every boot, `pid` is the
// process ID of this mailpail instance, `delivery` is an monotonically increasing
// number representing the number of messages delivered by this process, `random`
// is a 10 byte random number, `millisecond` is the millisecond component of the
// earlier timestamp, and `machine-id` is the unique machine ID set at installation.
//
// To deter drawing correlations between mailpail and other applications using the
// machine and boot IDs, the mailpail uses app specific forms of the identifiers:
// an HMAC hash of an appliction ID keyed by the corresponding system IDs.
func uniqueFilename() (string, error) {
	now := time.Now()

	bs := make([]byte, 10)
	if _, err := io.ReadFull(rand.Reader, bs); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.X%sP%dQ%dR%xM%d.D%s",
		now.Unix(),
		bootID,
		pid,
		atomic.AddInt64(&deliveries, 1),
		bs,
		now.UnixNano()/1000%1000,
		machineID,
	), nil
}
