package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/i-bielik/boot-dev-gator/internal/config"
	"github.com/i-bielik/boot-dev-gator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}
	// fmt.Printf("Read config: %+v\n", cfg)

	// handle database connection
	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}
	defer db.Close()

	// initialize database queries
	queries := database.New(db)

	state := &state{
		db:     queries,
		Config: &cfg,
	}

	cmds := &commands{}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerListUsers)
	cmds.register("agg", handlerRssAggregate)
	cmds.register("addfeed", handlerAddFeed)
	cmds.register("feeds", handlerListFeeds)

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
