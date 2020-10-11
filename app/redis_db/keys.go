package redis_db

import "gitlab.com/stihi/stihi-backend/app"

// Возвращает порцию ключей подходящую под шаблон
// Вызов должен быть итерационным - current_cursor, возвращенный на текущем шаге нужно передавать на следующий
// Инициирующий курсор должен быть "0"
// Если возвращается курсор "0", значит текущая порция данных последняя

type KeysScanner struct {
	Conn		*RedisConnection
	Template    string
	Count       int64
	ServerIndex int
	Cursor      uint64
	Finished    bool
}

func (conn *RedisConnection) NewKeysScaner(template string, count int64) (*KeysScanner) {
	k := &KeysScanner{}

	k.Conn = conn
	k.Template = template
	k.Count = count - 1
	k.ServerIndex = 0
	k.Finished = false

	return k
}

func (k *KeysScanner) Next() []string {
	var err error
	var result []string

	for len(result) <= 0 {
		server := k.Conn.MainServers[k.ServerIndex]
		result, k.Cursor, err = server.Connection.Scan(k.Cursor, k.Template, k.Count).Result()
		if err != nil {
			app.Error.Print("Error Redis Scan keys: ", err)
		}

		if k.Cursor == 0 {
			k.ServerIndex++
			if k.ServerIndex >= len(k.Conn.MainServers) {
				k.Finished = true
				return result
			}
		}
	}

	return result
}
