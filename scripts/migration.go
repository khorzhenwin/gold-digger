package scripts

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"log"
)

func RunMigration(dsn string) error {
	log.Println("📦 Running migrations via golang-migrate...")

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to initialize migrate: %v", err)
		return err
	}

	err = m.Up()

	if err != nil && err.Error() != "no change" {
		log.Fatalf("❌ Migration failed: %v", err)
		return err
	}

	log.Println("✅  Migrations complete")
	return nil
}
