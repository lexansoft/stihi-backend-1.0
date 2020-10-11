package cache_level1

import (
	"errors"
	"math"
	"strconv"
	"time"

	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/cyber/operations"
)

const (
	ExpirationUserPeriodLeader = 30 * time.Minute
	ExpirationUserBatteryPeriod = 10 * time.Minute
)

// TODO: Сделать кэширующий вариант

func (cache *CacheLevel1) CreateUser(cyberName, login, ownerPubKey, activePubKey, postingPubKey string, ts time.Time) error {
	// Проверка юзера на существование
	userId, err := cache.GetUserId(login)
	if userId > 0 && err == nil {
		return errors.New("l10n:authorize.already_exists")
	}

	return cache.Level2.CreateUser(cyberName, login, ownerPubKey, activePubKey, postingPubKey, ts)
}

func (cache *CacheLevel1) SaveUserFromOperation(op *operations.NewAccountOp, ts time.Time) error {
	return cache.Level2.SaveUserFromOperation(op, ts)
}

func (cache *CacheLevel1) UpdateUserAuthFromOperation(op *operations.UpdateAuthOp, ts time.Time) error {
	return cache.Level2.UpdateUserAuthFromOperation(op, ts)
}

func (cache *CacheLevel1) SaveUser(user *cache_level2.User, ts time.Time) error {
	return cache.Level2.SaveUser(user, ts)
}

func (cache *CacheLevel1) GetUserId(name string) (int64, error) {
	return cache.Level2.GetUserId(name)
}

func (cache *CacheLevel1) GetUserByName(name string) (*cache_level2.User, error) {
	return cache.Level2.GetUserByName(name)
}

func (cache *CacheLevel1) GetUserNameById(id int64) (string, error) {
	return cache.Level2.GetUserNameById(id)
}

func (cache *CacheLevel1) IsKeysExists(key string) (bool, error) {
	return cache.Level2.IsKeysExists(key)
}

func (cache *CacheLevel1) GetUserInfo(id int64) (*cache_level2.UserInfo, error) {
	return cache.Level2.GetUserInfo(id)
}

func (cache *CacheLevel1) GetUserInfoByName(name string) (*cache_level2.UserInfo, error) {
	return cache.Level2.GetUserInfoByName(name)
}

func (cache *CacheLevel1) UpdateUserInfo(userInfo *cache_level2.UserInfo) (error) {
	return cache.Level2.UpdateUserInfo(userInfo)
}

func (cache *CacheLevel1) UpdateUserBattery(userInfo *cache_level2.UserInfo) (error) {
	return cache.Level2.UpdateUserBattery(userInfo)
}


//////////////////////////////////////////////////
func (cache *CacheLevel1) GetNewUsersListLast(count int, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNewUsersListLast(count, filter)
}

func (cache *CacheLevel1) GetNameUsersListLast(count int, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNameUsersListLast(count, filter)
}

func (cache *CacheLevel1) GetNicknameUsersListLast(count int, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNicknameUsersListLast(count, filter)
}

func (cache *CacheLevel1) GetNewUsersListAfter(lastUserId int64, count int, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNewUsersListAfter(lastUserId, count, filter)
}

func (cache *CacheLevel1) GetNameUsersListAfter(lastUserId int64, count int, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNameUsersListAfter(lastUserId, count, filter)
}

func (cache *CacheLevel1) GetNicknameUsersListAfter(lastUserId int64, count int, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNicknameUsersListAfter(lastUserId, count, filter)
}

func (cache *CacheLevel1) GetNewUsersListBefore(lastUserId int64, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNewUsersListBefore(lastUserId, filter)
}

func (cache *CacheLevel1) GetNameUsersListBefore(lastUserId int64, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNameUsersListBefore(lastUserId, filter)
}

func (cache *CacheLevel1) GetNicknameUsersListBefore(lastUserId int64, filter string) (*[]*cache_level2.UserInfo, error) {
	return cache.Level2.GetNicknameUsersListBefore(lastUserId, filter)
}
//////////////////////////////////////////////////

