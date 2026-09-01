package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dcc-bigfred/rb/decoder"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// ClearSoundSlot removes all sound files from the given slot on the decoder.
func ClearSoundSlot(slot uint8, opts ...decoder.Option) error {
	rb := decoder.NewClient(opts...)
	return rb.ClearSoundSlot(slot)
}

// SyncSoundSlot synchronises a local directory with the given sound slot on the decoder:
//   - files present locally but missing on the decoder are uploaded
//   - files present on the decoder but missing locally are deleted from the decoder
//   - files present on both sides but differing in size (KB) are re-uploaded
//   - unless syncWithoutLast is true, the 5 most recently modified local files
//     (modified within the last 24 h) are always re-uploaded
//
// When dryRun is true, no changes are made – only a summary is printed.
func SyncSoundSlot(slot uint8, localDir string, dryRun bool, syncWithoutLast bool, opts ...decoder.Option) error {
	rb := decoder.NewClient(opts...)

	if dryRun {
		fmt.Printf("[dry-run] no changes will be made\n")
	}

	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("cannot read local directory %q: %w", localDir, err)
	}
	type localInfo struct {
		sizeBytes int64
		modTime   time.Time
	}
	localFiles := make(map[string]localInfo, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, statErr := e.Info()
		if statErr != nil {
			return fmt.Errorf("cannot stat %q: %w", e.Name(), statErr)
		}
		localFiles[e.Name()] = localInfo{sizeBytes: fi.Size(), modTime: fi.ModTime()}
	}

	recentlyModified := make(map[string]bool)
	if !syncWithoutLast {
		cutoff := time.Now().Add(-24 * time.Hour)

		type nameTime struct {
			name    string
			modTime time.Time
		}
		var candidates []nameTime
		for name, info := range localFiles {
			if info.modTime.After(cutoff) {
				candidates = append(candidates, nameTime{name, info.modTime})
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].modTime.After(candidates[j].modTime)
		})
		if len(candidates) > 5 {
			candidates = candidates[:5]
		}
		for _, c := range candidates {
			recentlyModified[c.name] = true
		}
		if len(recentlyModified) > 0 {
			logrus.Debugf("sync: %d recently modified file(s) will be force-uploaded (modified within last 24 h)", len(recentlyModified))
		}
	}

	remoteList, err := rb.ListSoundSlot(slot)
	if err != nil {
		return fmt.Errorf("cannot list slot %d on decoder: %w", slot, err)
	}
	remoteFiles := make(map[string]int64, len(remoteList))
	for _, info := range remoteList {
		remoteFiles[info.Name] = info.SizeKB
	}

	changes := 0
	for name, local := range localFiles {
		remoteSizeKB, existsRemotely := remoteFiles[name]
		if existsRemotely {
			localSizeKB := (local.sizeBytes + 1023) / 1024
			diff := localSizeKB - remoteSizeKB
			if diff < 0 {
				diff = -diff
			}
			if diff <= 1 {
				if recentlyModified[name] {
					fmt.Printf("recent:   %s (modified within last 24 h)\n", name)
					logrus.Infof("sync: force-uploading %q – modified within last 24 h", name)
				} else {
					logrus.Debugf("sync: skipping %q (size within tolerance: local %d KB, remote %d KB)", name, localSizeKB, remoteSizeKB)
					continue
				}
			} else {
				fmt.Printf("changed:  %s (local %d KB, remote %d KB)\n", name, localSizeKB, remoteSizeKB)
				logrus.Infof("sync: re-uploading %q (local %d KB, remote %d KB)", name, localSizeKB, remoteSizeKB)
			}
		} else {
			fmt.Printf("upload:   %s\n", name)
			logrus.Infof("sync: uploading new file %q to slot %d", name, slot)
		}

		changes++
		if dryRun {
			continue
		}

		f, openErr := os.Open(filepath.Join(localDir, name))
		if openErr != nil {
			return fmt.Errorf("cannot open %q: %w", name, openErr)
		}
		uploadErr := rb.UploadSoundFile(slot, name, f)
		_ = f.Close()
		if uploadErr != nil {
			return fmt.Errorf("upload %q failed: %w", name, uploadErr)
		}
	}

	for name := range remoteFiles {
		if _, exists := localFiles[name]; exists {
			continue
		}
		fmt.Printf("delete:   %s\n", name)
		logrus.Infof("sync: deleting %q from slot %d on decoder", name, slot)
		changes++
		if dryRun {
			continue
		}
		if delErr := rb.DeleteSoundFile(slot, name); delErr != nil {
			return fmt.Errorf("delete %q failed: %w", name, delErr)
		}
	}

	if changes == 0 {
		fmt.Printf("everything is up to date\n")
	}

	return nil
}

// WatchSoundSlot watches localDir for filesystem changes and triggers SyncSoundSlot
// each time a file is created, written or removed. A debounce of 500 ms is applied
// so that rapid bursts of events (e.g. an editor saving atomically) produce only
// one synchronisation run. The function blocks until the process is interrupted
// (i.e. the watcher channels are closed). Errors – including a failed initial sync
// or a failed triggered sync – are logged and printed, but never stop the watch loop.
func WatchSoundSlot(slot uint8, localDir string, dryRun bool, syncWithoutLast bool, opts ...decoder.Option) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("cannot create filesystem watcher: %w", err)
	}
	defer watcher.Close()

	if err = watcher.Add(localDir); err != nil {
		return fmt.Errorf("cannot watch directory %q: %w", localDir, err)
	}

	fmt.Printf("watch: watching %q for changes (Ctrl+C to stop)\n", localDir)
	logrus.Infof("watch: fsnotify watcher started on %q", localDir)

	runSync := func(reason string) {
		fmt.Printf("watch: %s, syncing…\n", reason)
		logrus.Infof("watch: %s, triggering sync of %q → slot %d", reason, localDir, slot)
		if syncErr := SyncSoundSlot(slot, localDir, dryRun, syncWithoutLast, opts...); syncErr != nil {
			fmt.Printf("watch: sync error: %v\n", syncErr)
			logrus.Errorf("watch: sync failed: %v", syncErr)
		}
	}

	runSync("starting initial sync")

	const debounce = 500 * time.Millisecond
	var timer *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				logrus.Debugf("watch: fsnotify event %s on %q", event.Op, event.Name)
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, func() { runSync("change detected") })
			}

		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("watch: watcher error: %v\n", watchErr)
			logrus.Errorf("watch: watcher error: %v", watchErr)
		}
	}
}
