package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type Director struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Surname string   `json:"surname"`
	Awards  []string `json:"awards,omitempty"`
}

type DirectorModel struct {
	DB *sql.DB
}

func (d DirectorModel) Insert(director *Director) error {
	query := `
		INSERT INTO directors(direc_name, direc_surname, awards)
		VALUES ($1, $2, $3)
		RETURNING id`

	return d.DB.QueryRow(query, &director.Name, &director.Surname, pq.Array(&director.Awards)).Scan(&director.ID)
}

func (d DirectorModel) Get(id int64) (*Director, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT *
		FROM directors
		WHERE id = $1`

	var director Director

	err := d.DB.QueryRow(query, id).Scan(
		&director.ID,
		&director.Name,
		&director.Surname,
		pq.Array(&director.Awards),
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &director, nil

}

// method for updating a specific record in the movies table.
func (d DirectorModel) Update(director *Director) error {
	query := `
		UPDATE directors
		SET direc_name = $1, direc_surname = $2, awards = $3
		WHERE id = $4`

	args := []interface{}{
		director.Name,
		director.Surname,
		pq.Array(director.Awards),
		director.ID,
	}

	return d.DB.QueryRow(query, args...).Scan()
}

// method for deleting a specific record from the movies table.
func (d DirectorModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	// Construct the SQL query to delete the record.
	query := `
		DELETE FROM directors
		WHERE id = $1`

	result, err := d.DB.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (d DirectorModel) GetAll(name string, surname string, awards []string, filters Filters) ([]*Director, error) {
	// query := `
	// SELECT *
	// FROM directors
	// ORDER BY id`

	// 	query := fmt.Sprintf(`
	// SELECT id, name, surname, awards
	// FROM directors
	// WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
	// AND (awards @> $2 OR $2 = '{}')
	// ORDER BY %s %s, id ASC
	// LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortByAwards())
	query := fmt.Sprintf(`
SELECT id, direc_name, direc_surname, awards
FROM directors
WHERE   (to_tsvector('simple', direc_name) @@ plainto_tsquery('simple', $1) OR $1 = '')
AND (awards @> $2 OR $2 = '{}')
ORDER BY %s %s, id ASC
LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortByAwards())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// rows, err := d.DB.QueryContext(ctx, query)
	rows, err := d.DB.QueryContext(ctx, query, name, surname, pq.Array(awards))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	directors := []*Director{}

	for rows.Next() {
		var director Director

		err := rows.Scan(
			&director.ID,
			&director.Name,
			&director.Surname,
			pq.Array(&director.Awards),
		)
		if err != nil {
			return nil, err
		}

		directors = append(directors, &director)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return directors, nil
}

func (d DirectorModel) GetOneByName(name string, filters Filters) ([]*Director, error) {
	// query := `
	// SELECT *
	// FROM directors
	// WHERE direc_name = '$1'
	// ORDER BY id`

	query := fmt.Sprintf(`
SELECT id, direc_name, direc_surname, awards
FROM directors
WHERE   (to_tsvector('simple', direc_name) @@ plainto_tsquery('simple', $1) OR $1 = '')
AND (awards @> $2 OR $2 = '{}')
ORDER BY %s %s, id ASC
LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortByAwards())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// rows, err := d.DB.Query("SELECT * FROM directors WHERE direc_name = ?", name)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	directors := []*Director{}

	for rows.Next() {
		var director Director

		// err := rows.Scan(
		// 	&director.ID,
		// 	&director.Name,
		// 	&director.Surname,
		// 	pq.Array(&director.Awards),
		// )
		// if err != nil {
		// 	return nil, err
		// }

		if err := rows.Scan(&director.ID, &director.Name, &director.Surname, pq.Array(&director.Awards)); err != nil {
			return directors, err
		}

		directors = append(directors, &director)
	}

	if err = rows.Err(); err != nil {
		return directors, err
	}

	return directors, nil
}
