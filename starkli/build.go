/*
 * Copyright 2018-2020 the original author or authors.
 *
 * COPY FROM amp-buildpacks/scarb.git
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package starkli

import (
	"fmt"
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"log"
)

type Build struct {
	Logger bard.Logger
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()
	config, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	//buildStarkli, _ := config.Resolve("BP_ENABLE_STARKLI_PROCESS")
	dependency, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, err
	}
	libc, _ := config.Resolve("BP_STARKLI_LIBC")

	version, _ := config.Resolve("BP_STARKLI_VERSION")
	log.Printf("version = %s", version)
	buildDependency, _ := dependency.Resolve(fmt.Sprintf("starkli-%s", libc), version)
	log.Println("buildDependency = ", buildDependency)
	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	starkli := NewStarkli(buildDependency, dc)
	starkli.Logger = b.Logger
	result.Layers = append(result.Layers, starkli)
	return result, nil
}
