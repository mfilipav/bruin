package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	jd "github.com/josephburnett/jd/lib"
)

func main() {
	current, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	integrationTestsFolder := filepath.Join(current, "integration-tests")

	entries, err := os.ReadDir(integrationTestsFolder)

	if err != nil {
		fmt.Printf("failed to read directory %s: %s\n", integrationTestsFolder, err.Error())
		os.Exit(2)
	}

	// Iterate over entries
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".git" || entry.Name() == "logs" {
			continue
		}
		// Check if the entry is a directory
		runTest(entry.Name(), integrationTestsFolder)
	}
}

func runTest(testName, integrationTestsFolder string) {
	integrationTestsFolder = integrationTestsFolder + string(os.PathSeparator)
	folder := filepath.Join(integrationTestsFolder, testName) + string(os.PathSeparator)
	assetFolder := filepath.Join(folder, "assets") + string(os.PathSeparator)
	fmt.Println("Running test for:", folder)
	cmd := exec.Command("go", "run", "main.go", "validate", folder)
	stdout, err := cmd.Output()
	fmt.Println(string(stdout))
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	//cmd = exec.Command("go", "run", "main.go", "run", "--use-uv", folder)
	//stdout, err = cmd.Output()
	//fmt.Println(string(stdout))
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(4)
	//}

	cmd = exec.Command("go", "run", "main.go", "internal", "parse-pipeline", folder)
	stdout, err = cmd.Output()
	if err != nil {
		fmt.Println("Error running parse-pipeline")
		fmt.Println(err)
		os.Exit(4)
	}
	expectation, err := jd.ReadJsonFile(filepath.Join(folder, "expectations", "pipeline.yml.json"))
	if err != nil {
		fmt.Println("Error running expectation for parse-pipeline")
		fmt.Println(err)
		os.Exit(5)
	}
	out := replacePaths(string(stdout), integrationTestsFolder, folder, assetFolder)

	parsed, err := jd.ReadJsonString(out)
	if err != nil {
		fmt.Println("Error parsing json output for pipeline " + folder)
		fmt.Println(err)
		os.Exit(5)
	}

	diff := expectation.Diff(parsed)
	if len(diff) != 0 {
		fmt.Println("Parsed pipeline not matching")
		fmt.Println(diff.Render())
		os.Exit(6)
	}

	assets, err := os.ReadDir(filepath.Join(folder, "assets"))
	if err != nil {
		fmt.Println("Error reading assets folder")
		fmt.Println(err)
		os.Exit(5)
	}
	for _, asset := range assets {
		if asset.IsDir() {
			continue
		}
		fmt.Println("Checking expectations for:" + asset.Name())
		cmd = exec.Command("go", "run", "main.go", "internal", "parse-asset", filepath.Join(folder, "assets", asset.Name())) //nolint:gosec
		stdout, err = cmd.Output()
		if err != nil {
			fmt.Println("Error running parse asset")
			fmt.Println(err)
			os.Exit(7)
		}

		expectation, err = jd.ReadJsonFile(filepath.Join(folder, "expectations", asset.Name()) + ".json")
		if err != nil {
			fmt.Println("Error reading expectation for parse asset")
			fmt.Println(err)
			os.Exit(8)
		}

		out = replacePaths(string(stdout), integrationTestsFolder, folder, assetFolder)
		replaced := out
		parsed, err = jd.ReadJsonString(replaced)
		if err != nil {
			fmt.Println("Error parsing json output for asset " + asset.Name())
			fmt.Println(err)
			os.Exit(8)
		}
		diff = expectation.Diff(parsed)
		if len(diff) != 0 {
			fmt.Printf("Asset %s not matching\n", asset.Name())
			fmt.Println(diff.Render())
			os.Exit(6)
		}
	}
}

func replacePaths(input, base, pipeline, asset string) string {
	if runtime.GOOS == "windows" {
		base = strings.ReplaceAll(base, "\\", "\\\\")
		pipeline = strings.ReplaceAll(pipeline, "\\", "\\\\")
		asset = strings.ReplaceAll(asset, "\\", "\\\\")
	}
	fmt.Printf("Replacing %s with %s\n", asset, "__ASSETSDIR__")
	fmt.Printf("Replacing %s with %s\n", pipeline, "__PIPELINEDIR__")
	fmt.Printf("Replacing %s with %s\n", base, "__BASEDIR__")
	input = strings.ReplaceAll(input, asset, "__ASSETSDIR__")
	input = strings.ReplaceAll(input, pipeline, "__PIPELINEDIR__")
	input = strings.ReplaceAll(input, filepath.Clean(base), "__BASEDIR__")
	return input
}
