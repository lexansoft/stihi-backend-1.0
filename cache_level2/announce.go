package cache_level2

import (
	"github.com/pkg/errors"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
)

type AnnouncePage struct {
	Code 	string 		`json:"code"`
	Name 	string 		`json:"name"`
	Price 	int64 		`json:"price"`
	Unit    string		`json:"unit"`
}

func (dbConn *CacheLevel2) GetAnnouncesPages() (*[]*AnnouncePage, error) {
	rows, err := dbConn.Query(
		`
			SELECT code, name, price, unit
             FROM announces_pages
             ORDER BY id
		`,
		)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	list := make([]*AnnouncePage, 0)
	for rows.Next() {
		page := AnnouncePage{}
		rows.Scan(
			&page.Code,
			&page.Name,
			&page.Price,
			&page.Unit,
		)

		list = append(list, &page)
	}

	return &list, nil
}

func (dbConn *CacheLevel2) GetAnnouncePage(code string) (*AnnouncePage, error) {
	rows, err := dbConn.Query(
		`
			SELECT code, name, price, unit
             FROM announces_pages
             WHERE code = $1
		`,
		code,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		page := AnnouncePage{}
		rows.Scan(
			&page.Code,
			&page.Name,
			&page.Price,
			&page.Unit,
		)

		return &page, nil
	}

	return nil, errors.New("announce page with code "+code+" absent")
}

func (dbConn *CacheLevel2) CreateAnnounce(pageCode string, contentId int64, payer string, payData string) error {
	payerId, err := dbConn.GetUserId(payer)
	if err != nil {
		app.Error.Printf("Error get user ID for %s: %s", payer, err)
		payerId = -1
	}

	_, err = dbConn.Insert(
		`
			INSERT INTO announces
				(page_code, content_id, place_time, payer, payer_id, pay_data)
			VALUES
				($1, $2, $3, $4, $5, $6)
		`,
		pageCode, contentId, time.Now().UTC(), payer, payerId, payData,
	)
	if err != nil {
		app.EmailErrorf("Error add announce (%s, %d, %s): %s", pageCode, contentId, payer, err)
	}

	return err
}

func (dbConn *CacheLevel2) GetAnnouncesList(code string, count int, notMat bool, adminMode bool) (*[]*Article, error) {
	ignoreTagsJoin, ignoreTagsWhere := prepareIgnoreTagsSQL(notMat, false)

	sqlBan := " AND NOT a.ban AND NOT u.ban "
	if adminMode {
		sqlBan = " "
	}

	rows, err := dbConn.Query(`
			SELECT `+ArticlesListFieldSQL+` 
			FROM articles a `+
			ignoreTagsJoin+
			`, announces an, users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE an.page_code = $1 AND an.content_id = a.id AND a.author = u.name `+
			sqlBan+
			ignoreTagsWhere+
			`ORDER BY an.place_time DESC
			LIMIT $2
		`,
		code, count,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	list := make([]*Article, 0)
	for rows.Next() {
		content, _ := prepareArticle(rows, BodyContentPreview)

		list = append(list, content)
	}

	return &list, nil
}
