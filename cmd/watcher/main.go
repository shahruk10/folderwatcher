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
	"embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/gen2brain/beeep"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/shahruk10/watcher/internal/watcher"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

//go:embed assets
var embeddedData embed.FS

const (
	warnIconName = "warning.png"
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
		logger.Errorf("failed to parse command line arguments, %v", err)
		os.Exit(1)
	}

	waitCh := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := root.Run(ctx); err != nil && !errors.Is(err, flag.ErrHelp) && !errors.Is(err, ctx.Err()) {
			logger.Errorf("failed to execute, %v", err)
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

	w, err := watcher.New(logger, cfg.Watcher)
	if err != nil {
		return err
	}

	if err := w.AddFolders(cfg.Watcher.Folders...); err != nil {
		return fmt.Errorf("failed to add folders to watch list: %w", err)
	}

	callbacks := []watcher.Callback{
		CheckSizeAndFrame(cfg),
	}

	if err := w.AddCallbacks(callbacks...); err != nil {
		return fmt.Errorf("failed to add callbacks: %w", err)
	}

	warnIcon, err := embeddedData.ReadFile(path.Join("assets", warnIconName))
	if err != nil {
		return fmt.Errorf("loading assets: %w", err)
	}

	if err := os.WriteFile(path.Join(os.TempDir(), warnIconName), warnIcon, os.ModePerm); err != nil {
		return fmt.Errorf("loading assets: %w", err)
	}

	defer w.Close()

	logger.Infof("Press CTRL + C to close")
	logger.Infof("Monitoring %d folders ...", len(cfg.Watcher.Folders))
	logger.Debugf("Watched Folders: %v", cfg.Watcher.Folders)

	return w.Watch(ctx)
}

func CheckSizeAndFrame(cfg Config) watcher.Callback {
	return func(ctx context.Context, logger *logrus.Logger, e watcher.Event) error {
		const (
			dirNameDelimiter  = " "
			fileNameDelimiter = "_"
		)

		if !e.HasOp(watcher.CreateOp) && !e.HasOp(watcher.WriteOp) {
			return nil
		}

		fileName := strings.TrimSuffix(path.Base(e.Name), path.Ext(e.Name))
		parts := strings.Split(fileName, fileNameDelimiter)

		// Checking if file name as appropriate number of parts.
		if len(parts) != 3 {
			title := "<< INVALID NAME >>"
			msg := fmt.Sprintf("does not have 3 parts separated by %q: %q", fileNameDelimiter, e.Name)

			return showAlert(logger, title, msg)
		}

		frameType := strings.TrimSpace(parts[1])
		frameSize := strings.TrimSpace(parts[2])

		dirName := path.Base(path.Dir(e.Name))
		parts = strings.SplitN(dirName, dirNameDelimiter, 2)

		var dirFrameTypeName, dirFrameSize string

		if len(parts) == 1 {
			dirFrameSize = parts[0]
		} else if len(parts) == 2 {
			dirFrameSize, dirFrameTypeName = parts[0], parts[1]
		}

		dirFrameTypeName = strings.TrimSpace(dirFrameTypeName)
		dirFrameSize = strings.TrimSpace(dirFrameSize)

		frameTypeName, ok := cfg.Metadata.FrameType2Name[frameType]
		if !ok {
			title := "<< UNKNOWN FRAME TYPE >>"
			msg := fmt.Sprintf("unknown frame type abbreviation %q: %q", frameType, e.Name)

			return showAlert(logger, title, msg)
		}

		wrongFrameType := dirFrameTypeName != frameTypeName
		wrongFrameSize := dirFrameSize != frameSize

		if wrongFrameSize || wrongFrameType {
			correctDirName := strings.TrimSpace(fmt.Sprintf("%s %s", frameSize, frameTypeName))
			title := "<< WRONG FOLDER >>"
			msg := fmt.Sprintf("should be placed in %q instead of %q: %q", correctDirName, dirName, e.Name)

			return showAlert(logger, title, msg)
		}

		logger.Debugf("<< CORRECT FOLDER >> %q: %q", dirName, e.Name)

		return nil
	}
}

func showAlert(logger *logrus.Logger, title, msg string) error {
	logger.Infof("%s %s", title, msg)

	if err := beeep.Alert(title, msg, path.Join(os.TempDir(), warnIconName)); err != nil {
		return fmt.Errorf("failed to display %q alert: %v", title, err)
	}

	return nil
}
