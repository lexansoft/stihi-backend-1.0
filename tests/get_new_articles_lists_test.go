package tests

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"gitlab.com/stihi/stihi-backend/actions"
	"strings"
	"encoding/json"
	"fmt"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"gitlab.com/stihi/stihi-backend/cache_level2"
	"gitlab.com/stihi/stihi-backend/app/jwt"
)

func TestGetNewArticlesListLast(t *testing.T) {
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_articles_list",
		strings.NewReader(`{"type":"new","count":2}`),
	)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetArticlesList)

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Fatalf("Not 200 response for 2 new articles: %d - %s", rr.Result().StatusCode, rr.Result().Status)
	}

	jsonResponse := rr.Body.String()
	resp := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonResponse), &resp)
	if err != nil {
		t.Fatalf("Error parse JSON response: %s\n%s", err, jsonResponse)
	}

	// Проверяем статус ответа
	if resp["status"] != "ok" {
		t.Fatalf("Response error: %s", resp["error"])
	}

	if resp["list"] == nil {
		t.Fatalf("Response return empty list")
	}

	articles := resp["list"].([]interface{})

	if len(articles) != 2 {
		t.Fatalf("Returned wrong count of articles. Should be 2, returned %d", len(articles))
	}

	row1 := articles[0].(map[string]interface{})
	row2 := articles[1].(map[string]interface{})

	// Проверяем возвращенные данные в строке 1
	if	row1["id"].(float64) != float64(article4Id) ||
		row1["author"].(string) != "test-user4" ||
	   	row1["permlink"].(string) != "permlink4" ||
		row1["title"].(string) != "Title 4" ||
		row1["body"].(string) != "Body 4" ||
		row1["time"].(string) != "2018-04-28 12:00:00 +0000 UTC" ||
		row1["image"].(string) != "http://imghosting.net/img4.jpg" ||
		row1["last_comment_time"].(string) != "2018-04-29 18:00:00 +0000 UTC" ||
		row1["comments_count"].(float64) != 500.0 ||
		row1["votes_count"].(float64) != 15.0 ||
		row1["votes_count_positive"].(float64) != 5.0 ||
		row1["votes_count_negative"].(float64) != 10.0 ||
		row1["votes_sum_positive"].(float64) != 5000.0 ||
		row1["votes_sum_negative"].(float64) != -10000.0 {
			t.Fatalf("Wrong return data in row 1: %+v", row1)
	}

	// Проверяем возвращенные данные в строке 2
	if 	row2["id"].(float64) != float64(article3Id) ||
		row2["author"].(string) != "test-user3" ||
		row2["permlink"].(string) != "permlink3" ||
		row2["title"].(string) != "Title 3" ||
		row2["body"].(string) != "Body 3" ||
		row2["time"].(string) != "2018-04-27 12:00:00 +0000 UTC" ||
		row2["image"].(string) != "http://imghosting.net/img3.jpg" ||
		row2["last_comment_time"].(string) != "2018-04-27 18:00:00 +0000 UTC" ||
		row2["comments_count"].(float64) != 10.0 ||
		row2["votes_count"].(float64) != 5.0 ||
		row2["votes_count_positive"].(float64) != 5.0 ||
		row2["votes_count_negative"].(float64) != 0.0 ||
		row2["votes_sum_positive"].(float64) != 5000.0 ||
		row2["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 2: %+v", row2)
	}
}

func TestGetNewArticlesListAfter(t *testing.T) {
	requestStr := fmt.Sprintf(`{"type":"new","count":2,"after_article":%d}`, article3Id)
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_articles_list",
		strings.NewReader(requestStr),
	)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetArticlesList)

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Fatalf("Not 200 response for 2 new articles: %d - %s", rr.Result().StatusCode, rr.Result().Status)
	}

	jsonResponse := rr.Body.String()
	resp := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonResponse), &resp)
	if err != nil {
		t.Fatalf("Error parse JSON response: %s\n%s", err, jsonResponse)
	}

	// Проверяем статус ответа
	if resp["status"] != "ok" {
		t.Fatalf("Response error: %s", resp["error"])
	}

	if resp["list"] == nil {
		t.Fatalf("Response return empty list")
	}

	articles := resp["list"].([]interface{})

	if len(articles) != 2 {
		t.Fatalf("Returned wrong count of articles. Should be 2, returned %d", len(articles))
	}

	row1 := articles[0].(map[string]interface{})
	row2 := articles[1].(map[string]interface{})

	// Проверяем возвращенные данные в строке 1
	if	row1["id"].(float64) != float64(article2Id) ||
		row1["author"].(string) != "test-user2" ||
		row1["permlink"].(string) != "permlink2" ||
		row1["title"].(string) != "Title 2" ||
		row1["body"].(string) != "Body 2" ||
		row1["time"].(string) != "2018-04-26 16:00:00 +0000 UTC" ||
		row1["image"].(string) != "http://imghosting.net/img2.jpg" ||
		row1["last_comment_time"].(string) != "2018-04-30 12:00:00 +0000 UTC" ||
		row1["comments_count"].(float64) != 100.0 ||
		row1["votes_count"].(float64) != 10.0 ||
		row1["votes_count_positive"].(float64) != 10.0 ||
		row1["votes_count_negative"].(float64) != 0.0 ||
		row1["votes_sum_positive"].(float64) != 10000.0 ||
		row1["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 1: %+v", row1)
	}

	// Проверяем возвращенные данные в строке 2
	if 	row2["id"].(float64) != float64(article1Id) ||
		row2["author"].(string) != "test-user1" ||
		row2["permlink"].(string) != "permlink1" ||
		row2["title"].(string) != "Title 1" ||
		row2["body"].(string) != "Body 1" ||
		row2["time"].(string) != "2018-04-26 15:00:00 +0000 UTC" ||
		row2["image"].(string) != "http://imghosting.net/img1.jpg" ||
		row2["last_comment_time"].(string) != "" ||
		row2["comments_count"].(float64) != 0.0 ||
		row2["votes_count"].(float64) != 0.0 ||
		row2["votes_count_positive"].(float64) != 0.0 ||
		row2["votes_count_negative"].(float64) != 0.0 ||
		row2["votes_sum_positive"].(float64) != 0.0 ||
		row2["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 2: %+v", row2)
	}
}

