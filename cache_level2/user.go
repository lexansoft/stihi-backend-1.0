package cache_level2

import (
	"context"
	"database/sql"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"strconv"
	"sync"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cyber/cyberdb"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

var (
	StihiCreator    string
	UpdateUserMutex = &sync.Mutex{}
)

type User struct {
	operations.NewAccountOp
	Id            int64  `json:"id"`
	Ban           bool   `json:"ban"`
	StihiUserTime string `json:"stihi_user_time,omitempty"`

	Keys  map[string]string `json:"keys"`
	Names map[string]string `json:"names"`
}

type UserInfo struct {
	User
	Name               string  `json:"name"`
	ValGolos           float64 `json:"val_golos"`
	ValCyber           float64 `json:"val_cyber"`
	ValPower           float64 `json:"val_power"`
	ValPowerGOLOS      float64 `json:"val_power_golos"`
	ValDelegationGOLOS float64 `json:"val_delegation_golos"`
	ValReceivedGOLOS   float64 `json:"val_received_golos"`
	ValReputation      int64   `json:"val_reputation"`
	NickName           string  `json:"nickname"`
	BirthDate          string  `json:"birthdate"`
	Biography          string  `json:"biography"`

	Sex              string `json:"sex"`
	Place            string `json:"place"`
	WebSite          string `json:"web_site"`
	AvatarImage      string `json:"avatar"`
	BackgroundImage  string `json:"background_image"`
	PvtPostsShowMode string `json:"pvt_posts_show_mode"`
	Email            string `json:"email"`

	Battery1000  int    `json:"battery1000"`
	LastVoteTime string `json:"last_vote_time"`
}

type UserPublicKey struct {
	KeyType   string `json:"key_type"`
	KeyString string `json:"key_string"`
}

func (dbConn *CacheLevel2) CreateUser(cyberName, login, ownerPubKey, activePubKey, postingPubKey string, ts time.Time) error {
	userId, err := dbConn.Insert("INSERT INTO users"+
		"(name, stihi_user, time)"+
		"VALUES"+
		"($1, $2, NOW())",
		cyberName, true)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	for _, key := range [][]string{
		[]string{"owner", ownerPubKey},
		[]string{"active", activePubKey},
		[]string{"posting", postingPubKey},
	} {
		keyType := key[0]
		keyPub := key[1]
		_, err = dbConn.Do(`
				INSERT INTO users_keys
				(user_id, key_type, key)
				VALUES
				($1, $2, $3)
				`,
			userId, keyType, keyPub,
		)
	}

	// Добавляем имя
	_, err = dbConn.Do(`
				INSERT INTO users_names
				(user_id, creator, name)
				VALUES
				($1, $2, $3)
			`,
		userId, "gls", login,
	)
	if err != nil {
		return errors.Wrap(err, "Error when insert new user name")
	}

	return nil
}

func (dbConn *CacheLevel2) SaveUserFromOperation(op *operations.NewAccountOp, ts time.Time) error {
	user := User{}
	user.NewAccountOp = *op
	return dbConn.SaveUser(&user, ts)
}

func (dbConn *CacheLevel2) UpdateUserAuthFromOperation(op *operations.UpdateAuthOp, ts time.Time) error {
	// Добавляем новый ключ или если уже есть пара user_id/key_type - обновляем

	userId, err := dbConn.GetUserId(op.Data.Account)
	if err != nil {
		return errors.Wrap(err, "Error when find user id")
	}

	if userId < 0 {
		return nil
	}

	for _, key := range op.Data.Auth.Keys {
		affected, err := dbConn.Do(`
			UPDATE users_keys 
			SET key = $1 
			WHERE user_id = $2 AND key_type = $3`,
			key.Key, userId, op.Data.Permission,
		)
		if err != nil {
			return errors.Wrap(err, "Error when update user key")
		}

		if affected != 1 {
			// Если нет такого ключа - добавляем
			_, err = dbConn.Do(`
				INSERT INTO users_keys
				(user_id, key_type, key)
				VALUES
				($1, $2, $3)
				`,
				userId, op.Data.Permission, key.Key,
			)
		}
	}

	return nil
}

func (dbConn *CacheLevel2) SaveUser(user *User, ts time.Time) error {
	// Проверяем нет-ли уже такого юзера
	userId, err := dbConn.GetUserId(user.Data.Name)
	if err != nil && err.Error() != "l10n:info.data_absent" {
		app.Error.Print(err)
		return err
	}

	isStihiUser := user.Data.Creator == StihiCreator

	if userId < 0 {
		userId, err = dbConn.Insert("INSERT INTO users"+
			"(name, stihi_user, time)"+
			"VALUES"+
			"($1, $2, NOW())",
			user.Data.Name, isStihiUser)
		if err != nil {
			app.Error.Print(err)
			return err
		}
	}

	keysList, err := dbConn.GetUserKeysNodeos(user.Data.Name)
	for keyType, key := range keysList {
		affected, err := dbConn.Do(`
				UPDATE users_keys
				SET key = $1
				WHERE user_id = $2 AND key_type = $3
			`,
			key, userId, keyType,
		)

		if affected != 1 {
			_, err = dbConn.Do(`
				INSERT INTO users_keys
				(user_id, key_type, key)
				VALUES
				($1, $2, $3)
				`,
				userId, keyType, key,
			)
			if err != nil {
				app.Error.Printf("Insert user '%s' key `%s` error: %s", user.Data.Name, keyType, err)
				return errors.Wrap(err, "Insert user key error")
			}
		}
	}

	return nil
}

func (dbConn *CacheLevel2) GetUserByName(name string) (*User, error) {
	rows, err := dbConn.Query(
		`SELECT u.id, u.name, u.ban
			FROM users u
			LEFT JOIN users_names un ON un.user_id = u.id AND un.creator IN ('gls', 'stihi') 
			WHERE
				u.name = $1 OR un.name = $2
			LIMIT 1`,
		name, name,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	if !rows.Next() {
		rows.Close()
		return nil, nil
	}

	user := User{}
	err = rows.Scan(
		&user.Id,
		&user.Name,
		&user.Ban,
	)

	if err != nil {
		app.Error.Print(err)
		rows.Close()
		return nil, err
	}
	rows.Close()

	// Извлекаем ключи
	err = dbConn.GetUserKeys(&user)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	// Извлекаем имена ("gls" - имя в Голосе)
	err = dbConn.GetUserNames(&user)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	return &user, nil
}

func (dbConn *CacheLevel2) GetUserKeys(user *User) error {
	// Извлекаем ключи
	rows, err := dbConn.Query(`
		SELECT key_type, key
		FROM users_keys
		WHERE user_id = $1
		`,
		user.Id,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	if user.Keys == nil {
		user.Keys = make(map[string]string)
	}
	for rows.Next() {
		var keyType, key string
		err = rows.Scan(
			&keyType,
			&key)
		if err != nil {
			app.Error.Print(err)
			return err
		}

		user.Keys[keyType] = key
	}

	return nil
}

func (dbConn *CacheLevel2) GetUserKeysNodeos(userName string) (map[string]string, error) {
	// Извлекаем ключи
	keys := make(map[string]string)

	dbConn.mongo.Check()

	dbConn.mongo.SetDB(cyberdb.CyberDBName).SetCollection("permission")

	c := dbConn.mongo.Collection

	filter := bson.D{
		{"owner", userName},
	}

	cur, err := c.Find(context.TODO(), filter)
	if err != nil {
		app.Error.Printf("Error read permissions for user: %s : %s", userName, err)
		return nil, err
	}
	for cur.Next(context.TODO()) {
		perm := cyberdb.PermissionType{}
		err = cur.Decode(&perm)
		if err != nil {
			app.Error.Printf("Error decode permission for user: %s : %s", userName, err)
			return nil, err
		}

		if len(perm.Auth.Keys) > 0 {
			keys[perm.Name] = perm.Auth.Keys[0].Key
		}
	}
	_ = cur.Close(context.TODO())

	return keys, nil
}

func (dbConn *CacheLevel2) GetUserNames(user *User) error {
	// Извлекаем ключи
	rows, err := dbConn.Query(`
		SELECT creator, name
		FROM users_names
		WHERE user_id = $1
		`,
		user.Id,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	if user.Names == nil {
		user.Names = make(map[string]string)
	}
	for rows.Next() {
		var nameType, name string
		err = rows.Scan(
			&nameType,
			&name)
		if err != nil {
			app.Error.Print(err)
			return err
		}

		user.Names[nameType] = name
	}

	return nil
}

func (dbConn *CacheLevel2) GetUserId(name string) (int64, error) {
	if name == "" {
		return -1, errors.New("User name is empty")
	}

	rows, err := dbConn.Query(
		`SELECT u.id
			FROM users u
			LEFT JOIN users_names un ON un.user_id = u.id AND un.creator IN ('gls', 'stihi') 
			WHERE
				u.name = $1 OR un.name = $2
			LIMIT 1`,
		name, name,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	if !rows.Next() {
		return -1, nil
	}

	var userId int64
	err = rows.Scan(&userId)
	if err != nil {
		app.Error.Print(err)
		return userId, err
	}

	return userId, nil
}

func (dbConn *CacheLevel2) GetUserNameById(id int64, nameType ...string) (string, error) {
	var rows *sql.Rows
	var err error

	if len(nameType) <= 0 || nameType[0] == "" || nameType[0] == "cyber" {
		rows, err = dbConn.Query(
			`SELECT name FROM users WHERE id = $1`,
			id,
		)
		if err != nil {
			app.Error.Print(err)
			return "", err
		}
	} else {
		rows, err = dbConn.Query(
			`SELECT name FROM users_names WHERE user_id = $1 AND creator = $2`,
			id, nameType[0],
		)
		if err != nil {
			app.Error.Print(err)
			return "", err
		}
	}
	defer rows.Close()

	if !rows.Next() {
		return "", nil
	}

	var userName string
	err = rows.Scan(&userName)
	if err != nil {
		app.Error.Print(err)
		return "", err
	}

	return userName, nil
}

func (dbConn *CacheLevel2) GetUsersCount() (int64, error) {
	return dbConn.GetTableCount("users")
}

func (dbConn *CacheLevel2) IsKeysExists(key string) (bool, error) {
	rows, err := dbConn.Query(
		`SELECT user_id
			FROM users_keys 
			WHERE key = $1`,
		key,
	)
	if err != nil {
		app.Error.Print(err)
		return false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return false, nil
	}

	return true, nil
}

func (dbConn *CacheLevel2) GetUserInfo(id int64) (*UserInfo, error) {
	return dbConn.GetUserInfoByField("id", id)
}

func (dbConn *CacheLevel2) GetUserInfoByName(name string) (*UserInfo, error) {
	userId, err := dbConn.GetUserId(name)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	return dbConn.GetUserInfo(userId)
}

func (dbConn *CacheLevel2) GetUserInfoByField(fieldName string, val interface{}) (*UserInfo, error) {
	rows, err := dbConn.Query(
		`SELECT u.id, u.name,
				u.val_cyber_10x6, u.val_golos_10x6, u.val_power_10x6, u.val_delegated_10x6, u.val_received_10x6, u.val_reputation,			
				ui.nickname, ui.birthdate, ui.biography,
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, ui.background_image, ui.pvt_posts_show_mode, ui.email,
				u.ban, u.stihi_user_time,
				u.battery1000, u.last_vote_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id  
			WHERE
				u.`+fieldName+` = $1`,
		val,
	)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

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
	var stihiUserTime NullTime
	var lastVoteTime NullTime

	var valCyber int64
	var valGolos int64
	var valPower int64
	var valDelegated int64
	var valReceived int64

	err = rows.Scan(
		&userInfo.Id,
		&userInfo.Name,
		&valCyber,
		&valGolos,
		&valPower,
		&valDelegated,
		&valReceived,
		&userInfo.ValReputation,
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
		&userInfo.Ban,
		&stihiUserTime,
		&userInfo.Battery1000,
		&lastVoteTime,
	)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}

	userInfo.ValCyber = float64(valCyber) / FinanceSaveIndex
	userInfo.ValGolos = float64(valGolos) / FinanceSaveIndex
	userInfo.ValPower = float64(valPower) / FinanceSaveIndex

	userInfo.ValPowerGOLOS, userInfo.ValDelegationGOLOS, userInfo.ValReceivedGOLOS =
		dbConn.CalcPowerGOLOS(valPower, valDelegated, valReceived)

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

	if stihiUserTime.Valid {
		userInfo.StihiUserTime = stihiUserTime.Format()
	}

	if lastVoteTime.Valid {
		userInfo.LastVoteTime = lastVoteTime.Format()
	}

	// Ключи
	err = dbConn.GetUserKeys(&userInfo.User)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}

	// Извлекаем имена ("gls" - имя в Голосе)
	err = dbConn.GetUserNames(&userInfo.User)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}

	return &userInfo, nil
}

func (dbConn *CacheLevel2) UpdateUserInfo(userInfo *UserInfo) error {
	if userInfo == nil {
		app.Error.Printf("user info absent")
		return errors.New("user info absent")
	}
	if userInfo.Id <= 0 {
		app.Error.Printf("user ID absent")
		return errors.New("user ID absent")
	}

	_, err := dbConn.Do(
		`
			INSERT INTO users_info (user_id, nickname, birthdate, biography, sex, place, web_site,
    			avatar_image, background_image, pvt_posts_show_mode, email) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT
			( user_id )
			DO UPDATE SET
				nickname = EXCLUDED.nickname,
				birthdate = EXCLUDED.birthdate,
				biography = EXCLUDED.biography,
				sex = EXCLUDED.sex,
				place = EXCLUDED.place,
				web_site = EXCLUDED.web_site,
				avatar_image = EXCLUDED.avatar_image,
				background_image = EXCLUDED.background_image,
				pvt_posts_show_mode = EXCLUDED.pvt_posts_show_mode,
				email = EXCLUDED.email
		`,
		userInfo.Id, userInfo.NickName, NewNullString(userInfo.BirthDate), userInfo.Biography,
		userInfo.Sex, userInfo.Place, userInfo.WebSite, userInfo.AvatarImage, userInfo.BackgroundImage,
		userInfo.PvtPostsShowMode, userInfo.Email,
	)
	if err != nil {
		app.Error.Printf("Error when update user info: %s", err)
		return err
	}

	return nil
}

/*

Друзья, наш разработчик @kiathai сформировал предложение: https://golos.io/leaders/proposals/aqmi5nrretmn/pr3422155223.
На данный момент не разрешена проблема, связанная с невозможностью размещения постов пользователями, которые давно этого не делали (более 160 дней).
Она возникает из-за переполнения, создающегося при вычислении батарейки, ограничивающей награду за пост в условиях частого размещения постов. Для ликвидирования проблемы необходимо скорректировать настройки этой батарейки, иными словами - ограничить максимальные значения для входных параметров функции.
Предлашаем лидерам сообщества Голос утвердить данное предложение для исправления ошибки.

*/

// TODO: При вычислении батарейки ограничить время последнего действия максимальным значением в 160 дней
func (dbConn *CacheLevel2) UpdateUserBattery(userInfo *UserInfo) error {
	if userInfo == nil {
		app.Error.Printf("user info absent")
		return errors.New("user info absent")
	}
	if userInfo.Id <= 0 {
		app.Error.Printf("user ID absent")
		return errors.New("user ID absent")
	}

	lastVoteTime, _ := time.Parse(TimeJSONFormat, userInfo.LastVoteTime)

	_, err := dbConn.Do(
		`
			UPDATE 
				users
			SET
				battery1000 = $1,
				last_vote_time = $2
			WHERE
				id = $3
		`,
		userInfo.Battery1000, lastVoteTime, userInfo.Id,
	)
	if err != nil {
		app.Error.Printf("Error when update user batery info: %s", err)
		return err
	}

	return nil
}

/*
	LAST
*/

func (dbConn *CacheLevel2) GetNewUsersListLast(count int, filter string) (*[]*UserInfo, error) {
	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user `+
			filterWhere+`
			ORDER BY stihi_user_time DESC NULLS LAST
			LIMIT $1
		`,
		count,
	))
	if err != nil {
		app.Error.Printf("Error get users list LAST: %s", err)
		return nil, err
	}

	return rows, nil
}

func (dbConn *CacheLevel2) GetNameUsersListLast(count int, filter string) (*[]*UserInfo, error) {
	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user `+
			filterWhere+`
			ORDER BY u.name
			LIMIT $1
		`,
		count,
	))
	if err != nil {
		app.Error.Printf("Error get users names list LAST: %s", err)
		return nil, err
	}

	return rows, nil
}

