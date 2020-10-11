package cache_level2

import (
	"database/sql"

	"gitlab.com/stihi/stihi-backend/app"
)

const (
	AccountCreateOperationName					= "account_create"
	AccountCreateWithDelegationOperationName	= "account_create_with_delegation"
	TransferOperationName 						= "transfer"
	TransferFromSavingsOperationName 			= "transfer_from_savings"
	TransferToSavingsOperationName 				= "transfer_to_savings"
	TransferToVestingOperationName 				= "transfer_to_vesting"
	WithdrawVestingOperationName				= "withdraw_vesting"
	FillVestingWithdrawOperationName			= "fill_vesting_withdraw"
	FillTransferFromSavings						= "fill_transfer_from_savings"
	DelegateVestingSharesOperationName			= "delegate_vesting_shares"
	ReturnVestingDelegationOperationName		= "return_vesting_delegation"
)

func (dbConn *CacheLevel2) GetUserHistory(userId, offset, count int64) (*[]*HistoryRecord, error) {
	list := make([]*HistoryRecord, 0)

	rows, err := dbConn.Query(`
		SELECT 
			uh.user_id, u1.name as user_name,
			uh.from_user_id, u2.name as from_user_name, 
			uh.val_cyber_change_10x6, uh.val_golos_change_10x6, uh.val_power_change_10x6, 
			uh.content_id, c.author, c.permlink,
			uh.operation_type, uh.time
		FROM
			users_history uh
			LEFT JOIN users u1 ON u1.id = uh.user_id
			LEFT JOIN users u2 ON u2.id = uh.from_user_id
			LEFT JOIN content c ON c.id = uh.content_id
		WHERE
			( user_id = $1 OR from_user_id = $2 )
		ORDER BY time DESC
		LIMIT $3
		OFFSET $4
	`,
		userId, userId, count, offset,
	)
	if err != nil {
		app.EmailErrorf(err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userId, fromUserId sql.NullInt64
		var userName, fromUserName sql.NullString
		var cyberChange, golosChange, powerChange int64
		var contentId sql.NullInt64
		var contentAuthor, contentPermlink sql.NullString
		var operationType string
		var operationTime NullTime

		err = rows.Scan(
			&userId, &userName,
			&fromUserId, &fromUserName,
			&cyberChange, &golosChange, &powerChange,
			&contentId, &contentAuthor, &contentPermlink,
			&operationType,	&operationTime,
		)
		if err != nil {
			app.EmailErrorf(err.Error())
			return nil, err
		}

		history := HistoryRecord{
			OpType: operationType,
			OpTime: operationTime.Format(),
			ToUser: userName.String,
			ToUserId: userId.Int64,
			CyberChange: float64(cyberChange) / FinanceSaveIndex,
			GolosChange: float64(golosChange) / FinanceSaveIndex,
			PowerChange: float64(powerChange) / FinanceSaveIndex,
		}

		if fromUserId.Valid && fromUserName.Valid {
			history.FromUserId = fromUserId.Int64
			history.FromUser = fromUserName.String
		}

		if contentId.Valid && contentAuthor.Valid && contentPermlink.Valid {
			history.ContentId = contentId.Int64
			history.ContentAuthor = contentAuthor.String
			history.ContentPermlink = contentPermlink.String
		}

		list = append(list, &history)
	}

	return &list, nil
}

/*
func (dbConn *CacheLevel2) AddToHistoryTransferOperation(fromUserId, toUserId int64, op *types.TransferOperation, trx *types.Transaction, block *database.Block) error {
	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Amount)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Do(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), -cyberChange, -golosChange, -powerChange, -1, TransferOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryTransferFromSavingsOperation(fromUserId, toUserId int64, op *types.TransferFromSavingsOperation, trx *types.Transaction, block *database.Block) error {
	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Amount)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Do(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), cyberChange, golosChange, powerChange, -1, TransferFromSavingsOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryTransferToSavingsOperation(fromUserId, toUserId int64, op *types.TransferToSavingsOperation, trx *types.Transaction, block *database.Block) error {
	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Amount)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Do(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), -cyberChange, -golosChange, -powerChange, -1, TransferToSavingsOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryTransferToVestingOperation(fromUserId, toUserId int64, op *types.TransferToVestingOperation, trx *types.Transaction, block *database.Block) error {
	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Amount)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Do(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), -cyberChange, -golosChange, -powerChange, -1, TransferToVestingOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryWithdrawVestingOperation(fromUserId int64, op *types.WithdrawVestingOperation, trx *types.Transaction, block *database.Block) error {
	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.VestingShares)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Do(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		fromUserId, fromUserId, string(jsonOp), -cyberChange, -golosChange, -powerChange, -1, WithdrawVestingOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryFillVestingWithdrawOperation(fromUserId, toUserId int64, op *types.FillVestingWithdrawOperation, trx *types.Transaction, block *database.Block) error {
	var powerChange int64
	var golosChange int64

	_, _, powerChange = AddPayOut(0, 0, 0, op.Withdrawn)
	_, golosChange, _ = AddPayOut(0, 0, 0, op.Deposited)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Do(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), 0, golosChange, -powerChange, -1, FillVestingWithdrawOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryFillTransferFromSavingsOperation(fromUserId, toUserId int64, op *types.FillTransferFromSavingsOperation, trx *types.Transaction, block *database.Block) error {
	var powerChange int64
	var golosChange int64
	var cyberChange int64

	cyberChange, golosChange, powerChange = AddPayOut(0, 0, 0, op.Amount)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), cyberChange, golosChange, powerChange, -1, FillTransferFromSavings, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryAccountCreateOperation(fromUserId, toUserId int64, op *types.AccountCreateOperation, trx *types.Transaction, block *database.Block) error {
	var powerChange int64
	var golosChange int64
	var cyberChange int64

	cyberChange, golosChange, powerChange = AddPayOut(0, 0, 0, op.Fee)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), cyberChange, golosChange, powerChange, -1, AccountCreateOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryAccountCreateWithDelegationOperation(fromUserId, toUserId int64, op *types.AccountCreateWithDelegationOperation, trx *types.Transaction, block *database.Block) error {
	var powerChange int64
	var golosChange int64
	var cyberChange int64

	cyberChange, golosChange, powerChange = AddPayOut(0, 0, 0, op.Fee)
	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Delegation)

	jsonOp, err := json.Marshal(*op)

	_, err = dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), cyberChange, golosChange, powerChange, -1, AccountCreateWithDelegationOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryDelegateVestingSharesOperation(fromUserId int64, toUserId int64, op *types.DelegateVestingSharesOperation, trx *types.Transaction, block *database.Block) error {
	jsonOp, err := json.Marshal(*op)

	var powerChange int64
	var golosChange int64
	var cyberChange int64

	cyberChange, golosChange, powerChange = AddPayOut(0, 0, 0, op.VestingShares)

	_, err = dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, fromUserId, string(jsonOp), cyberChange, golosChange, powerChange, -1, DelegateVestingSharesOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}

func (dbConn *CacheLevel2) AddToHistoryReturnVestingDelegationOperation(toUserId int64, op *types.ReturnVestingDelegationOperation, trx *types.Transaction, block *database.Block) error {
	jsonOp, err := json.Marshal(*op)

	var powerChange int64
	var golosChange int64
	var cyberChange int64

	cyberChange, golosChange, powerChange = AddPayOut(0, 0, 0, op.VestingShares)

	_, err = dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, from_user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		toUserId, -1, string(jsonOp), cyberChange, golosChange, powerChange, -1, ReturnVestingDelegationOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}

	return nil
}


 */