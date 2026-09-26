package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchDebounce = 250 * time.Millisecond

func watchContent(contentDir, assetDir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := addContentDirs(watcher, contentDir); err != nil {
		return err
	}
	log.Printf("watching %s for Markdown changes", contentDir)

	// Keep watching even if the current content has a temporary Pikchr error;
	// the next edit should be able to recover without restarting the process.
	if err := generate(contentDir, assetDir); err != nil {
		log.Printf("Pikchr generation failed: %v", err)
	}

	var timer *time.Timer
	var timerC <-chan time.Time
	schedule := func() {
		if timer == nil {
			timer = time.NewTimer(watchDebounce)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(watchDebounce)
		}
		timerC = timer.C
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := addContentDirs(watcher, event.Name); err != nil {
						log.Printf("watching new directory %s: %v", event.Name, err)
					}
					schedule()
					continue
				}
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 ||
				(strings.EqualFold(filepath.Ext(event.Name), ".md") && event.Op&(fsnotify.Write|fsnotify.Create) != 0) {
				schedule()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("filesystem watcher error: %v", err)
			schedule()
		case <-timerC:
			timer = nil
			timerC = nil
			if err := generate(contentDir, assetDir); err != nil {
				log.Printf("Pikchr generation failed: %v", err)
			}
		}
	}
}

func addContentDirs(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if err := watcher.Add(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("add watch for %s: %w", path, err)
		}
		return nil
	})
}