func (dbConn *CacheLevel2) GetNicknameUsersListLast(count int, filter string) (*[]*UserInfo, error) {
	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND ui.nickname IS NOT NULL `+
			filterWhere+`
			ORDER BY ui.nickname
			LIMIT $1
		`,
		count,
	))
	if err != nil {
		app.Error.Printf("Error get users nicknames list LAST: %s", err)
		return nil, err
	}

	return rows, nil
}

/*
	AFTER - конец списка
*/

func (dbConn *CacheLevel2) GetNewUsersListAfter(lastUserId int64, count int, filter string) (*[]*UserInfo, error) {
	user, err := dbConn.GetUserInfo(lastUserId)
	if err != nil {
		app.Error.Printf("Error get users list AFTER: %s", err)
		return nil, err
	}

	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND stihi_user_time < $1 `+
			filterWhere+`
			ORDER BY stihi_user_time DESC NULLS LAST
			LIMIT $2
		`,
		user.StihiUserTime, count,
	))
	if err != nil {
		app.Error.Printf("Error get users list AFTER: %s", err)
		return nil, err
	}

	return rows, nil
}

func (dbConn *CacheLevel2) GetNameUsersListAfter(lastUserId int64, count int, filter string) (*[]*UserInfo, error) {
	filterWhere := prepareFilterSQL(filter)

	lastUserName, err := dbConn.GetUserNameById(lastUserId)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	if lastUserName == "" {
		return nil, errors.New("user in params not exists")
	}

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND u.name > $1 `+
			filterWhere+`
			ORDER BY u.name
			LIMIT $2
		`,
		lastUserName, count,
	))
	if err != nil {
		app.Error.Printf("Error get users names list AFTER: %s", err)
		return nil, err
	}

	return rows, nil
}

func (dbConn *CacheLevel2) GetNicknameUsersListAfter(lastUserId int64, count int, filter string) (*[]*UserInfo, error) {
	filterWhere := prepareFilterSQL(filter)

	userInfo, err := dbConn.GetUserInfo(lastUserId)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	if userInfo.NickName == "" {
		return nil, errors.New("user nickname not exists")
	}

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND u.nickname IS NOT NULL AND u.nickname > $1 `+
			filterWhere+`
			ORDER BY u.name
			LIMIT $2
		`,
		userInfo.NickName, count,
	))
	if err != nil {
		app.Error.Printf("Error get users nicknames list AFTER: %s", err)
		return nil, err
	}

	return rows, nil
}

