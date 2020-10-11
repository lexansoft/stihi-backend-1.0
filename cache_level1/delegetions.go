package cache_level1

/*
func (cache *CacheLevel1) SaveDelegateVestingSharesOperation( op *types.DelegateVestingSharesOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.Delegator);	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.Delegatee); 	if err != nil { return err }

	return cache.Level2.SaveDelegateVestingSharesOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) SaveReturnVestingDelegationOperation( op *types.ReturnVestingDelegationOperation, trx *types.Transaction, block *database.Block) error {
	toUserId, err := cache.GetUserId(op.Account);	if err != nil { return err }

	return cache.Level2.SaveReturnVestingDelegationOperation(toUserId, op, trx, block)
}
*/