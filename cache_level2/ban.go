package cache_level2

import (
	"time"

	"gitlab.com/stihi/stihi-backend/app"
)

const (
	BanUserType = "user"
	BanContentType = "content"
)

func (dbConn *CacheLevel2) BanUser(userId int64, adminName string, description string) error {
	return dbConn.DoBanUnban("users", true, userId, adminName, description)
}

func (dbConn *CacheLevel2) UnbanUser(userId int64, adminName string, description string) error {
	return dbConn.DoBanUnban("users", false, userId, adminName, description)
}

func (dbConn *CacheLevel2) BanContent(contentId int64, adminName string, description string) error {
	return dbConn.DoBanUnban("content", true, contentId, adminName, description)
}

func (dbConn *CacheLevel2) UnbanContent(contentId int64, adminName string, description string) error {
	return dbConn.DoBanUnban("content", false, contentId, adminName, description)
}

func (dbConn *CacheLevel2) DoBanUnban(tableName string, ban bool, id int64, adminName string, description string) error {
	if id <= 0 || adminName == "" {
		return nil
	}

	_, err := dbConn.Do(`
		UPDATE `+tableName+` SET ban = $1 WHERE id = $2
		`,
		ban, id,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	_, err = dbConn.Do(`
		INSERT INTO ban_history
			(ban_object_type, ban_object_id, unban, admin_name, admin_description, time)
		VALUES
			($1, $2, $3, $4, $5, $6)
		`,
		BanContentType, id, !ban, adminName, description, time.Now().UTC(),
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}