package cache_level2

import (
	"github.com/pkg/errors"

	"gitlab.com/stihi/stihi-backend/app"
)

type FixPage struct {
	Code 	string		`json:"code"`
	Title	string		`json:"title"`
	Html	string		`json:"html,omitempty"`
}

func (dbConn *CacheLevel2) UpdateFixPage(code string, html string, title string, adminName string) error {
	_, err := dbConn.Do(`
			INSERT INTO fix_pages
				(code, html, title)
			VALUES
				($1, $2, $3)
			ON CONFLICT
				(code)
			DO UPDATE SET
				html = EXCLUDED.html,
				title = EXCLUDED.title
		`,
		code, html,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) GetFixPage(code string) (*FixPage, error) {
	rows, err := dbConn.Query(`
			SELECT html, title
			FROM fix_pages
			WHERE code = $1
		`,
		code,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var page FixPage
		rows.Scan(
			&page.Html,
			&page.Title,
		)
		return &page, nil
	}

	return nil, errors.New("l10n:info.data_absent")
}

func (dbConn *CacheLevel2) GetFixPagesList() ([]*FixPage, error) {
	rows, err := dbConn.Query(`
			SELECT code, html, title
			FROM fix_pages
			ORDER BY title
		`,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	list := make([]*FixPage, 0)
	for rows.Next() {
		var page FixPage
		rows.Scan(
			&page.Code,
			&page.Html,
			&page.Title,
		)

		list = append(list, &page)
	}

	return list, nil
}
