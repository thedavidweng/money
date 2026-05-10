package main

import (
	"fmt"
	"os"

	"github.com/thedavidweng/money/internal/contracts"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		env := contracts.NewSuccess("version", map[string]string{"version": "0.0.0"})
		if err := contracts.WriteJSON(os.Stdout, env); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "money is bootstrapped. See docs/PRD.md for the initial product scope.")
	os.Exit(0)
}
