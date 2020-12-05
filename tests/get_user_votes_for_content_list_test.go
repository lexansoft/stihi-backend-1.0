package tests

import (
	"testing"
	"net/http/httptest"
	"strings"
	"net/http"
	"gitlab.com/stihi/stihi-backend/actions"
	"encoding/json"
	"gitlab.com/stihi/stihi-backend/cache_level1"
	"gitlab.com/stihi/stihi-backend/app/jwt"
	"reflect"
	"strconv"
	"gitlab.com/stihi/stihi-backend/cache_level2"
)

func TestGetUserVotesForContentList(t *testing.T) {
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_articles_list",
		strings.NewReader(`{"type":"new","count":10}`),
	)

	userName := "test-user1"
	userId, err := cache_level1.DB.GetUserId(userName)
	if err != nil || userId <= 0 {
		t.Fatalf("Error get user 'test-user1': %s", err)
	}

	userInfo := cache_level2.UserInfo{}
	userInfo.Id = userId
	userInfo.PvtPostsShowMode = ""

	err = cache_level1.DB.UpdateUserInfo(&userInfo)
	if err != nil {
		t.Fatalf("Error update userInfo: %s", err)
	}

	token, _, err := jwt.New(userId, userName, "u", "", "")
	if err != nil {
		t.Fatalf("Error generate JWT: %s", err)
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

	currentUserVotesI, ok := resp["current_user_votes"]
	if !ok {
		t.Error("Absent field 'current_user_votes' in response")
		return
	}

	currentUserVotes := currentUserVotesI.(map[string]interface{})

	if len(reflect.ValueOf(currentUserVotes).MapKeys()) != 3 {
		t.Errorf("Data 'current_user_votes' is bad size. Expected %d, got %d", 3, len(reflect.ValueOf(currentUserVotes).MapKeys()))
		return
	}

	if int64(currentUserVotes[strconv.FormatInt(article1Id, 10)].(float64)) != 10000 {
		t.Errorf("Wrong vote data for conten id: %d", article1Id)
	}
	if int64(currentUserVotes[strconv.FormatInt(article3Id, 10)].(float64)) != 10000 {
		t.Errorf("Wrong vote data for conten id: %d", article3Id)
	}
	if int64(currentUserVotes[strconv.FormatInt(article4Id, 10)].(float64)) != -10000 {
		t.Errorf("Wrong vote data for conten id: %d", article4Id)
	}
}
