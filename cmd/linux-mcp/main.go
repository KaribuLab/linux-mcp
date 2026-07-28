package main

import (
	"os"

	"github.com/KaribuLab/linux-mcp/internal/command"
)

func main() {
	os.Exit(command.Execute())
}