func TestGetNewArticlesListBefore(t *testing.T) {
	requestStr := fmt.Sprintf(`{"type":"new","before_article":%d}`, article2Id)
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_articles_list",
		strings.NewReader(requestStr),
	)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetArticlesList)

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Fatalf("Not 200 response for 2 new articles: %d - %s", rr.Result().StatusCode, rr.Result().Status)
	}

	jsonResponse := rr.Body.String()
	resp := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonResponse), &resp)
	if err != nil {
		t.Fatalf("Error parse JSON response: %s\n%s", err, jsonResponse)
	}

	// Проверяем статус ответа
	if resp["status"] != "ok" {
		t.Fatalf("Response error: %s", resp["error"])
	}

	if resp["list"] == nil {
		t.Fatalf("Response return empty list")
	}

	articles := resp["list"].([]interface{})

	if len(articles) != 2 {
		t.Fatalf("Returned wrong count of articles. Should be 2, returned %d", len(articles))
	}

	row1 := articles[0].(map[string]interface{})
	row2 := articles[1].(map[string]interface{})

	// Проверяем возвращенные данные в строке 1
	if row1["id"].(float64) != float64(article4Id) ||
		row1["author"].(string) != "test-user4" ||
		row1["permlink"].(string) != "permlink4" ||
		row1["title"].(string) != "Title 4" ||
		row1["body"].(string) != "Body 4" ||
		row1["time"].(string) != "2018-04-28 12:00:00 +0000 UTC" ||
		row1["image"].(string) != "http://imghosting.net/img4.jpg" ||
		row1["last_comment_time"].(string) != "2018-04-29 18:00:00 +0000 UTC" ||
		row1["comments_count"].(float64) != 500.0 ||
		row1["votes_count"].(float64) != 15.0 ||
		row1["votes_count_positive"].(float64) != 5.0 ||
		row1["votes_count_negative"].(float64) != 10.0 ||
		row1["votes_sum_positive"].(float64) != 5000.0 ||
		row1["votes_sum_negative"].(float64) != -10000.0 {
		t.Fatalf("Wrong return data in row 1: %+v", row1)
	}

	// Проверяем возвращенные данные в строке 2
	if row2["id"].(float64) != float64(article3Id) ||
		row2["author"].(string) != "test-user3" ||
		row2["permlink"].(string) != "permlink3" ||
		row2["title"].(string) != "Title 3" ||
		row2["body"].(string) != "Body 3" ||
		row2["time"].(string) != "2018-04-27 12:00:00 +0000 UTC" ||
		row2["image"].(string) != "http://imghosting.net/img3.jpg" ||
		row2["last_comment_time"].(string) != "2018-04-27 18:00:00 +0000 UTC" ||
		row2["comments_count"].(float64) != 10.0 ||
		row2["votes_count"].(float64) != 5.0 ||
		row2["votes_count_positive"].(float64) != 5.0 ||
		row2["votes_count_negative"].(float64) != 0.0 ||
		row2["votes_sum_positive"].(float64) != 5000.0 ||
		row2["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 2: %+v", row2)
	}
}

