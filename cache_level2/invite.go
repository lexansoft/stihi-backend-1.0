package cache_level2

import (
	"time"

	"gitlab.com/stihi/stihi-backend/app"
)

type Invite struct {
	Id			int64		`json:"id"`
	AuthorName	string		`json:"author_name"`
	Author 		UserInfo 	`json:"author"`
}

func (dbConn *CacheLevel2) GetInvitesList( count int ) (*[]*Invite, error) {
	rows, err := dbConn.Query(`
			SELECT 
				id, author 
			FROM 
				invites 
			ORDER BY place_time DESC
			LIMIT $1
		`,
		count,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	list := make([]*Invite, 0)
	for rows.Next() {
		invite := Invite{}
		err = rows.Scan(
			&invite.Id,
			&invite.AuthorName,
		)

		list = append(list, &invite)
	}
	rows.Close()

	return &list, nil
}

func (dbConn *CacheLevel2) CreateInvite( login, payData string ) error {
	payerId, err := dbConn.GetUserId(login)
	if err != nil {
		app.Error.Printf("Error get user ID for %s: %s", login, err)
		payerId = -1
	}

	_, err = dbConn.Insert(
		`
			INSERT INTO invites
				(author, place_time, payer, payer_id, pay_data)
			VALUES
				($1, $2, $3, $4, $5)
		`,
		login, time.Now().UTC(), login, payerId, payData,
	)
	if err != nil {
		app.Error.Printf("Error add invite for %s: %s", login, err)
	}

	return err
}