/*
	BEFORE
*/

func (dbConn *CacheLevel2) GetNewUsersListBefore(lastUserId int64, filter string) (*[]*UserInfo, error) {
	user, err := dbConn.GetUserInfo(lastUserId)
	if err != nil {
		app.Error.Printf("Error get users list AFTER: %s", err)
		return nil, err
	}

	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND stihi_user_time > $1 `+
			filterWhere+`
			ORDER BY stihi_user_time DESC NULLS LAST
		`,
		user.StihiUserTime,
	))
	if err != nil {
		app.Error.Printf("Error get users list BEFORE: %s", err)
		return nil, err
	}

	return rows, nil
}

func (dbConn *CacheLevel2) GetNameUsersListBefore(lastUserId int64, filter string) (*[]*UserInfo, error) {
	lastUserName, err := dbConn.GetUserNameById(lastUserId)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	if lastUserName == "" {
		return nil, errors.New("user in params not exists")
	}

	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND u.name < $1 `+
			filterWhere+`
			ORDER BY u.name
		`,
		lastUserId,
	))
	if err != nil {
		app.Error.Printf("Error get users names list BEFORE: %s", err)
		return nil, err
	}

	return rows, nil
}

func (dbConn *CacheLevel2) GetNicknameUsersListBefore(lastUserId int64, filter string) (*[]*UserInfo, error) {
	userInfo, err := dbConn.GetUserInfo(lastUserId)
	if err != nil {
		app.Error.Print(err)
		return nil, err
	}
	if userInfo.NickName == "" {
		return nil, errors.New("user nickname not exists")
	}

	filterWhere := prepareFilterSQL(filter)

	rows, err := prepareUsersList(dbConn.Query(
		`SELECT u.id, u.name, ui.nickname, ui.birthdate, 
				ui.sex, ui.place, ui.web_site,
    			ui.avatar_image, u.val_power, u.val_reputation,
				u.ban, u.stihi_user_time
			FROM users u
			LEFT JOIN users_info ui ON u.id = ui.user_id
			WHERE u.stihi_user AND u.nickname IS NOT NULL AND u.nickname < $1 `+
			filterWhere+`
			ORDER BY u.nickname
		`,
		userInfo.NickName,
	))
	if err != nil {
		app.Error.Printf("Error get users nicknames list BEFORE: %s", err)
		return nil, err
	}

	return rows, nil
}

