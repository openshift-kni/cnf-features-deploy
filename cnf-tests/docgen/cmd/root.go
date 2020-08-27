package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "docs",
	Short: "CNF tests doc generator",
	Long: `Docgen is a cli to generate the CNF tests detailed list of tests
including a description.

The description is provided via a json map between the test name and the
description itself.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func readCurrentDescriptions(fileName string) (map[string]string, error) {
	res := make(map[string]string)
	_, err := os.Stat(fileName)
	if !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("Failed reading file %s - %v", fileName, err)
		}

		err = json.Unmarshal(data, &res)
		if err != nil {
			log.Fatalf("Failed reading file %s - %v", fileName, err)
		}
	}
	return res, nil
}
