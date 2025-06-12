package main

import (
	"fmt"
	"github.com/Cadeusept/dependency-validator/internal"
	"github.com/Cadeusept/dependency-validator/internal/usecases/dependency_validator"
	"log"
	"os"
)

func main() {
	cfg, err := internal.LoadConfig(".dependency-validator-config.yaml")
	if err != nil {
		log.Fatalf("Failed to load .dependency-validator-config.yaml: %v", err)
	}

	dependencyValidator := dependency_validator.NewUsecase(cfg.Repos)

	sbomPath, err := dependencyValidator.DetectSBOM(".")
	if err != nil {
		log.Fatalf("Error detecting SBoM: %v\n", err)
	}

	fmt.Printf("Found SBoM file: %s\n", sbomPath)

	err = dependencyValidator.ParseSBOM(sbomPath)
	if err != nil {
		log.Fatalf("Failed to parse SBoM: %v", err)
	}

	_ = dependencyValidator.GetAssetVersions()

	outdated := dependencyValidator.CheckDependencies()

	if len(outdated) > 0 {
		fmt.Println("\n" + internal.TextColorRed + "The following dependencies are outdated:")
		for _, msg := range outdated {
			fmt.Println(" - " + msg)
		}
		fmt.Print(internal.TextColorReset)
		os.Exit(1)
	}

	fmt.Println("\n" + internal.TextColorGreen + "All dependencies are up to date." + internal.TextColorReset)
	os.Exit(0)
}
