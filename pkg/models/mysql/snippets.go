package mysql

import (
	"database/sql"
	"errors"

	"github.com/snippetbox/pkg/models"
)

type SnippetModel struct {
	DB *sql.DB
}

func (m *SnippetModel) Insert(title string, content string, expires string, userID int) (int, error) {
	stmt := `INSERT INTO snippets (title, content, user_id, created, expires)
VALUES(?, ?, ?,UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY))`

	result, err := m.DB.Exec(stmt, title, content, userID, expires)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil

}

func (m *SnippetModel) Get(id int) (*models.Snippet, error) {
	stmt := `SELECT id, title, content, created, expires FROM snippets
    WHERE expires > UTC_TIMESTAMP() AND id = ?`

	row := m.DB.QueryRow(stmt, id)

	s := &models.Snippet{}
	err := row.Scan(&s.ID, &s.Title, &s.Content, &s.Created, &s.Expires)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNoRecord
		} else {
			return nil, err
		}
	}

	return s, nil
}

func (m *SnippetModel) Latest() ([]*models.Snippet, error) {
	stmt := `SELECT id, title, content, user_id ,created, expires FROM snippets
WHERE expires > UTC_TIMESTAMP() ORDER BY created DESC LIMIT 10`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snippets := []*models.Snippet{}

	for rows.Next() {
		s := &models.Snippet{}

		err = rows.Scan(&s.ID, &s.Title, &s.Content, &s.UserID, &s.Created, &s.Expires)
		if err != nil {
			return nil, err
		}

		snippets = append(snippets, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return snippets, nil

}

func (m *SnippetModel) GetAuthor(userID int) (string, error) {
	stmt := `SELECT name FROM users WHERE id=?`
	row := m.DB.QueryRow(stmt, userID)

	var author string
	err := row.Scan(&author)
	if err != nil {
		return "", err
	}
	return author, nil
}

func (m *SnippetModel) Update(title string, content string, expires string, snippetID int) error {
	// 	stmt := `UPDATE snippets SET (title, content, expires) =
	// (?, ?, DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY)) WHERE id =?`
	stmt := `UPDATE snippets SET title = ?, content = ?, expires = DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY) WHERE id = ?`

	result, err := m.DB.Exec(stmt, title, content, expires, snippetID)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if n != 1 {
		return errors.New("error not equal to 1")
	}

	return nil
}

func (m *SnippetModel) Delete(id int) (int64, error) {

	// stmt := `DELETE FROM snippets WHERE id = ?`

	// result, err := m.DB.Exec(stmt)
	// if err != nil {
	// 	return err
	// }

	result, err := m.DB.Exec(`DELETE FROM snippets WHERE id = ?`, id)
	if err != nil {
		return 0, err
	} else {
		return result.RowsAffected()
	}

	//return nil
}
