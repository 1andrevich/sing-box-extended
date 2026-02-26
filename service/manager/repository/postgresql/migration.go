package postgresql

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/sagernet/sing-box/common/migrate/source"
)

var migrations = map[string]string{
	"1_initialize_schema.up.sql": `
        CREATE TABLE squads (
            id          SERIAL PRIMARY KEY,
            name        TEXT NOT NULL UNIQUE,
            created_at  TIMESTAMP NOT NULL,
            updated_at  TIMESTAMP NOT NULL
        );

        CREATE TABLE nodes (
            uuid        VARCHAR(36) PRIMARY KEY,
            name        TEXT NOT NULL UNIQUE,
            created_at  TIMESTAMP NOT NULL,
            updated_at  TIMESTAMP NOT NULL
        );

        CREATE TABLE node_to_squad (
            node_uuid VARCHAR(36) NOT NULL,
            squad_id INTEGER NOT NULL,
            PRIMARY KEY (node_uuid, squad_id),
            FOREIGN KEY (node_uuid) REFERENCES nodes(uuid) ON DELETE CASCADE,
            FOREIGN KEY (squad_id) REFERENCES squads(id) ON DELETE RESTRICT
        );

        CREATE TABLE users (
            id          SERIAL PRIMARY KEY,
            node_uuid   VARCHAR(36),
            username    TEXT NOT NULL,
            type        TEXT NOT NULL,
            inbound     TEXT NOT NULL,
            uuid        TEXT NOT NULL,
            password    TEXT NOT NULL,
            flow        TEXT NOT NULL,
            alter_id    INTEGER NOT NULL,
            created_at  TIMESTAMP NOT NULL,
            updated_at  TIMESTAMP NOT NULL,
            UNIQUE (username, inbound)
        );

        CREATE TABLE user_to_squad (
            user_id  INTEGER NOT NULL,
            squad_id INTEGER NOT NULL,
            PRIMARY KEY (user_id, squad_id),
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (squad_id) REFERENCES squads(id) ON DELETE RESTRICT
        );

        CREATE TABLE connection_limiters (
            id              SERIAL PRIMARY KEY,
            username        TEXT NOT NULL,
            outbound        TEXT NOT NULL,
            strategy        TEXT NOT NULL,
            connection_type TEXT NOT NULL,
            lock_type       TEXT NOT NULL,
            count           INTEGER NOT NULL,
            created_at      TIMESTAMP NOT NULL,
            updated_at      TIMESTAMP NOT NULL,
            UNIQUE (username, outbound)
        );

        CREATE TABLE connection_limiter_to_squad (
            connection_limiter_id  INTEGER NOT NULL,
            squad_id INTEGER NOT NULL,
            PRIMARY KEY (connection_limiter_id, squad_id),
            FOREIGN KEY (connection_limiter_id) REFERENCES connection_limiters(id) ON DELETE CASCADE,
            FOREIGN KEY (squad_id) REFERENCES squads(id) ON DELETE RESTRICT
        );

        CREATE TABLE bandwidth_limiters (
            id              SERIAL PRIMARY KEY,
            username        TEXT NOT NULL,
            outbound        TEXT NOT NULL,
            strategy        TEXT NOT NULL,
            mode            TEXT NOT NULL,
            connection_type TEXT NOT NULL,
            speed           TEXT NOT NULL,
            raw_speed       BIGINT NOT NULL DEFAULT 0,
            created_at      TIMESTAMP NOT NULL,
            updated_at      TIMESTAMP NOT NULL,
            UNIQUE (username, outbound)
        );

        CREATE TABLE bandwidth_limiter_to_squad (
            bandwidth_limiter_id  INTEGER NOT NULL,
            squad_id INTEGER NOT NULL,
            PRIMARY KEY (bandwidth_limiter_id, squad_id),
            FOREIGN KEY (bandwidth_limiter_id) REFERENCES bandwidth_limiters(id) ON DELETE CASCADE,
            FOREIGN KEY (squad_id) REFERENCES squads(id) ON DELETE RESTRICT
        );
    `,
	"1_initialize_schema.down.sql": `
	 	DROP TABLE IF EXISTS squas;
        DROP TABLE IF EXISTS nodes;
        DROP TABLE IF EXISTS users;
        DROP TABLE IF EXISTS bandwidth_limiters;
        DROP TABLE IF EXISTS connection_limiters;
    `,
}

func Migrate(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	sourceDriver := source.NewRawDriver(migrations)
	if err := sourceDriver.Init(); err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"raw",
		sourceDriver,
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	return m.Up()
}
