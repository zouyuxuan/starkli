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
	"bytes"
	"fmt"
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/crush"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sbom"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"os"
	"path/filepath"
	"strings"
)

const TARGET_PATH = "/workplaces/target/dev"

type Starkli struct {
	Version          string
	LayerContributor libpak.DependencyLayerContributor
	Logger           bard.Logger
	Executor         effect.Executor
	keyStore         string
	account          string
	keystorePassword string
	declareHash      string
	rpcAddress       string
}

func NewStarkli(dependency libpak.BuildpackDependency, cache libpak.DependencyCache, args ...string) Starkli {
	contributor := libpak.NewDependencyLayerContributor(dependency, cache, libcnb.LayerTypes{
		Cache:  true,
		Launch: true,
		Build:  true,
	})
	return Starkli{
		account:          args[0],
		keyStore:         args[1],
		keystorePassword: args[2],
		rpcAddress:       args[3],
		Executor:         effect.NewExecutor(),
		Version:          dependency.Version,
		LayerContributor: contributor,
	}
}

func (s Starkli) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	return s.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		bin := filepath.Join(layer.Path, "bin")

		s.Logger.Bodyf("Expanding %s to %s", artifact.Name(), layer.Path)
		if err := crush.Extract(artifact, bin, 0); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to expand %s\n%w", artifact.Name(), err)
		}

		s.Logger.Bodyf("Setting %s as executable", bin)
		file := filepath.Join(bin, "starkli")
		if err := os.Chmod(file, 0755); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to chmod %s\n%w", file, err)
		}

		s.Logger.Bodyf("Setting %s in PATH", layer.Path)
		if err := os.Setenv("PATH", sherpa.AppendToEnvVar("PATH", ":", bin)); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to set $PATH\n%w", err)
		}

		buf := &bytes.Buffer{}
		if err := s.Executor.Execute(effect.Execution{
			Command: "starkli",
			Args:    []string{"-V"},
			Stdout:  buf,
			Stderr:  buf,
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error executing '%s -V':\n Combined Output: %s: \n%w", file, buf.String(), err)
		}
		ver := strings.Split(strings.TrimSpace(buf.String()), " ")
		s.Logger.Bodyf("Checking %s version: %s", file, ver[1])

		sbomPath := layer.SBOMPath(libcnb.SyftJSON)
		dep := sbom.NewSyftDependency(layer.Path, []sbom.SyftArtifact{
			{
				ID:      "starkli",
				Name:    "Starkli",
				Version: ver[1],
				Type:    "UnknownPackage",
				FoundBy: "amp-buildpacks/starkli",
				Locations: []sbom.SyftLocation{
					{Path: "amp-buildpacks/starkli/starkli/starkli.go"},
				},
				Licenses: []string{"MIT"},
				CPEs:     []string{fmt.Sprintf("cpe:2.3:a:starkli:starkli:%s:*:*:*:*:*:*:*", ver[1])},
				PURL:     fmt.Sprintf("pkg:generic/starkli@%s", ver[1]),
			},
		})
		s.Logger.Debugf("Writing Syft SBOM at %s: %+v", sbomPath, dep)
		if err := dep.WriteTo(sbomPath); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to write SBOM\n%w", err)
		}
		return layer, nil
	})
}

func (s Starkli) Name() string {
	return s.LayerContributor.LayerName()
}

func (s Starkli) StarknetContractBuild(build string) ([]libcnb.Process, error) {
	processes := []libcnb.Process{}
	if build == "true" {
		processes = append(processes, libcnb.Process{
			Type:             "web",
			Command:          "scarb build",
			Arguments:        []string{},
			Direct:           true,
			WorkingDirectory: "/workspace",
		})
	}
	return processes, nil
}
func (s Starkli) StarknetContractDeploy(deploy string) ([]libcnb.Process, error) {
	processes := []libcnb.Process{}
	//err := os.Setenv("STARKNET_KEYSTORE", s.keyStore)
	//err = os.Setenv("STARKNET_ACCOUNT ", s.account)
	//err = os.Setenv("STARKNET_RPC ", s.rpcAddress)
	//if err != nil {
	//	return []libcnb.Process{}, fmt.Errorf("os set env err = %s ", err)
	//}

	// todo constructor params
	if deploy == "true" {
		buf := &bytes.Buffer{}
		contractSierraPath := s.getTargetAbsolutePath()
		declareArgs := fmt.Sprintf("--keystore-password %s --keystore %s --account %s %s", s.keystorePassword, s.keyStore, s.account, contractSierraPath)
		err := s.Executor.Execute(effect.Execution{
			Command: "starkli declare",
			Args:    []string{declareArgs},
			Dir:     "/workspaces",
			Stdout:  buf,
			Stderr:  buf,
		})
		if err != nil {
			return []libcnb.Process{}, fmt.Errorf("error executing '%s declare':\n Combined Output: %s: \n%w", "starkli", buf.String(), err)
		}
		declareInfo := strings.Split(strings.TrimSpace(buf.String()), " ")
		for _, info := range declareInfo {
			if strings.HasPrefix(info, "0x") {
				s.declareHash = info
			}
		}

		deployArgs := fmt.Sprintf(" --keystore %s --account %s %s ", s.keyStore, s.account, s.declareHash)

		processes = append(processes, libcnb.Process{
			Type:             "web",
			Command:          "starkli deploy",
			Arguments:        []string{deployArgs},
			Direct:           true,
			WorkingDirectory: "/workspace",
		})
	}

	return processes, nil
}

func (s Starkli) getTargetAbsolutePath() string {
	files, err := filepath.Glob(TARGET_PATH)
	if err != nil {
		return ""
	}
	for _, file := range files {
		if strings.HasSuffix(file, ".sierra.json") {
			return filepath.Join(TARGET_PATH, file)
		}
	}
	return ""
}
