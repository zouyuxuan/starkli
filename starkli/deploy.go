package starkli

import (
	"bytes"
	"fmt"
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/effect"
	"log"
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
	param, _ := cr.Resolve("BP_STARKNET_DEPLOY_PARAMS")

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
		if strings.Contains(declareInfo, "Class hash declared:") {
			log.Println("declare info ================= ", declareInfo)
			hash := strings.Split(strings.TrimSpace(declareInfo), ":")
			// contract declare hash
			classHash = hash[1]
		}

	}
	// deploy contract
	// starkli deploy --keystore ~/.starknet_accounts/key.json
	//--account ~/.starknet_accounts/starkli.json  0x05f9d67a7d0d935f1547bd93b6c58839c97951abf80eee4a7d4e509b29144a5e  0x4200f636d5efce3d72a027fde3d8bee2dad4234f0ef12b481d7f38a36b1c5d u256:10000 1  --keystore-password 12345

	process.Type = "web"
	process.Default = true
	process.Command = "starkli"
	process.Arguments = []string{
		"deploy ",
		"--account", account,
		"--keystore", keystore,
		"--rpc", rpc,
		classHash,
		param,
		"--keystore-password", keystorePassword,
	}
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
		log.Println("target file =============", file)
		if strings.Contains(file, "contract_class.json") {
			return file
		}
	}
	return ""
}