/*
---
*/

func prepareUsersList(rows *sql.Rows, err error) (*[]*UserInfo, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*UserInfo, 0)
	for rows.Next() {
		user := UserInfo{}
		var nickname sql.NullString
		var birthdate NullTime
		var sex sql.NullString
		var place sql.NullString
		var web_site sql.NullString
		var avatar_image sql.NullString
		var stihiUserTime NullTime
		err = rows.Scan(
			&user.Id,
			&user.Name,
			&nickname,
			&birthdate,
			&sex,
			&place,
			&web_site,
			&avatar_image,
			&user.ValPower,
			&user.ValReputation,
			&user.Ban,
			&stihiUserTime,
		)
		if err != nil {
			app.Error.Printf("Error when scan data row: %s", err)
		}

		if nickname.Valid {
			user.NickName = nickname.String
		}
		if birthdate.Valid {
			user.BirthDate = birthdate.Format()
		}
		if sex.Valid {
			user.Sex = sex.String
		}
		if place.Valid {
			user.Place = place.String
		}
		if web_site.Valid {
			user.WebSite = web_site.String
		}
		if avatar_image.Valid {
			user.AvatarImage = avatar_image.String
		}
		if stihiUserTime.Valid {
			user.StihiUserTime = stihiUserTime.Format()
		}

		list = append(list, &user)
	}

	return &list, nil
}

