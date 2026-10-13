// Command admin provisions ImageFlow projects and API credentials.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	authpostgres "github.com/nickdavid1811/imageflow/internal/auth/adapters/postgres"
	authapplication "github.com/nickdavid1811/imageflow/internal/auth/application"
	"github.com/nickdavid1811/imageflow/internal/platform/config"
	"github.com/nickdavid1811/imageflow/internal/platform/database"
)

func main() {
	name := flag.String("name", "", "project name")
	environment := flag.String("environment", "test", "test or live")
	flag.Parse()
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	project, key, err := authapplication.NewIssuer(authpostgres.NewRepository(pool)).CreateProject(ctx, *name, *environment)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stdout, "project_id=%s\napi_key=%s\n", project.ID, key.Token)
}
