package cache_level2

import (
	"database/sql"
	"gitlab.com/stihi/stihi-backend/app"
)

/*
func (dbConn *CacheLevel2) SaveFollowFromOperation(op *types.FollowOperation, ts time.Time) (int64, error) {
	// Все проверки на уровне cache_level1
	var id int64

	// Получаем id юзера
	userId, err := dbConn.GetUserId(op.Follower)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	if len(op.What) == 0 || op.What[0] == "" {
		// Отписка и удаление из игнора

		// Отписка
		err := dbConn.FollowRemove(userId, op.Following)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}

		// Удаление из игнора
		err = dbConn.IgnoreRemove(userId, op.Following)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}

		id = 0
	} else if op.What[0] == "blog" {
		// Подписка и удаление из игнора

		// Удаление из игнора
		err = dbConn.IgnoreRemove(userId, op.Following)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}

		// Подписка
		err = dbConn.FollowAdd(userId, op.Following)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}
	} else if op.What[0] == "ignore" {
		// Отписка и добавление в игнор

		// Отписка
		err := dbConn.FollowRemove(userId, op.Following)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}

		// Добавление в игнор
		err = dbConn.IgnoreAdd(userId, op.Following)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}
	}

	return id, nil
}
 */

func (dbConn *CacheLevel2) IgnoreRemove(userId int64, name string) error {
	_, err := dbConn.Do(
		`DELETE FROM blacklist WHERE user_id = $1 AND ignore_author = $2`,
		userId, name,
	)
	return err
}

func (dbConn *CacheLevel2) FollowRemove(userId int64, name string) error {
	_, err := dbConn.Do(
		`DELETE FROM follows WHERE user_id = $1 AND subscribed_for = $2`,
		userId, name,
	)
	return err
}

func (dbConn *CacheLevel2) IgnoreAdd(userId int64, name string) (error) {
	_, err := dbConn.Do(
		`INSERT INTO blacklist (user_id, ignore_author) 
		 VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		userId, name,
	)
	return err
}

func (dbConn *CacheLevel2) FollowAdd(userId int64, name string) (error) {
	_, err := dbConn.Do(
		`INSERT INTO follows (user_id, subscribed_for)
		 VALUES ($1, $2)
         ON CONFLICT DO NOTHING`,
		userId, name,
	)
	return err
}

func (dbConn *CacheLevel2) GetFollowsCount() (int64, error) {
	return dbConn.GetTableCount("follows")
}

// Список подписчиков
func (dbConn *CacheLevel2) GetUserFollowersList(userId int64) ([]*UserInfo, error) {
	userName, err := dbConn.GetUserNameById(userId)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	rows, err := dbConn.Query(`
		SELECT u.id, u.name,
			u.val_cyber_10x6, u.val_golos_10x6, u.val_power_10x6,
			ui.nickname, ui.birthdate, ui.biography,
			ui.sex, ui.place, ui.web_site,
			ui.avatar_image, ui.background_image, ui.pvt_posts_show_mode, ui.email
		FROM follows f, users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
		WHERE
			f.subscribed_for = $1 AND u.id = f.user_id AND u.stihi_user
		ORDER BY ui.nickname, u.name`,
	userName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*UserInfo, 0)
	for rows.Next() {
		userInfo := UserInfo{}
		var nickName sql.NullString
		var biography sql.NullString
		var birthdate NullTime
		var sex sql.NullString
		var place sql.NullString
		var webSite sql.NullString
		var avatarImage sql.NullString
		var backgroundImage sql.NullString
		var pvtPostsShowMode sql.NullString
		var email sql.NullString
		rows.Scan(
			&userInfo.Id,
			&userInfo.Name,
			&userInfo.ValCyber,
			&userInfo.ValGolos,
			&userInfo.ValPower,
			&nickName,
			&birthdate,
			&biography,
			&sex,
			&place,
			&webSite,
			&avatarImage,
			&backgroundImage,
			&pvtPostsShowMode,
			&email,
		)

		if nickName.Valid {
			userInfo.NickName = nickName.String
		}
		if biography.Valid {
			userInfo.Biography = biography.String
		}
		if birthdate.Valid {
			userInfo.BirthDate = birthdate.Format()
		}
		if sex.Valid {
			userInfo.Sex = sex.String
		}
		if place.Valid {
			userInfo.Place = place.String
		}
		if webSite.Valid {
			userInfo.WebSite = webSite.String
		}
		if avatarImage.Valid {
			userInfo.AvatarImage = avatarImage.String
		}
		if backgroundImage.Valid {
			userInfo.BackgroundImage = backgroundImage.String
		}
		if pvtPostsShowMode.Valid {
			userInfo.PvtPostsShowMode = pvtPostsShowMode.String
		}
		if email.Valid {
			userInfo.Email = email.String
		}

		list = append(list, &userInfo)
	}

	return list, nil
}

// Список тех, на кого подписан
func (dbConn *CacheLevel2) GetUserFollowsList(userId int64) ([]*UserInfo, error) {
	rows, err := dbConn.Query(`
		SELECT u.id, u.name,
			u.val_cyber_10x6, u.val_golos_10x6, u.val_power_10x6,
			ui.nickname, ui.birthdate, ui.biography,
			ui.sex, ui.place, ui.web_site,
			ui.avatar_image, ui.background_image, ui.pvt_posts_show_mode, ui.email
		FROM follows f, users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
		WHERE
			f.user_id = $1 AND u.name = f.subscribed_for AND u.stihi_user
		ORDER BY ui.nickname, u.name`,
		userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*UserInfo, 0)
	for rows.Next() {
		userInfo := UserInfo{}
		var nickName sql.NullString
		var biography sql.NullString
		var birthdate NullTime
		var sex sql.NullString
		var place sql.NullString
		var webSite sql.NullString
		var avatarImage sql.NullString
		var backgroundImage sql.NullString
		var pvtPostsShowMode sql.NullString
		var email sql.NullString
		rows.Scan(
			&userInfo.Id,
			&userInfo.Name,
			&userInfo.ValCyber,
			&userInfo.ValGolos,
			&userInfo.ValPower,
			&nickName,
			&birthdate,
			&biography,
			&sex,
			&place,
			&webSite,
			&avatarImage,
			&backgroundImage,
			&pvtPostsShowMode,
			&email,
		)

		if nickName.Valid {
			userInfo.NickName = nickName.String
		}
		if biography.Valid {
			userInfo.Biography = biography.String
		}
		if birthdate.Valid {
			userInfo.BirthDate = birthdate.Format()
		}
		if sex.Valid {
			userInfo.Sex = sex.String
		}
		if place.Valid {
			userInfo.Place = place.String
		}
		if webSite.Valid {
			userInfo.WebSite = webSite.String
		}
		if avatarImage.Valid {
			userInfo.AvatarImage = avatarImage.String
		}
		if backgroundImage.Valid {
			userInfo.BackgroundImage = backgroundImage.String
		}
		if pvtPostsShowMode.Valid {
			userInfo.PvtPostsShowMode = pvtPostsShowMode.String
		}
		if email.Valid {
			userInfo.Email = email.String
		}

		list = append(list, &userInfo)
	}

	return list, nil
}