func (dbConn *CacheLevel2) UpdateUserBalances(name string, balance Balance, reputation int64, keys ...string) error {
	var err error
	var affected int64
	if len(keys) == 4 {
		affected, err = dbConn.Do(`
			UPDATE users 
				SET 
					val_cyber_10x6 = $1, val_golos_10x6 = $2, val_power_10x6 = $3,
					val_delegated_10x6 = $4, val_received_10x6 = $5,
					val_reputation = $6
				WHERE name = $11
			`,
			balance.Cyber, balance.Golos, balance.Power,
			balance.Delegated, balance.Received,
			reputation,
			name,
		)

		if affected < 1 {
			_, err = dbConn.Do(`
  				INSERT INTO users 
					(val_cyber_10x6, val_golos_10x6, val_power_10x6,
                     val_delegated_10x6, val_received_10x6,
                     val_reputation, name)
				VALUES
					($1, $2, $3, $4, $5, $6, $7)
				`,
				balance.Cyber, balance.Golos, balance.Power,
				balance.Delegated, balance.Received,
				reputation, name,
			)
		}
	} else {
		_, err = dbConn.Do(
			`
			UPDATE users 
				SET 
					val_cyber_10x6 = $1, val_golos_10x6 = $2, val_power_10x6 = $3,
					val_delegated_10x6 = $4, val_received_10x6 = $5,
					val_reputation = $6
				WHERE name = $7
		`,
			balance.Cyber, balance.Golos, balance.Power,
			balance.Delegated, balance.Received,
			reputation, name,
		)
	}
	if err != nil {
		app.Error.Printf("Error update user balance: %s", err)
	}
	return err
}

func (dbConn *CacheLevel2) ChangeUserBalances(name string, balance Balance) error {
	_, err := dbConn.Do(
		`
			UPDATE users 
				SET 
					val_cyber_10x6 = val_cyber_10x6 + $1, val_golos_10x6 = val_golos_10x6 + $2, val_power_10x6 = val_power_10x6 + $3,
					val_delegated_10x6 = val_delegated_10x6 + $4, val_received_10x6 = val_received_10x6 + $5  
				WHERE name = $6
		`,
		balance.Cyber, balance.Golos, balance.Power, balance.Delegated, balance.Received, name,
	)
	if err != nil {
		app.Error.Printf("Error change user balance: %s", err)
	}
	return err
}

