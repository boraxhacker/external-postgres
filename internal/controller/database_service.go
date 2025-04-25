package controller

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

type pginstance struct {
	host     string
	port     int
	username string
	password string
}

type pgdatabase struct {
	name        string
	role        string
	password    string
	keepUpdated bool
}

func (pginst *pginstance) updateDatabase(ctx context.Context, pgdb *pgdatabase) error {

	connstr := fmt.Sprintf("host=%s port=%d user=%s password=%s",
		pginst.host, pginst.port, pginst.username, pginst.password)

	conn, err := pgx.Connect(ctx, connstr)
	if err != nil {
		return fmt.Errorf("connection host=%s:%d failed: %w", pginst.host, pginst.port, err)
	}
	defer conn.Close(ctx)

	err = pginst.doUpdateRole(ctx, conn, pgdb)
	if err != nil {
		return fmt.Errorf("create/update role=%s failed: %w", pgdb.role, err)
	}

	err = pginst.doUpdateDatabase(ctx, conn, pgdb)
	if err != nil {
		return fmt.Errorf("create/update database=%s failed: %w", pgdb.name, err)
	}

	return nil
}

func (pginst *pginstance) doUpdateRole(ctx context.Context, conn *pgx.Conn, pgdb *pgdatabase) error {

	exists, err := isRoleExists(ctx, conn, pgdb.role)
	if err == nil {

		pgconn := conn.PgConn()
		rolename, _ := pgconn.EscapeString(pgdb.role)
		password, _ := pgconn.EscapeString(pgdb.password)

		if !exists {

			adminuser, _ := pgconn.EscapeString(pginst.username)

			sql := fmt.Sprintf(`
			create role %s login password '%s'
			;
			grant %s to %s
			;
			`, rolename, password, rolename, adminuser)

			_, err = conn.Exec(ctx, sql)
			if err != nil {
				return fmt.Errorf("exec create role=%s sql failed: %w", pgdb.role, err)
			}

		} else if pgdb.keepUpdated {

			sql := fmt.Sprintf(`
			alter role %s with password '%s'
			;
			`, rolename, password)

			_, err = conn.Exec(ctx, sql)
		}
	}

	return err
}

func (pginst *pginstance) doUpdateDatabase(ctx context.Context, conn *pgx.Conn, pgdb *pgdatabase) error {

	exists, err := isDatabaseExists(ctx, conn, pgdb.name)

	if err == nil && !exists {

		pgconn := conn.PgConn()
		rolename, _ := pgconn.EscapeString(pgdb.role)
		dbname, _ := pgconn.EscapeString(pgdb.name)

		sql := fmt.Sprintf(`
		create database %s owner %s encoding utf8
		;
		`, dbname, rolename)

		_, err = conn.Exec(ctx, sql)
	}

	return err
}

func isRoleExists(ctx context.Context, conn *pgx.Conn, name string) (bool, error) {

	args := pgx.NamedArgs{
		"roleName": name,
	}

	rs, err := conn.Query(ctx, "SELECT 1 FROM pg_roles WHERE rolname = @roleName", args)
	defer rs.Close()

	if err != nil {
		return false, fmt.Errorf("checking role=%s exists failed: %w", name, err)
	}

	return rs.Next(), nil
}

func isDatabaseExists(ctx context.Context, conn *pgx.Conn, name string) (bool, error) {

	args := pgx.NamedArgs{
		"dbName": name,
	}

	rs, err := conn.Query(ctx, "SELECT 1 FROM pg_database WHERE datname = @dbName", args)
	defer rs.Close()

	if err != nil {
		return false, err
	}

	return rs.Next(), nil
}
