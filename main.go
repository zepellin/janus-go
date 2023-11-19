/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"janus/cmd"
	_ "janus/cmd/aws"
	_ "janus/cmd/gcp"
)

func main() {
	cmd.Execute()
}
