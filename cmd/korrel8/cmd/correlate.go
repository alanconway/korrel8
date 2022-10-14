/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START GOAL FILE",
	Short: "Correlate from class START to class GOAL using start object in FILE. '-' means use stdin",
	Long: `
START  Name of start class.
GOAL   Name of goal class.
FILE   File containing instance of START class.
`,
	Args: cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		startClass, goalClass := must(e.ParseClass(args[0])), must(e.ParseClass(args[1]))
		f := open(args[2])
		defer f.Close()
		start := startClass.New()
		check(yaml.NewYAMLOrJSONDecoder(f, 1024).Decode(&start))
		paths := e.Rules.FindPaths(startClass, goalClass)
		var queries []korrel8.Query
		for _, p := range paths {
			queries = append(queries, must(e.Follow(context.Background(), start, nil, p))...)
		}
		fmt.Printf("\nresulting queries: %v\n\n", queries)
	},
}

func init() {
	rootCmd.AddCommand(correlateCmd)
}
