package skills

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type SkillFileWatcher struct {
	registry *SkillRegistry
	parser   *SkillParser
	watcher  *fsnotify.Watcher
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	debounce map[string]time.Time
}

type WatcherConfig struct {
	Directory  string
	AutoReload bool
	DebounceMs int
}

func NewSkillFileWatcher(registry *SkillRegistry, parser *SkillParser) (*SkillFileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SkillFileWatcher{
		registry: registry,
		parser:   parser,
		watcher:  watcher,
		ctx:      ctx,
		cancel:   cancel,
		debounce: make(map[string]time.Time),
	}, nil
}

func (w *SkillFileWatcher) Watch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	if err := w.watcher.Add(absPath); err != nil {
		return err
	}

	go w.processEvents()

	log.Printf("Skill file watcher started for: %s", path)
	return nil
}

func (w *SkillFileWatcher) WatchDirectory(dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	if err := w.watcher.Add(absDir); err != nil {
		return err
	}

	go w.processEvents()

	log.Printf("Skill file watcher started for directory: %s", dir)
	return nil
}

func (w *SkillFileWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cancel()

	if w.watcher != nil {
		w.watcher.Close()
	}

	log.Println("Skill file watcher stopped")
}

func (w *SkillFileWatcher) processEvents() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if w.shouldProcessEvent(event) {
				w.handleFileEvent(event)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (w *SkillFileWatcher) shouldProcessEvent(event fsnotify.Event) bool {
	if !strings.HasSuffix(strings.ToLower(event.Name), ".md") {
		return false
	}

	return event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Create == fsnotify.Create ||
		event.Op&fsnotify.Remove == fsnotify.Remove ||
		event.Op&fsnotify.Rename == fsnotify.Rename
}

func (w *SkillFileWatcher) handleFileEvent(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()

	if lastEvent, exists := w.debounce[event.Name]; exists {
		if now.Sub(lastEvent) < 500*time.Millisecond {
			return
		}
	}

	w.debounce[event.Name] = now

	go func() {
		time.Sleep(500 * time.Millisecond)
		w.processFileChange(event)
	}()
}

func (w *SkillFileWatcher) processFileChange(event fsnotify.Event) {
	if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
		w.handleFileRemoval(event.Name)
		return
	}

	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		w.handleFileUpdate(event.Name)
	}
}

func (w *SkillFileWatcher) handleFileUpdate(path string) {
	skill, err := w.parser.Parse(w.ctx, path)
	if err != nil {
		log.Printf("Failed to parse skill file %s: %v", path, err)
		return
	}

	if err := w.registry.Register(skill); err != nil {
		log.Printf("Failed to register skill %s from file %s: %v", skill.ID, path, err)
		return
	}

	log.Printf("Skill %s (%s) updated from file: %s", skill.ID, skill.Name, path)
}

func (w *SkillFileWatcher) handleFileRemoval(path string) {
	skills := w.registry.ListAll()

	filename := filepath.Base(path)
	filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	for _, skill := range skills {
		if strings.HasPrefix(skill.ID, filenameWithoutExt) {
			if err := w.registry.Unregister(skill.ID); err != nil {
				log.Printf("Failed to unregister skill %s: %v", skill.ID, err)
			} else {
				log.Printf("Skill %s (%s) removed due to file deletion: %s", skill.ID, skill.Name, path)
			}
			break
		}
	}
}

func (w *SkillFileWatcher) ReloadDirectory(ctx context.Context, dir string) error {
	w.registry.Clear()

	skills, err := w.parser.ParseDirectory(ctx, dir)
	if err != nil {
		return err
	}

	for _, skill := range skills {
		if err := w.registry.Register(skill); err != nil {
			log.Printf("Failed to register skill %s: %v", skill.ID, err)
		}
	}

	log.Printf("Reloaded %d skills from directory: %s", len(skills), dir)
	return nil
}

func (w *SkillFileWatcher) IsWatching(path string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	watchedDirs := w.watcher.WatchList()
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, dir := range watchedDirs {
		if strings.HasPrefix(absPath, dir) {
			return true
		}
	}

	return false
}

func (w *SkillFileWatcher) GetWatchedPaths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.watcher.WatchList()
}
