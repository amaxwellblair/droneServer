package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	// Set flags
	ip := flag.String("ip", "localhost", "-ip=IP-ADDRESS")
	flag.Parse()

	// Continuously connect to the drone coordinator
	fmt.Println(*ip)
	for {
		if _, err := clientPostConnect(*ip); err != nil {
			fmt.Println(err)
			return
		}
		cmd := exec.Command("node", "./parrotAPI/runner.js")

		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		fmt.Println("Running Drone Runner")
		if err := cmd.Start(); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(cmd.Wait())
	}
}
