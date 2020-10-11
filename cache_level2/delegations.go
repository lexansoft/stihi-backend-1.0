package cache_level2

/*
func (dbConn *CacheLevel2) SaveDelegateVestingSharesOperation(fromUserId int64, toUserId int64, op *types.DelegateVestingSharesOperation, trx *types.Transaction, block *database.Block) error {
	opTime := types.Time{}.Local()
	if block != nil {
		opTime = *block.Timestamp.Time
	}

	_, err := dbConn.Insert(
		`
			INSERT INTO delegation_balance
				(from_user_id, to_user_id, val_10x6, updated_at)
			VALUES
				($1, $2, $3, $4)
			ON CONFLICT
				(from_user_id, to_user_id)
			DO UPDATE SET
				val_10x6 = EXCLUDED.val_10x6,
				updated_at = EXCLUDED.updated_at
		`,
		fromUserId, toUserId, op.VestingShares.Amount, opTime,
	)
	if err != nil {
		app.Error.Print(err)
		return err
	}
	return nil
}

func (dbConn *CacheLevel2) SaveReturnVestingDelegationOperation(toUserId int64, op *types.ReturnVestingDelegationOperation, trx *types.Transaction, block *database.Block) error {
	return nil
}
*/