func (dbConn *CacheLevel2) GetUserPeriodLeader(days int) (int64, error) {
	rows, err := dbConn.Query(
		`
			SELECT * 
				FROM 
					(
						SELECT user_id, sum(val_cyber_change) as sum 
						FROM users_history 
						WHERE operation_type = 'author_reward' AND time > NOW() - ($1 || ' days')::interval 
						GROUP BY user_id
					) sq 
				ORDER BY sum DESC 
				LIMIT 1
		`,
		strconv.FormatInt(int64(days), 10),
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}
	defer rows.Close()

	userId := int64(-1)
	if rows.Next() {
		var sum int64
		err = rows.Scan(
			&userId,
			&sum,
		)
		if err != nil {
			app.Error.Print(err)
			return -1, err
		}
	}

	return userId, nil
}

func (dbConn *CacheLevel2) SetStihiUser(userId int64) error {
	_, err := dbConn.Do(`
			UPDATE users SET stihi_user = 't', stihi_user_time = NOW() WHERE id = $1 AND NOT stihi_user 
		`,
		userId,
	)
	return err
}

func (dbConn *CacheLevel2) SetStihiUserByLogin(login string) error {
	_, err := dbConn.Do(`
			UPDATE users SET stihi_user = 't', stihi_user_time = NOW() WHERE name = $1 AND NOT stihi_user
		`,
		login,
	)
	return err
}

func prepareFilterSQL(filter string) string {
	where := ""
	switch filter {
	case "banned":
		where = " AND u.ban "
	case "notbanned":
		where = " AND NOT u.ban "
	}

	return where
}

func (dbConn *CacheLevel2) StihiUserListFilter(names []string) []string {
	list := make([]string, 0)
	rows, err := dbConn.Query(
		`
			SELECT name 
				FROM 
					users 
				WHERE
					name = ANY($1) AND
					stihi_user
		`,
		pq.Array(names),
	)
	if err != nil {
		app.Error.Print(err)
		return []string{}
	}
	defer rows.Close()

	if rows.Next() {
		var name string
		err = rows.Scan(
			&name,
		)
		if err != nil {
			app.Error.Print(err)
		} else {
			list = append(list, name)
		}
	}

	return list
}

/*
func (dbConn *CacheLevel2) UpdateUserAccountInfo(account *database.Account) error {
	if account == nil {
		return errors.New("account data is null")
	}

	golos := int64(account.Balance.Amount * FinanceSaveIndex)
	gbg := int64(account.SbdBalance.Amount * FinanceSaveIndex)
	power := int64(account.VestingShares.Amount * FinanceSaveIndex)
	reputation := int64(*account.Reputation)

	accountJson, err := json.Marshal(account)
	if err != nil {
		app.Error.Println(err)
		return err
	}

	_, err = dbConn.Do(`
		UPDATE users
			SET
				account_info = $1,
				val_golos_10x6 = $2,
				val_power_10x6 = $3,
				val_cyber_10x6 = $4,
				val_reputation = $5
			WHERE name = $6
	`,
		accountJson,
		golos,
		power,
		gbg,
		reputation,
		account.Name,
	)
	if err != nil {
		app.Error.Println(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) GetUserAccountByName(name string) (*database.Account, error) {
	rows, err := dbConn.Query(`
			SELECT account_info
			FROM users
			WHERE name = $1
		`,
		name,
	)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var jsonStr []byte
		err = rows.Scan(&jsonStr)
		if err != nil {
			app.Error.Println(err)
			return nil, err
		}

		account := database.Account{}
		err := json.Unmarshal(jsonStr, &account)
		if err != nil {
			app.Error.Println(err)
			return nil, err
		}

		return &account, nil
	}

	return nil, nil
}

func (dbConn *CacheLevel2) AddDelegateReturnCron(userName, returnTo string, returnAfter time.Duration, val *types.Asset) (error) {
	returnTime := time.Now().UTC().Add(returnAfter)
	val10x6 := int64(val.Amount * FinanceSaveIndex)

	_, err := dbConn.Do(`
			INSERT INTO delegate_return_cron
				(user_name, return_to, val_10x6, created_at, return_at)
			VALUES
				($1, $2, $3, $4, $5)
		`,
		userName, returnTo, val10x6, time.Now().UTC(), returnTime,
	)

	if err != nil {
		app.Error.Println(err)
		return err
	}

	return nil
}
*/

func (dbConn *CacheLevel2) SaveNewUserNameFromOperation(op *operations.NewUserNameOp, ts time.Time) error {
	userId, err := dbConn.GetUserId(op.Data.Owner)
	if err != nil {
		return errors.Wrap(err, "Error when find user id")
	}

	affected, err := dbConn.Do(`
			UPDATE users_names
			SET name = $1
			WHERE user_id = $2 AND creator = $3
		`,
		op.Data.Name, userId, op.Data.Creator,
	)
	if err != nil {
		return errors.Wrap(err, "Error when update new user name")
	}

	if affected <= 0 {
		_, err = dbConn.Do(`
				INSERT INTO users_names
				(user_id, creator, name)
				VALUES
				($1, $2, $3)
			`,
			userId, op.Data.Creator, op.Data.Name,
		)
		if err != nil {
			return errors.Wrap(err, "Error when insert new user name")
		}
	}

	return nil
}

