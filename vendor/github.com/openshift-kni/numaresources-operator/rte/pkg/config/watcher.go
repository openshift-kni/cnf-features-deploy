/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"github.com/fsnotify/fsnotify"

	"k8s.io/klog/v2"
)

type Watcher struct {
	watcher    *fsnotify.Watcher
	configPath string
	stopChan   chan struct{}
	callback   func() error
}

func NewWatcher(configPath string, callback func() error) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Warningf("config watch: failed to create the fsnotify watcher: %v", err)
		return nil, err
	}

	err = watcher.Add(configPath)
	if err != nil {
		klog.Warningf("config watch: failed to watch configuration file %q: %v", configPath, err)
		return nil, err
	}
	klog.Infof("config watch: added %q", configPath)

	return &Watcher{
		watcher:    watcher,
		configPath: configPath,
		callback:   callback,
		stopChan:   make(chan struct{}),
	}, nil
}

func (cw *Watcher) Close() {
	cw.watcher.Close()
}

func (cw *Watcher) Stop() {
	cw.stopChan <- struct{}{}
}

// WaitUntilChanges wait until it notices the first change of the config file.
// It nevers rearm itself, so it fires at most once.
// Make sure this run on a separate (not main) goroutine: see https://github.com/fsnotify/fsnotify#faq
func (cw *Watcher) WaitUntilChanges() {
	for {
		select {
		case <-cw.stopChan:
			return

		case event := <-cw.watcher.Events:
			klog.V(2).Infof("config watch: fsnotify event from %q: %v", event.Name, event.Op)
			if filterEvent(event) {
				err := cw.callback()
				if err != nil {
					klog.Warning("config watch: callback failed for %q: %v", cw.configPath, err)
				}
				// we would need to rearm the watch, which we don't yet. So exit.
				return
			}

		case err := <-cw.watcher.Errors:
			// and yes, keep going
			klog.Warningf("config watch: fsnotify error: %v", err)
		}
	}
}

func filterEvent(event fsnotify.Event) bool {
	if (event.Op & fsnotify.Write) == fsnotify.Write {
		return true
	}
	return (event.Op & fsnotify.Remove) == fsnotify.Remove
}
