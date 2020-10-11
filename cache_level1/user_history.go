package cache_level1

import (
	"gitlab.com/stihi/stihi-backend/cache_level2"
)

func (cache *CacheLevel1) GetUserHistory(login string, offset, count int64) (*[]*cache_level2.HistoryRecord, error) {
	userId, err 	:= cache.GetUserId(login); 		if err != nil { return nil, err }

	return cache.Level2.GetUserHistory(userId, offset, count)
}

/*
func (cache *CacheLevel1) AddToHistoryTransferOperation(op *types.TransferOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.From); 	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.To); 		if err != nil { return err }

	return cache.Level2.AddToHistoryTransferOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryTransferFromSavingsOperation(op *types.TransferFromSavingsOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.From); 	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.To); 		if err != nil { return err }

	return cache.Level2.AddToHistoryTransferFromSavingsOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryTransferToSavingsOperation(op *types.TransferToSavingsOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.From); 	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.To); 		if err != nil { return err }

	return cache.Level2.AddToHistoryTransferToSavingsOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryTransferToVestingOperation(op *types.TransferToVestingOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.From); 	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.To); 		if err != nil { return err }

	return cache.Level2.AddToHistoryTransferToVestingOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryWithdrawVestingOperation(op *types.WithdrawVestingOperation, trx *types.Transaction, block *database.Block) error {
	toUserId, err 	:= cache.GetUserId(op.Account);	if err != nil { return err }

	return cache.Level2.AddToHistoryWithdrawVestingOperation(toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryFillVestingWithdrawOperation(op *types.FillVestingWithdrawOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.FromAccount); 	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.ToAccount); 		if err != nil { return err }

	return cache.Level2.AddToHistoryFillVestingWithdrawOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryFillTransferFromSavingsOperation(op *types.FillTransferFromSavingsOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.From); 	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.To); 		if err != nil { return err }

	return cache.Level2.AddToHistoryFillTransferFromSavingsOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryAccountCreateOperation(op *types.AccountCreateOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.Creator); 		if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.NewAccountName); 	if err != nil { return err }

	return cache.Level2.AddToHistoryAccountCreateOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryAccountCreateWithDelegationOperation(op *types.AccountCreateWithDelegationOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.Creator); 		if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.NewAccountName); 	if err != nil { return err }

	return cache.Level2.AddToHistoryAccountCreateWithDelegationOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryDelegateVestingSharesOperation( op *types.DelegateVestingSharesOperation, trx *types.Transaction, block *database.Block) error {
	fromUserId, err := cache.GetUserId(op.Delegator);	if err != nil { return err }
	toUserId, err 	:= cache.GetUserId(op.Delegatee); 	if err != nil { return err }

	return cache.Level2.AddToHistoryDelegateVestingSharesOperation(fromUserId, toUserId, op, trx, block)
}

func (cache *CacheLevel1) AddToHistoryReturnVestingDelegationOperation( op *types.ReturnVestingDelegationOperation, trx *types.Transaction, block *database.Block) error {
	toUserId, err := cache.GetUserId(op.Account);	if err != nil { return err }

	return cache.Level2.AddToHistoryReturnVestingDelegationOperation(toUserId, op, trx, block)
}
*/