// Получаем величину текущей батарейки пользователя из "_CYBERWAY_gls_charge.balances" и ".restorers"
func (dbConn *CacheLevel2) GetUserBatteryNodeos(userName string) float64 {
	dbConn.mongo.Check()

	// Get charge balance
	dbConn.mongo.SetDB(cyberdb.ChargeDBName).SetCollection("balances")

	c := dbConn.mongo.Collection

	filter := bson.D{
		{"_SERVICE_.scope", userName},
		{"charge_id", 0},
	}

	cur, err := c.Find(context.TODO(), filter)
	if err != nil {
		app.Error.Printf("Error read charge balance for user: %s : %s", userName, err)
		return 100.0
	}

	var charge cyberdb.ChargeBalanceType
	if cur.Next(context.TODO()) {
		err = cur.Decode(&charge)
		if err != nil {
			app.Error.Printf("Error decode charge balance for user: %s : %s", userName, err)
			return 100.0
		}
	} else {
		// If record absent, then 100% battery
		app.Info.Printf("Charge balance for user absent: %s", userName)
		_ = cur.Close(context.TODO())
		return 100.0
	}
	_ = cur.Close(context.TODO())

	// Get charge restorer
	dbConn.mongo.SetDB(cyberdb.ChargeDBName).SetCollection("restorers")

	c = dbConn.mongo.Collection

	filter = bson.D{
		{"charge_id", 0},
	}

	cur, err = c.Find(context.TODO(), filter)
	if err != nil {
		app.Error.Printf("Error read charge restorer for user: %s : %s", userName, err)
		return 100.0
	}

	var restorer cyberdb.ChargeRestorerType
	if cur.Next(context.TODO()) {
		err = cur.Decode(&restorer)
		if err != nil {
			app.Error.Printf("Error decode charge balance for user: %s : %s", userName, err)
			return 100.0
		}
	} else {
		app.Error.Printf("Error read (absent) charge restorer for user: %s", userName)
		return 100.0
	}
	_ = cur.Close(context.TODO())

	// Calculate restored value of battery
	app.Debug.Printf("Call atmsp for userbattery: (%d, 0, %d)", charge.Value, time.Now().Unix()-cyberdb.Dec128ToInt64(charge.LastUpdate, 1000000))
	f := restorer.Func.SetParams(charge.Value, 0, time.Now().Unix()-cyberdb.Dec128ToInt64(charge.LastUpdate, 1000000))
	restoredVal, _ := f.Run().Float64()

	app.Debug.Printf("Userbattery atmsp result: %f", restoredVal)

	curBattery := (40960000 - float64(restoredVal)) / 409600

	app.Debug.Printf("Userbattery final result: %f", curBattery)

	if curBattery > 100.0 {
		curBattery = 100.0
	}

	return curBattery
}

// Синхронизация данных пользователей по БД mongodb ноды cyberway
// Если такого пользователя нет в нашей БД - добавляем его на основании данных из mongodb
func (dbConn *CacheLevel2) SyncUsersByNames(list []string) {
	for _, userName := range list {
		golos, cyber, vesting, err := dbConn.GetUserBalanceNodeos(userName)
		if err != nil {
			app.Error.Printf("Error get balance for user: %s : %s", userName, err)
			continue
		}

		var golosBalance, cyberBalance, vestingBalance float64
		var vestingDelegatedBalance, vestingReceivedBalance float64
		if golos != nil {
			golosBalance, _ = golos.Balance.GetValue()
		}
		if cyber != nil {
			cyberBalance, _ = cyber.Balance.GetValue()
		}
		if vesting != nil {
			vestingBalance, _ = vesting.Vesting.GetValue()
			vestingDelegatedBalance, _ = vesting.Delegated.GetValue()
			vestingReceivedBalance, _ = vesting.Received.GetValue()
		}

		golosVal := int64(golosBalance * FinanceSaveIndex)
		cyberVal := int64(cyberBalance * FinanceSaveIndex)
		vestingVal := int64(vestingBalance * FinanceSaveIndex)
		vestingDelegatedVal := int64(vestingDelegatedBalance * FinanceSaveIndex)
		vestingReceivedVal := int64(vestingReceivedBalance * FinanceSaveIndex)

		var affected int64
		// В цикле делаем обновление для ситуации если пользователя изначально не существует в нашей БД
		for affected == 0 {
			UpdateUserMutex.Lock()
			affected, err = dbConn.Do(`
				UPDATE users
				SET 
					val_golos_10x6 = $1, 
					val_cyber_10x6 = $2,
					val_power_10x6 = $3,
					val_delegated_10x6 = $4,
					val_received_10x6 = $5
				WHERE
					name = $6
			`,
				golosVal, cyberVal, vestingVal, vestingDelegatedVal, vestingReceivedVal, userName,
			)
			UpdateUserMutex.Unlock()
			if err != nil {
				app.Error.Printf("Error update user balance: %s", err)
			}
			// Проверяем на всякий случай что такого пользователя точно нет
			if affected == 0 {
				id, err := dbConn.GetUserId(userName)
				if err != nil {
					app.Error.Printf("Error get user by id: %s", err)
				}
				if id > 0 {
					affected = 1
				}
			}

			if affected != 1 {
				// Добавляем нового пользователя по информации из mongodb

				// Извлекаем ключи пользователя из mongodb
				keys, err := dbConn.GetUserKeysNodeos(userName)
				if err != nil {
					app.Error.Printf("Error get keys for user: %s : %s", userName, err)
					continue
				}

				names, err := dbConn.GetUserNamesNodeos(userName)
				if err != nil {
					app.Error.Printf("Error get usernames for user: %s : %s", userName, err)
					continue
				}

				// Добавляем пользователя с балансом (т.к. уже известен)
				userId, err := dbConn.Insert(`
						INSERT INTO users
						(name, val_golos_10x6, val_cyber_10x6, val_power_10x6, val_delegated_10x6, val_received_10x6)
						VALUES
						($1, $2, $3, $4, $5, $6)
					`,
					userName, golosVal, cyberVal, vestingVal, vestingDelegatedVal, vestingReceivedVal,
				)
				if err != nil {
					app.Error.Printf("Error insert user: %s : %s", userName, err)
					continue
				}

				// Добавляем ключи пользователя
				for keyType, key := range keys {
					_, err := dbConn.Do(`
							INSERT INTO users_keys
							(user_id, key_type, key)
							VALUES
							($1, $2, $3)
						`,
						userId, keyType, key,
					)
					if err != nil {
						app.Error.Printf("Error insert user key '%s' for user: %s : %s", keyType, userName, err)
						continue
					}
				}

				// Добавляем имена пользователя
				for nameType, name := range names {
					_, err := dbConn.Do(`
							INSERT INTO users_names
							(user_id, creator, name)
							VALUES
							($1, $2, $3)
						`,
						userId, nameType, name,
					)
					if err != nil {
						app.Error.Printf("Error insert username '%s' for user: %s : %s", nameType, userName, err)
						continue
					}
				}
			}
		}
	}
}

