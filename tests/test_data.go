package tests

import (
	"gitlab.com/stihi/stihi-backend/app/db"
	"log"
	"gitlab.com/stihi/stihi-backend/app"
	"gitlab.com/stihi/stihi-backend/actions"
	"gitlab.com/stihi/stihi-backend/app/errors_l10n"
	"gitlab.com/stihi/stihi-backend/cache_level1"
		"gitlab.com/stihi/stihi-backend/app/config"
	"gitlab.com/stihi/stihi-backend/app/jwt"
)

var (
	backendConfig config.BackendConfig
	article1Id, article2Id, article3Id, article4Id int64
	npArticle1Id, npArticle2Id, npArticle3Id, npArticle4Id, npArticle5Id int64
	dbConn *db.Connection
)

func init() {
	// Настройки JWT
	app.LoadFromConfig("../configs/stihi_backend_test_config.yaml", &backendConfig)
	actions.Config = &backendConfig

	err := jwt.Init(&backendConfig.JWT)
	if err != nil {
		log.Fatalln(err)
	}

	// l10n для сообщений об ошибках
	for _, langFile := range backendConfig.L10NErrors {
		errors_l10n.LoadL10NBase(langFile.Lang, langFile.FileName)
	}

	// Настройки DB
	cache_level1.DB, err = cache_level1.New("../configs/redis_test_config.yaml", "../configs/db_test_config.yaml")
	if err != nil {
		app.Error.Fatalln(err)
	}
	actions.DB = cache_level1.DB

	cache_level1.DB.Level2.RunMigrations(true)

	dbC, ok := cache_level1.DB.Level2.QueryProcessor.(*db.Connection)
	if !ok {
		app.Error.Fatalln("Bad type of QueryProcessor")
	}
	dbConn = dbC

	InitTestUsers(dbConn)
	InitTestArticles(dbConn)
}

func InitTestUsers(dbConn *db.Connection) {
	dbConn.Do(`DELETE FROM users;`)
	dbConn.Do(`
		INSERT INTO users 
			(name, owner_key, active_key, posting_key) 
		VALUES 
			('test-user1', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6'),
			('test-user2', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6'),
			('test-user3', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6'),
			('test-user4', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6', 'GLS58g5rWYS3XFTuGDSxLVwiBiPLoAyCZgn6aB9Ueh8Hj5qwQA3r6');`)

}

func InitTestArticles(dbConn *db.Connection) {
	// Добавлям в БД несколько записей статей для разных проверок
	var err error
	dbConn.Do(`DELETE FROM content;`)
	article1Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user1', 'permlink1', 'Title 1', 'Body 1', '2018-04-26 15:00:00', 't', 0,
			0, 0, 'http://imghosting.net/img1.jpg', NULL, 0, 0, 0, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	article2Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user2', 'permlink2', 'Title 2', 'Body 2', '2018-04-26 16:00:00', 't', 10,
			10000, 0, 'http://imghosting.net/img2.jpg', '2018-04-30 12:00:00', 100, 10, 0, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	article3Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user3', 'permlink3', 'Title 3', 'Body 3', '2018-04-27 12:00:00', 't', 5,
			5000, 0, 'http://imghosting.net/img3.jpg', '2018-04-27 18:00:00', 10, 5, 0, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	article4Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user4', 'permlink4', 'Title 4', 'Body 4', '2018-04-28 12:00:00', 't', 15,
			5000, -10000, 'http://imghosting.net/img4.jpg', '2018-04-29 18:00:00', 500, 5, 10, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}

	// Тэги
	_, err = dbConn.Do(`INSERT INTO content_tags 
		(content_id, tag) VALUES
		($1, 'tag1'),
        ($2, 'tag2'),
        ($3, 'tag3'),
        ($4, 'tag4'),
        ($5, 'testmat');
	`, article1Id, article1Id, article1Id, article1Id, article1Id)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	_, err = dbConn.Do(`INSERT INTO content_tags 
		(content_id, tag) VALUES
		($1, 'tag11'),
        ($2, 'tag2'),
        ($3, 'tag3'),
        ($4, 'testmat');
	`, article2Id, article2Id, article2Id, article2Id)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	_, err = dbConn.Do(`INSERT INTO content_tags 
		(content_id, tag) VALUES
		($1, 'tag11'),
        ($2, 'tag21'),
        ($3, 'tag3'),
        ($4, 'tag4');
	`, article3Id, article3Id, article3Id, article3Id)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}

	// Голоса
	_, err = dbConn.Do(`DELETE FROM content_votes`)
	_, err = dbConn.Do(`INSERT INTO content_votes 
		(content_id, voter, weight, time) VALUES
		($1, 'test-user1', 10000, NOW()-'2 minutes'::interval),
        ($2, 'test-user1', 10000, NOW()-'1 minutes'::interval),
        ($3, 'test-user1', -10000, NOW())
	`, article1Id, article3Id, article4Id)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
}

func InitTestArticlesNextPrev(dbConn *db.Connection) {
	// Добавлям в БД несколько записей статей для разных проверок
	var err error
	npArticle1Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user1', 'permlink1np', 'Title 1 np', 'Body 1', '2010-04-27 15:00:00', 't', 0,
			0, 0, 'http://imghosting.net/img1.jpg', NULL, 0, 0, 0, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	npArticle2Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user1', 'permlink2np', 'Title 2 np', 'Body 2', '2010-04-27 16:00:00', 't', 10,
			10000, 0, 'http://imghosting.net/img2.jpg', '2018-04-30 12:00:00', 100, 10, 0, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	npArticle3Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user2', 'permlink3np', 'Title 3 np', 'Body 3', '2010-04-27 17:00:00', 't', 5,
			5000, 0, 'http://imghosting.net/img3.jpg', '2018-04-27 18:00:00', 10, 5, 0, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	npArticle4Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user1', 'permlink4np', 'Title 4 np', 'Body 4', '2010-04-27 18:00:00', 't', 15,
			5000, -10000, 'http://imghosting.net/img4.jpg', '2018-04-29 18:00:00', 500, 5, 10, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
	npArticle5Id, err = dbConn.Insert(`
		INSERT INTO articles 
			(parent_author, parent_permlink, author, permlink, title, body, time, confirmed, votes_count, 
			 votes_sum_positive, votes_sum_negative, image, last_comment_time, comments_count, votes_count_positive,
             votes_count_negative, val_cyber, val_golos, val_power, editor)
		VALUES
			('', '', 'test-user2', 'permlink5np', 'Title 5 np', 'Body 5', '2010-04-27 19:00:00', 't', 3,
			1000, -1000, 'http://imghosting.net/img5.jpg', '2018-04-29 18:00:00', 120, 5, 1, 0, 0, 0, 'html')
	`)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}


	// Тэги
	_, err = dbConn.Do(`INSERT INTO content_tags 
		(content_id, tag) VALUES
		($1, 'tag1'),
        ($2, 'tag2'),
        ($3, 'tag1'),
        ($4, 'tag2'),
        ($5, 'tag1');
	`, npArticle1Id, npArticle2Id, npArticle3Id, npArticle4Id, npArticle5Id)
	if err != nil {
		log.Fatalf("Error when add test data to DB: %s", err)
	}
}
