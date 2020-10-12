package repository

import (
	"log"
	"os"

	"github.com/alkmc/restClean/product/entity"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	// register sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

type sqliteRepo struct {
	db *sqlx.DB
}

//NewSQLite creates a new SQL lite repository
func NewSQLite() Repository {
	if err := os.Remove("./prods.db"); err != nil {
		log.Println(err)
	}

	sdb, err := sqlx.Open("sqlite3", "./prods.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sdb.Close()

	if _, err = sdb.Exec(sqlSchema); err != nil {
		log.Printf("%q: %s\n", err, sqlSchema)
	}
	return &sqliteRepo{db: sdb}
}

func (s *sqliteRepo) Save(p *entity.Product) (*entity.Product, error) {
	sdb, err := sqlx.Open("sqlite3", "./prods.db")
	if err != nil {
		return nil, err
	}

	tx, err := sdb.Begin()
	if err != nil {
		return nil, err
	}

	stmt, err := tx.Prepare(queryInsert)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()
	if _, err = stmt.Exec(p.ID, p.Name, p.Price); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *sqliteRepo) FindByID(id uuid.UUID) (*entity.Product, error) {
	sdb, err := sqlx.Open("sqlite3", "./prods.db")
	if err != nil {
		return nil, err
	}

	row := sdb.QueryRow(queryGetByID, id)

	var p entity.Product
	if err := row.Scan(&p.ID, &p.Name, &p.Price); err != nil {
		return nil, err
	}

	return &p, nil
}

func (s *sqliteRepo) FindAll() ([]entity.Product, error) {
	sdb, err := sqlx.Open("sqlite3", "./prods.db")
	if err != nil {
		return nil, err
	}
	rows, err := sdb.Query(queryGetAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []entity.Product
	for rows.Next() {
		var p entity.Product
		if err = rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *sqliteRepo) Update(p *entity.Product) error {
	sdb, err := sqlx.Open("sqlite3", "./prods.db")
	if err != nil {
		return err
	}

	tx, err := sdb.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(queryUpdate)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec(p.Name, p.Price, p.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *sqliteRepo) Delete(id uuid.UUID) error {
	sdb, err := sqlx.Open("sqlite3", "./prods.db")
	if err != nil {
		return err
	}
	tx, err := sdb.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(queryDelete)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err = stmt.Exec(id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}