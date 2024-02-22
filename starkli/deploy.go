package starkli

import (
	"bytes"
	"fmt"
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/effect"
	"os"
	"path/filepath"
	"strings"
)

func (s Starkli) DeployContract(cr libpak.ConfigurationResolver, app libcnb.Application) ([]libcnb.Process, error) {
	processes := []libcnb.Process{}

	enableDeploy := cr.ResolveBool("BP_STARKNET_DEPLOY")
	if enableDeploy {
		deployProcess, err := s.deploy(cr, app)
		if err != nil {
			return processes, err
		}
		processes = append(processes, deployProcess)
	}
	return processes, nil
}

func (s Starkli) deploy(cr libpak.ConfigurationResolver, app libcnb.Application) (libcnb.Process, error) {
	process := libcnb.Process{}
	// declare contract
	var classHash string

	contractName, _ := cr.Resolve("BP_STARKNET_CONTRACT_NAME")
	keystore, _ := cr.Resolve("BP_STARKNET_KEYSTORE_PATH")
	account, _ := cr.Resolve("BP_STARKNET_ACCOUNT_PATH")
	keystorePassword, _ := cr.Resolve("BP_STARKNET_KEYSTOREPASSWORD")
	rpc, _ := cr.Resolve("BP_STARKNET_NETWORK")
	contractPath := s.readContractTarget(app.Path, contractName)
	param, _ := cr.Resolve("BP_STARKNET_CONTRACT_NAME")

	buf := &bytes.Buffer{}
	if err := s.Executor.Execute(effect.Execution{
		Command: "starkli",
		Args: []string{
			"declare",
			"--account", account,
			"--keystore", keystore,
			"--keystore-password", keystorePassword,
			"--rpc", rpc,
			contractPath,
		},
		Stdout: buf,
		Stderr: buf,
	}); err != nil {
		return process, fmt.Errorf("error executing starkli declare:\n Combined Output: %s: \n%w", buf.String(), err)
	}
	declareInfos := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, declareInfo := range declareInfos {
		if strings.Contains(declareInfo, "class-hash") {
			hash := strings.Split(strings.TrimSpace(declareInfo), " ")
			// contract declare hash
			classHash = hash[1]
		}

	}
	// deploy contract
	process.Type = "web"
	process.Default = true
	process.Command = "starkli"
	process.Arguments = []string{
		"deploy ",
		classHash,
		contractPath,
	}
	process.Arguments = s.resolveParams(param, process.Arguments)
	return process, nil
}

func (s Starkli) readContractTarget(contractPath, contractName string) string {
	var files []string

	err := filepath.Walk(contractPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		return ""
	}
	for _, file := range files {
		if strings.Contains(file, contractName+".sierra.json") {
			return file
		}
	}
	return ""
}

func (s Starkli) resolveParams(param string, target []string) []string {
	list := strings.Split(param, ",")
	for _, l := range list {
		target = append(target, l)
	}
	return target
}