func (dbConn *CacheLevel2) GetNodeosName(userName string) string {
	dbConn.mongo.Check()

	dbConn.mongo.SetDB(cyberdb.CyberDBName).SetCollection("username")

	c := dbConn.mongo.Collection

	filter := bson.D{
		{"name", userName},
	}

	baseName := cyberdb.UserNameType{}
	err := c.FindOne(context.TODO(), filter).Decode(&baseName)
	if err != nil {
		return userName
	}

	return baseName.Owner
}

func (dbConn *CacheLevel2) GetUserCreationAge(userName string) int {
	rows, err := dbConn.Query(`SELECT NOW() - time FROM users WHERE name = $1`, userName)
	if err != nil {
		return 0
	}
	defer rows.Close()

	if rows.Next() {
		var t sql.NullTime
		err := rows.Scan(&t)
		if err != nil {
			return 0
		}

		return int(t.Time.Unix())
	}

	return 0
}

func (dbConn *CacheLevel2) GetPowerGOLOSFactors() (float64, float64) {
	// Get usersVesting tokens
	dbConn.mongo.Check()

	dbConn.mongo.SetDB(cyberdb.TokensDBName).SetCollection("accounts")
	c := dbConn.mongo.Collection

	filter := bson.D{
		{"_SERVICE_.scope", "gls.vesting"},
		{"balance._sym", "GOLOS"},
	}

	usersVesting := cyberdb.TokenAccountType{}
	err := c.FindOne(context.TODO(), filter).Decode(&usersVesting)
	if err != nil {
		app.Error.Printf("GetPowerGOLOSFactors error: %s", err)
		return 0, 0
	}

	// Get systemVesting tokens
	dbConn.mongo.SetDB(cyberdb.VestingDBName).SetCollection("stat")
	c = dbConn.mongo.Collection

	filter = bson.D{
		{"_SERVICE_.scope", "gls.vesting"},
		{"supply._sym", "GOLOS"},
	}

	systemVesting := cyberdb.VestingStatType{}
	err = c.FindOne(context.TODO(), filter).Decode(&systemVesting)
	if err != nil {
		app.Error.Printf("GetPowerGOLOSFactors error: %s", err)
		return 0, 0
	}

	usersVestingVal, err := usersVesting.Balance.GetValue()
	if err != nil {
		app.Error.Printf("GetPowerGOLOSFactors error: %s", err)
		return 0, 0
	}

	systemVestingVal, err := systemVesting.Supply.GetValue()
	if err != nil {
		app.Error.Printf("GetPowerGOLOSFactors error: %s", err)
		return 0, 0
	}

	return usersVestingVal, systemVestingVal
}

func (dbConn *CacheLevel2) CalcPowerGOLOS(valPower, valDelegated, valReceived int64) (float64, float64, float64) {
	// Calc
	usersVesting, systemVesting := dbConn.GetPowerGOLOSFactors()

	divFactor := systemVesting * FinanceSaveIndex
	power := float64(valPower) * usersVesting / divFactor
	delegated := float64(valDelegated) * usersVesting / divFactor
	received := float64(valReceived) * usersVesting / divFactor

	return power, delegated, received
}
