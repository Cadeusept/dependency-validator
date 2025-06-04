package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Cadeusept/dependency-validator/internal"
	"github.com/Cadeusept/dependency-validator/internal/usecases/dependency_validator"
)

func main() {
	cfg, err := internal.LoadConfig(".dependency-validator-config.yaml")
	if err != nil {
		log.Fatalf("Failed to load .dependency-validator-config.yaml: %v", err)
	}

	dependencyValidator := dependency_validator.NewUsecase(cfg.Repos)

	depFile, err := dependencyValidator.DetectDependencyFile()
	if err != nil {
		log.Fatalf("Dependency file not found: %v", err)
	}
	fmt.Printf("Detected dependency file: %s\n", depFile)

	err = dependencyValidator.ParseDependencies(depFile)
	if err != nil {
		log.Fatalf("Failed to parse dependencies: %v", err)
	}

	_ = dependencyValidator.GetAssetVersions()

	outdated := dependencyValidator.CheckDependencies()

	if len(outdated) > 0 {
		fmt.Println("\nThe following dependencies are outdated:")
		for _, msg := range outdated {
			fmt.Println(" - " + msg)
		}
		os.Exit(1)
	}

	fmt.Println("\nAll dependencies are up to date.")
	os.Exit(0)
}
