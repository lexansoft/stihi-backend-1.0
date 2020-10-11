package cache_level2

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/valyala/fastjson"
	"unicode/utf8"

	"gitlab.com/stihi/stihi-backend/app"
)

type CyberwayBlock map[string]interface{}

func (b CyberwayBlock) Value() (driver.Value, error) {
	return json.Marshal(b)
}

func (b *CyberwayBlock) Scan(value interface{}) error {
	data, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(data, &b)
}

func (dbConn *CacheLevel2) SaveBlock(blockNum int64, content string, virtOps string) error {
	/*
		Для защиты от неверного JSON формата в блоке разделяем операцию сохранения сырого блока и операцию конвертации
		данных в jsonb.

		В случае проблем с форматом, сырой вариант будет в базе.
	*/

	/* Проверяем на проблемы с UTF-8 */
	if !utf8.ValidString(content) {
		app.Error.Printf("Block num %d content with bad UTF-8 codes!!!\n%s", blockNum, content)
		return nil
	}

	_, err := dbConn.Do(`
		INSERT INTO blockchain_cyber
			(num, block)
		VALUES
			($1, $2)
		ON CONFLICT DO NOTHING
		`,
		blockNum, content,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	if isJSONString(content) {
		_, err = dbConn.Do(`
   			UPDATE blockchain_cyber
			SET
				block_json = block::jsonb
			WHERE
				num = $1
			`,
			blockNum,
		)
		if err != nil {
			// Вывод ошибки чисто диагностический
			app.Error.Print(err)
		}
	}

	return nil
}

func (dbConn *CacheLevel2) LastBlockNum() (int64, error) {
	rows, err := dbConn.Query(`SELECT MAX(num) FROM blockchain_cyber`)
	if err != nil {
		app.Error.Println(err)
		return -1, err
	}
	defer rows.Close()

	if rows.Next() {
		var num sql.NullInt64
		err = rows.Scan(&num)
		if err != nil {
			app.Error.Println(err)
			return -1, err
		}
		if num.Valid {
			return num.Int64, nil
		}
	}

	return 1, nil
}

func (dbConn *CacheLevel2) GetBlock(blockNum int64) (*map[string]interface{}, error) {
	block := CyberwayBlock{}

	rows, err := dbConn.Query(`
		SELECT 
			block_json 
		FROM 
			blockchain_cyber
		WHERE
			num = $1`,
		blockNum,
	)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&block)
		if err != nil {
			app.Error.Println(err)
			return nil, err
		}
	}

	val := map[string]interface{}(block)
	return &val, nil
}

func (dbConn *CacheLevel2) GetBlocks(blockNum int64, count int) ([]*map[string]interface{}, error) {
	list := make([]*map[string]interface{}, 0, count)

	rows, err := dbConn.Query(`
		SELECT 
			block_json 
		FROM 
			blockchain_cyber
		WHERE
			num >= $1
		ORDER BY num
		LIMIT $2`,
		blockNum, count,
	)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		block := CyberwayBlock{}
		err = rows.Scan(&block)
		if err != nil {
			app.Error.Println(err)
			return nil, err
		}
		val := map[string]interface{}(block)
		list = append(list, &val)
	}

	return list, nil
}


type BlockchainOp struct {
	BlockNum int64
	OpType string
	OpData map[string]interface{}
	OpTime string
}

// TODO: Переделать под формат блока Cyberway
func (dbConn *CacheLevel2) GetOpsFromBlocks(startBlockNum int64, blocksCount int32) (*[]*BlockchainOp, error) {
	list := make([]*BlockchainOp, 0)

	// Реальные операции
	rows, err := dbConn.Query(`
		SELECT
 			b1.num,
 			b1.op->0 as op_type,
 			b1.op->1 as op_data,
			b2.block_json->'timestamp' as time
	 	FROM (
	 		SELECT
	 			num,
	 			jsonb_array_elements(jsonb_array_elements(block_json->'transactions')->'operations') as op
	 		FROM blockchain_cyber
			WHERE
				num >= $1 AND num < $2
 			ORDER by num) AS b1, blockchain_cyber b2
		WHERE b1.num = b2.num`,

		startBlockNum,
		startBlockNum + int64(blocksCount),
	)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}

	for rows.Next() {
		op := BlockchainOp{ OpData: make(map[string]interface{}) }

		err = rows.Scan(
			&op.BlockNum,
			&op.OpType,
			&op.OpData,
			&op.OpTime,
		)
		if err != nil {
			app.Error.Println(err)
			rows.Close()
			return nil, err
		}

		list = append(list, &op)
	}
	rows.Close()

	// Виртуальные операции
	rows, err = dbConn.Query(`
		SELECT
 			num,
 			r->'op'->0 as op_type,
 			r->'op'->1 as op_data,
			r->'timestamp' as time
	 	FROM (
	 		SELECT
	 			num,
	 			jsonb_array_elements(virtual_json) as r
	 		FROM blockchain_cyber
			WHERE
				num >= $1 AND num < $2
 			ORDER by num) AS t1`,
		startBlockNum,
		startBlockNum + int64(blocksCount),
	)
	if err != nil {
		app.Error.Println(err)
		return nil, err
	}

	for rows.Next() {
		op := BlockchainOp{ OpData: make(map[string]interface{}) }

		err = rows.Scan(
			&op.BlockNum,
			&op.OpType,
			&op.OpData,
			&op.OpTime,
		)
		if err != nil {
			app.Error.Println(err)
			rows.Close()
			return nil, err
		}

		list = append(list, &op)
	}
	rows.Close()

	return &list, nil
}

// Выборка всех операций c их типами в порядке следования их блоков и их нахождения в блоке:
// SELECT
// 		num,
// 		op->0 as op_type,
// 		op->1 as op_data
// 	FROM (
// 		SELECT
// 			num,
// 			jsonb_array_elements(jsonb_array_elements(block_json->'transactions')->'operations') as op
// 		FROM blockchain
// 		ORDER by num) AS t1;

func isJSONString(s string) bool {
	var p fastjson.Parser
	_, err := p.Parse(s)
	return err == nil
}
