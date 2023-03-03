// Copyright (2023 -- present) Shahruk Hossain <shahruk10@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//		 http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ==============================================================================

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/shahruk10/watcher/internal/watcher"
	"github.com/sirupsen/logrus"
	"github.com/sqweek/dialog"
	"gopkg.in/yaml.v3"
)

func main() {
	var (
		rootFlagSet = flag.NewFlagSet("watcher", flag.ExitOnError)
		cfgPath     = rootFlagSet.String("config", "watcher.yaml", "Path to watcher config file.")
		helpFlag    = rootFlagSet.Bool("help", false, "Display usage information.")
		verboseFlag = rootFlagSet.Bool("verbose", false, "Display debugging information.")
	)

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	root := &ffcli.Command{
		ShortUsage: "watcher [flags]",
		FlagSet:    rootFlagSet,
		Exec: func(ctx context.Context, args []string) error {
			if *helpFlag {
				rootFlagSet.Usage()
				return nil
			}

			if *verboseFlag {
				logger.SetLevel(logrus.DebugLevel)
			}

			if _, err := os.Stat(*cfgPath); os.IsNotExist(err) {
				return fmt.Errorf("failed to find watcher config file at %q", *cfgPath)
			}

			return watch(ctx, logger, *cfgPath)
		},
	}

	if err := root.Parse(os.Args[1:]); err != nil {
		title := "ERROR"
		msg := err.Error()
		showAlert(logger, title, msg)
		return
	}

	waitCh := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := root.Run(ctx); err != nil && !errors.Is(err, flag.ErrHelp) && !errors.Is(err, ctx.Err()) {
			title := "ERROR"
			msg := err.Error()
			showAlert(logger, title, msg)
		}

		cancel()
		close(waitCh)
	}()

	interrupt := make(chan os.Signal, 10)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-waitCh: // Terminated without user interrupt
		break
	case <-interrupt:
		cancel() // Stop the server
	}

	// Wait for the go routine above to return to gracefully stop.
	<-waitCh
}

