package cache_level2

const (
	AuthorRewardOperationName = "author_reward"
	CommentRewardOperationName = "comment_reward"
	CurationRewardOperationName = "curation_reward"
)
/*
func (dbConn *CacheLevel2) SaveAuthorRewardFromOperation(op *types.AuthorRewardOperation, trx *types.Transaction, block *database.Block) (int64, error) {
	contentId, err := dbConn.GetContentId(op.Author, op.Permlink)
	if err != nil || contentId == -1 {
		if err != nil {
			app.Error.Print(err)
		}
		return -1, nil
	}

	userId, err := dbConn.GetUserId(op.Author)
	if err != nil || userId == -1 {
		if err != nil {
			app.Error.Print(err)
		}
		return -1, nil
	}

	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.SbdPayout)
	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.SteemPayout)
	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.VestingPayout)

	jsonOp, err := json.Marshal(*op)
	id, err := dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		userId, string(jsonOp), cyberChange, golosChange, powerChange, contentId, AuthorRewardOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	return id, nil
}

func (dbConn *CacheLevel2) SaveCommentRewardFromOperation(op *types.CommentRewardOperation, trx *types.Transaction, block *database.Block) (int64, error) {
	contentId, err := dbConn.GetContentId(op.Author, op.Permlink)
	if err != nil || contentId == -1 {
		if err != nil {
			app.Error.Print(err)
		}
		return -1, nil
	}

	userId, err := dbConn.GetUserId(op.Author)
	if err != nil || userId == -1 {
		app.Error.Print(err)
		return -1, nil
	}

	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Payout)

	jsonOp, err := json.Marshal(*op)
	id, err := dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		userId, string(jsonOp), cyberChange, golosChange, powerChange, contentId, CommentRewardOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	return id, nil
}


func (dbConn *CacheLevel2) SaveCurationRewardFromOperation(op *types.CurationRewardOperation, trx *types.Transaction, block *database.Block) (int64, error) {
	contentId, err := dbConn.GetContentId(op.CommentAuthor, op.CommentPermlink)
	if err != nil || contentId == -1 {
		if err != nil {
			app.Error.Print(err)
		}
		return -1, nil
	}

	userId, err := dbConn.GetUserId(op.Curator)
	if err != nil || userId == -1 {
		if err != nil { app.Error.Print(err) }
		return -1, nil
	}

	var cyberChange int64
	var golosChange int64
	var powerChange int64

	cyberChange, golosChange, powerChange = AddPayOut(cyberChange, golosChange, powerChange, op.Reward)

	jsonOp, err := json.Marshal(*op)
	id, err := dbConn.Insert(
		`
			INSERT INTO users_history
				(user_id, operation, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6, content_id, operation_type, block_num, time)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT
				(user_id, from_user_id, content_id, operation_type, block_num, val_cyber_change_10x6, val_golos_change_10x6, val_power_change_10x6)
			DO NOTHING
		`,
		userId, string(jsonOp), cyberChange, golosChange, powerChange, contentId, CurationRewardOperationName, block.Number, block.Timestamp.Time,
	)
	if err != nil {
		app.Error.Print(err)
		return -1, err
	}

	return id, nil
}


func (dbConn *CacheLevel2) IsRewardExists(block *database.Block, contentId int64, userId int64, op_type string) bool {
	rows, err := dbConn.Query(
		`
			SELECT 
				id
			FROM 
				users_history
			WHERE 
				operation_type = $1 AND
				content_id = $2 AND
				user_id = $3 AND
				block_num = $4
		`,
		op_type, contentId, userId, block.Number,
	)
	if err != nil {
		app.Error.Printf("Check %s exists error: %s", op_type, err)
		return false
	}
	defer rows.Close()

	if rows.Next() {
		return true
	}

	return false
}

func AddPayOut(cyber_begin int64, golos_begin int64, power_begin int64, val *types.Asset) (int64, int64, int64) {
	if val != nil {
		switch val.Symbol {
		case "GBG":
			cyber_begin += int64(val.Amount * FinanceSaveIndex)
		case "GOLOS":
			golos_begin += int64(val.Amount * FinanceSaveIndex)
		case "GESTS":
			power_begin += int64(val.Amount * FinanceSaveIndex)
		}
	}

	return cyber_begin, golos_begin, power_begin
}

 */