func TestGetNewArticlesListLastWithTags(t *testing.T) {
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_articles_list",
		strings.NewReader(`{"type":"new","count":2,"tags":["tag11", "tag21"]}`),
	)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetArticlesList)

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Fatalf("Not 200 response for 2 new articles: %d - %s", rr.Result().StatusCode, rr.Result().Status)
	}

	jsonResponse := rr.Body.String()
	resp := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonResponse), &resp)
	if err != nil {
		t.Fatalf("Error parse JSON response: %s\n%s", err, jsonResponse)
	}

	// Проверяем статус ответа
	if resp["status"] != "ok" {
		t.Fatalf("Response error: %s", resp["error"])
	}

	if resp["list"] == nil {
		t.Fatalf("Response return empty list")
	}

	articles := resp["list"].([]interface{})

	if len(articles) != 2 {
		t.Fatalf("Returned wrong count of articles. Should be 2, returned %d", len(articles))
	}

	row1 := articles[0].(map[string]interface{})
	row2 := articles[1].(map[string]interface{})

	// Проверяем возвращенные данные в строке 1
	if 	row1["id"].(float64) != float64(article3Id) ||
		row1["author"].(string) != "test-user3" ||
		row1["permlink"].(string) != "permlink3" ||
		row1["title"].(string) != "Title 3" ||
		row1["body"].(string) != "Body 3" ||
		row1["time"].(string) != "2018-04-27 12:00:00 +0000 UTC" ||
		row1["image"].(string) != "http://imghosting.net/img3.jpg" ||
		row1["last_comment_time"].(string) != "2018-04-27 18:00:00 +0000 UTC" ||
		row1["comments_count"].(float64) != 10.0 ||
		row1["votes_count"].(float64) != 5.0 ||
		row1["votes_count_positive"].(float64) != 5.0 ||
		row1["votes_count_negative"].(float64) != 0.0 ||
		row1["votes_sum_positive"].(float64) != 5000.0 ||
		row1["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 1: %+v", row1)
	}

	// Проверяем возвращенные данные в строке 2
	if	row2["id"].(float64) != float64(article2Id) ||
		row2["author"].(string) != "test-user2" ||
		row2["permlink"].(string) != "permlink2" ||
		row2["title"].(string) != "Title 2" ||
		row2["body"].(string) != "Body 2" ||
		row2["time"].(string) != "2018-04-26 16:00:00 +0000 UTC" ||
		row2["image"].(string) != "http://imghosting.net/img2.jpg" ||
		row2["last_comment_time"].(string) != "2018-04-30 12:00:00 +0000 UTC" ||
		row2["comments_count"].(float64) != 100.0 ||
		row2["votes_count"].(float64) != 10.0 ||
		row2["votes_count_positive"].(float64) != 10.0 ||
		row2["votes_count_negative"].(float64) != 0.0 ||
		row2["votes_sum_positive"].(float64) != 10000.0 ||
		row2["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 2: %+v", row2)
	}
}

func TestGetNewArticlesListLastWithIgnoreTags(t *testing.T) {
	// TODO: Сделать корректное тестирование

	return

	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_articles_list",
		strings.NewReader(`{"type":"new","count":2,"tags":["tag3"]}`),
	)

	userName := "test-user1"
	userId, err := cache_level1.DB.GetUserId(userName)
	if err != nil || userId <= 0 {
		t.Fatalf("Error get user 'test-user1': %s", err)
	}

	token, _, err := jwt.New(userId, userName, "u", "", "")
	if err != nil {
		t.Fatalf("Error generate JWT: %s", err)
	}

	userInfo := cache_level2.UserInfo{}
	userInfo.Id = userId
	userInfo.PvtPostsShowMode = "H"

	err = cache_level1.DB.UpdateUserInfo(&userInfo)
	if err != nil {
		t.Fatalf("Error update userInfo: %s", err)
	}

	req.Header.Add("Authorization", "BEARER "+string(token))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetArticlesList)

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Fatalf("Not 200 response for 2 new articles: %d - %s", rr.Result().StatusCode, rr.Result().Status)
	}

	jsonResponse := rr.Body.String()
	resp := make(map[string]interface{})
	err = json.Unmarshal([]byte(jsonResponse), &resp)
	if err != nil {
		t.Fatalf("Error parse JSON response: %s\n%s", err, jsonResponse)
	}

	// Проверяем статус ответа
	if resp["status"] != "ok" {
		t.Fatalf("Response error: %s", resp["error"])
	}

	if resp["list"] == nil {
		t.Fatalf("Response return empty list")
	}

	articles := resp["list"].([]interface{})

	if len(articles) != 1 {
		fmt.Printf("%+v", articles)
		t.Fatalf("Returned wrong count of articles. Should be 1, returned %d", len(articles))
	}

	row1 := articles[0].(map[string]interface{})

	// Проверяем возвращенные данные в строке 1
	if 	row1["id"].(float64) != float64(article3Id) ||
		row1["author"].(string) != "test-user3" ||
		row1["permlink"].(string) != "permlink3" ||
		row1["title"].(string) != "Title 3" ||
		row1["body"].(string) != "Body 3" ||
		row1["time"].(string) != "2018-04-27 12:00:00 +0000 UTC" ||
		row1["image"].(string) != "http://imghosting.net/img3.jpg" ||
		row1["last_comment_time"].(string) != "2018-04-27 18:00:00 +0000 UTC" ||
		row1["comments_count"].(float64) != 10.0 ||
		row1["votes_count"].(float64) != 5.0 ||
		row1["votes_count_positive"].(float64) != 5.0 ||
		row1["votes_count_negative"].(float64) != 0.0 ||
		row1["votes_sum_positive"].(float64) != 5000.0 ||
		row1["votes_sum_negative"].(float64) != 0.0 {
		t.Fatalf("Wrong return data in row 1: %+v", row1)
	}
}
