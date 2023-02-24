
# Copyright (2023 -- present) Shahruk Hossain <shahruk10@gmail.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#		 http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ==============================================================================

ARCH?=amd64

all: build

build: build-watcher

.PHONY: build-watcher
build-watcher:
		mkdir -p bin
		GOARCH=$(ARCH) GOOS=linux go build -o bin/watcher -ldflags "-s -w" ./cmd/watcher
		GOARCH=$(ARCH) GOOS=windows go build -o bin/watcher.exe -ldflags "-s -w" ./cmd/watcher

clean:
	rm -rf bin
