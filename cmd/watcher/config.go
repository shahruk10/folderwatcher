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

import "github.com/shahruk10/watcher/internal/watcher"

type Config struct {
	Watcher  watcher.Config `yaml:"watcher"`
	Metadata Metadata       `yaml:"metadata"`
	Debug    bool           `yaml:"debug"`
}

type Metadata struct {
	FrameType2Name map[string]string `yaml:"frame_type_mapping"`
}

func (cfg *Config) Validate() error {
	return cfg.Watcher.Validate()
}
