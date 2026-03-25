package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	vcmd "zntr.io/vynilino/cmd/vynilino/internal/cmd"
)

func main() {
	root := &cobra.Command{
		Use:          "vynilino",
		Short:        "Vynilino vinyl collection manager",
		SilenceUsage: true,
	}
	root.AddCommand(vcmd.ServeCmd())
	root.AddCommand(vcmd.UserCmd())
	root.AddCommand(vcmd.BackupCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
