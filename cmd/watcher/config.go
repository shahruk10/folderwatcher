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
	"fmt"
	"regexp"
	"strings"

	"github.com/shahruk10/watcher/internal/watcher"
)

type Config struct {
	Watcher  watcher.Config `yaml:"watcher"`
	Metadata Metadata       `yaml:"metadata"`
	Debug    bool           `yaml:"debug"`
}

func (cfg *Config) Validate() error {
	if err := cfg.Metadata.Validate(); err != nil {
		return err
	}

	return cfg.Watcher.Validate()
}

type Metadata struct {
	FrameType2Name     map[string][]string `yaml:"frame_type_mapping"`
	FolderNamePatterns []string            `yaml:"folder_name_patterns"`
	FileNamePatterns   []string            `yaml:"file_name_patterns"`
}

func (cfg *Metadata) Validate() error {
	if len(cfg.FolderNamePatterns) == 0 {
		return fmt.Errorf("validate metadata: folder name pattern must be specified")
	}

	pattern := "(" + strings.Join(cfg.FolderNamePatterns, ")|(") + ")"
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("validate metadata: folder name pattern not valid regular expression, %w", err)
	}

	if len(cfg.FolderNamePatterns) == 0 {
		return fmt.Errorf("validate metadata: file name pattern must be specified")
	}

	pattern = "(" + strings.Join(cfg.FileNamePatterns, ")|(") + ")"
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("validate metadata: file name pattern not valid regular expression, %w", err)
	}

	if len(cfg.FrameType2Name) == 0 {
		return fmt.Errorf("validate metadata: frame_type_mapping must be specified")
	}

	return nil
}