func (cache *CacheLevel1) UpdateUserBalances(name string, balance cache_level2.Balance, reputation int64, keys ...string) error {
	return cache.Level2.UpdateUserBalances(name, balance, reputation, keys...)
}

func (cache *CacheLevel1) ChangeUserBalances(name string, balance cache_level2.Balance) error {
	return cache.Level2.ChangeUserBalances(name, balance)
}

func (cache *CacheLevel1) GetUserPeriodLeader(days int) (int64, error) {
	key := "user_period_leader:"+strconv.FormatInt(int64(days), 10)

	var cached int64
	err := cache.GetObject(key, &cached)
	if err == nil && cached > 0 {
		return cached, nil
	}

	cache.Lock(key)
	defer cache.Unlock(key)

	// После лока сначала проверяем кэш, а затем, если в кэше данных нет, загружаем из БД
	err = cache.GetObject(key, &cached)
	if err == nil && cached > 0 {
		return cached, nil
	}

	userId, err := cache.Level2.GetUserPeriodLeader(days)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	err = cache.SaveEx(key, userId, ExpirationUserPeriodLeader)
	if err != nil {
		app.Error.Print(err)
	}

	return userId, nil
}

func (cache *CacheLevel1) SetStihiUser(userId int64) error {
	return cache.Level2.SetStihiUser(userId)
}

func (cache *CacheLevel1) SetStihiUserByLogin(login string) error {
	return cache.Level2.SetStihiUserByLogin(login)
}

func (cache *CacheLevel1) StihiUserListFilter(names []string) []string {
	return cache.Level2.StihiUserListFilter(names)
}

func (cache *CacheLevel1) SaveNewUserNameFromOperation(op *operations.NewUserNameOp, ts time.Time) error {
	return cache.Level2.SaveNewUserNameFromOperation(op, ts)
}

// Синхронизация данных пользователей по БД mongodb ноды cyberway
func (cache *CacheLevel1) SyncUsersByNames(list []string) {
	cache.Level2.SyncUsersByNames(list)
}

func (cache *CacheLevel1) GetUserNames(user *cache_level2.User) error {
	return cache.Level2.GetUserNames(user)
}

func (cache *CacheLevel1) GetUserBatteryNodeos(userName string) (float64, error) {
	key := "user_battery_val:"+userName

	var err error
	var cached float64
	err = cache.GetObject(key, &cached)
	if err == nil && cached > 0 {
		return cached, nil
	}

	val := cache.Level2.GetUserBatteryNodeos(userName)
	if math.IsInf(val, 0) {
		app.Error.Printf("Infinity values in battery!!!")
		val = 100.0
	}

	err = cache.SaveEx(key, val, ExpirationUserBatteryPeriod)
	if err != nil {
		app.Error.Print(err)
	}

	return val, nil
}

func (cache *CacheLevel1) ResetCacheUserBatteryNodeos(userName string) {
	key := "user_battery_val:"+userName

	_ = cache.Clean(key)
}

func (cache *CacheLevel1) GetNodeosName(userName string) string {
	return cache.Level2.GetNodeosName(userName)
}

func (cache *CacheLevel1) GetGolosLogin(userName string) string {
	user, err := DB.GetUserByName(userName)
	if err != nil {
		return userName
	}

	if user != nil && user.Names != nil {
		golosLogin, ok := user.Names["gls"]
		if ok {
			return golosLogin
		}
	}

	return userName
}

func (cache *CacheLevel1) GetUserCreationAge(userName string) int {
	return cache.Level2.GetUserCreationAge(userName)
}

func (cache *CacheLevel1) GetPowerGOLOSFactors() (float64, float64) {
	return cache.GetPowerGOLOSFactors()
}

func (cache *CacheLevel1) CalcPowerGOLOS(valPower, valDelegated, valReceived int64) (float64, float64, float64) {
	return cache.Level2.CalcPowerGOLOS(valPower, valDelegated, valReceived)
}
