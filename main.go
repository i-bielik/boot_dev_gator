package main

import (
	"fmt"
	"log"
	"os"

	"github.com/i-bielik/boot-dev-gator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}
	fmt.Printf("Read config: %+v\n", cfg)

	state := &state{
		Config: &cfg,
	}

	cmds := &commands{}
	cmds.register("login", handlerLogin)

	// parse cli args
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("no command provided")
	}
	cmdName := args[1]
	cmdArgs := args[2:]
	cmd := command{
		Name: cmdName,
		Args: cmdArgs,
	}
	err = cmds.run(state, cmd)
	if err != nil {
		log.Fatalf("error running command: %v", err)
	}

}