func watch(ctx context.Context, logger *logrus.Logger, cfgPath string) error {
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		return fmt.Errorf("load config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}

	w, err := watcher.New(logger, cfg.Watcher)
	if err != nil {
		return err
	}

	watchList, err := getFoldersToWatch(cfg.Watcher)
	if err != nil {
		return err
	}

	if err := w.AddFolders(watchList...); err != nil {
		return fmt.Errorf("failed to add folders to watch list: %w", err)
	}

	callbacks := []watcher.Callback{
		CheckSizeAndFrame(cfg),
	}

	if err := w.AddCallbacks(callbacks...); err != nil {
		return fmt.Errorf("failed to add callbacks: %w", err)
	}

	defer w.Close()

	logger.Info("Monitoring following folders:")
	for i, folder := range watchList {
		logger.Printf("[%d] %s", i+1, folder)
	}

	logger.Info("Press CTRL + C to close")

	return w.Watch(ctx)
}

const (
	attrFrameSize = "frame_size"
	attrFrameType = "frame_type"
)

func CheckSizeAndFrame(cfg Config) watcher.Callback {
	return func(ctx context.Context, logger *logrus.Logger, e watcher.Event) error {
		if !e.HasOp(watcher.CreateOp) && !e.HasOp(watcher.WriteOp) {
			return nil
		}

		fileAttr, err := getFileAttributes(logger, e.Name, cfg.Metadata.FileNamePatterns)
		if err != nil {
			return err
		}

		dirAttr, err := getFolderAttributes(logger, filepath.Dir(e.Name), cfg.Metadata.FolderNamePatterns)
		if err != nil {
			return err
		}

		// File / folder attributes could not be parsed from file name;
		// getFileAttributes / getFolderAttributes will have shown an alert already,
		// so we just return here.
		if fileAttr == nil || dirAttr == nil {
			return nil
		}

		frameTypeNames, ok := cfg.Metadata.FrameType2Name[fileAttr[attrFrameType]]
		if !ok {
			title := "UNKNOWN FRAME TYPE"
			msg := fmt.Sprintf(
				"%s: %s\n%s: %s",
				"📁 file", e.Name, "❌ unknown frame type", fileAttr[attrFrameType],
			)

			return showAlert(logger, title, msg)
		}

		wrongFrameType := true
		for _, name := range frameTypeNames {
			wrongFrameType = wrongFrameType && dirAttr[attrFrameType] != name
		}

		wrongFrameSize := dirAttr[attrFrameSize] != fileAttr[attrFrameSize]
		currentDirName := filepath.Base(filepath.Dir(e.Name))

		if wrongFrameSize || wrongFrameType {
			var correctDirName string
			if !wrongFrameType {
				correctDirName = strings.TrimSpace(fmt.Sprintf("%s %s", fileAttr[attrFrameSize], dirAttr[attrFrameType]))
			} else {
				possibleNames := make([]string, 0, len(frameTypeNames))
				for _, name := range frameTypeNames {
					possibleNames = append(possibleNames, strings.TrimSpace(fmt.Sprintf("%s %s", fileAttr[attrFrameSize], name)))
				}

				correctDirName = strings.Join(possibleNames, " OR ")
			}

			title := "WRONG FOLDER"
			msg := fmt.Sprintf(
				"%s: %s\n%s: %s\n%s: %s",
				"📁 file", filepath.Base(e.Name), "❌ wrong", currentDirName, "✅ correct", correctDirName,
			)

			return showAlert(logger, title, msg)
		}

		logger.Debugf("CORRECT FOLDER %q: %q", currentDirName, e.Name)

		return nil
	}
}

func getFileAttributes(logger *logrus.Logger, filePath string, fileNamePatterns []string) (map[string]string, error) {
	pattern := "(" + strings.Join(fileNamePatterns, ")|(") + ")"
	fileNameRegex := regexp.MustCompile(pattern)
	attr := make(map[string]string)

	fileName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	matches := fileNameRegex.FindStringSubmatch(fileName)
	logger.Debugf("file attributes regex matches for %q: %s", fileName, matches)

	foundAttrFrameType := false
	foundAttrFrameSize := false

	if matches != nil {
		for i, attrName := range fileNameRegex.SubexpNames() {
			switch {
			case attrName == attrFrameType && matches[i] != "":
				attr[attrFrameType] = strings.ToLower(strings.TrimSpace(matches[i]))
				foundAttrFrameType = true
			case attrName == attrFrameSize && matches[i] != "":
				attr[attrFrameSize] = strings.ToLower(strings.TrimSpace(matches[i]))
				foundAttrFrameSize = true
			}
		}
	}

	if !foundAttrFrameType {
		title := "INVALID FILE NAME"
		msg := fmt.Sprintf(
			"%s: %s\n%s: %s",
			"📁 file", fileName, "❌ error", "does not specify frame type in the configured format",
		)

		return nil, showAlert(logger, title, msg)
	}

	if !foundAttrFrameSize {
		title := "INVALID FILE NAME"
		msg := fmt.Sprintf(
			"%s: %s\n%s: %s",
			"📁 file", fileName, "❌ error", "does not specify frame size in the configured format",
		)

		return nil, showAlert(logger, title, msg)
	}

	logger.Debugf("file attributes for %q: %s", fileName, attr)

	return attr, nil
}

func getFolderAttributes(logger *logrus.Logger, folderPath string, folderNamePatterns []string) (map[string]string, error) {
	patterns := "(" + strings.Join(folderNamePatterns, ")|(") + ")"
	dirNameRegex := regexp.MustCompile(patterns)
	attr := make(map[string]string)

	dirName := filepath.Base(folderPath)

	matches := dirNameRegex.FindStringSubmatch(dirName)
	logger.Debugf("folder attributes regex matches for %q: %s", dirName, matches)

	foundAttrFrameType := false
	foundAttrFrameSize := false

	if matches != nil {
		for i, attrName := range dirNameRegex.SubexpNames() {
			switch {
			case attrName == attrFrameType && matches[i] != "":
				attr[attrFrameType] = strings.ToLower(strings.TrimSpace(matches[i]))
				foundAttrFrameType = true
			case attrName == attrFrameSize && matches[i] != "":
				attr[attrFrameSize] = strings.ToLower(strings.TrimSpace(matches[i]))
				foundAttrFrameSize = true
			}
		}
	}

	// Frame type is optional in the folder name.
	if !foundAttrFrameType {
		attr[attrFrameType] = ""
	}

	if !foundAttrFrameSize {
		title := "INVALID FOLDER NAME"
		msg := fmt.Sprintf(
			"%s: %s\n%s: %s",
			"📁 folder", dirName, "❌ error", "does not specify frame size in the configured format",
		)

		return nil, showAlert(logger, title, msg)
	}

	logger.Debugf("folder attributes for %q: %s", dirName, attr)

	return attr, nil
}

func getFoldersToWatch(cfg watcher.Config) ([]string, error) {
	watchList := make([]string, 0)

	for _, topDir := range cfg.IncludeFolders {
		subDirs, err := filepath.Glob(topDir)
		if err != nil {
			return nil, fmt.Errorf("get sub directories in %q: %w", topDir, err)
		}

		shouldExclude := false

		for _, sd := range subDirs {
			info, err := os.Stat(sd)
			if err != nil || !info.IsDir() {
				continue
			}

			for _, toExclude := range cfg.ExcludeFolders {
				if toExclude == sd {
					shouldExclude = true
					break
				}
			}

			if shouldExclude {
				continue
			}

			watchList = append(watchList, sd)
		}
	}

	if len(watchList) == 0 {
		return nil, fmt.Errorf("no folders to watch under given config")
	}

	return watchList, nil
}

var windowMu sync.Mutex

var showAlert = func(logger *logrus.Logger, title, msg string) error {
	logger.Infof("<< %s >> %q", title, msg)

	windowMu.Lock()
	dialog.Message(msg).Title(title).Error()
	windowMu.Unlock()

	return nil
}
