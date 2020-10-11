package tests

/*
// Рассчет батарейки перенесли в сканер
func TestGetUserBattery(t *testing.T) {
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_user_battery",
		strings.NewReader(`{}`),
	)

	userName := "test-user1"
	userId, err := cache_level1.DB.GetUserId(userName)
	if err != nil || userId <= 0 {
		t.Fatalf("Error get user 'test-user1': %s", err)
	}

	token, _, err := jwt.New(userId, userName, "u", "")
	if err != nil {
		t.Fatalf("Error generate JWT: %s", err)
	}

	req.Header.Add("Authorization", "BEARER "+string(token))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetUserBattery)

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

	if (resp["value"] != "98.53") && (resp["value"] != "98.54") {
		t.Fatalf("Bad value of battery. Expected '98.53' or '98.54', got '%s'", resp["value"])
	}
}

func TestGetUserBatteryNoVotes(t *testing.T) {
	req := httptest.NewRequest(
		"POST",
		"/api/v1/get_user_battery",
		strings.NewReader(`{}`),
	)

	userName := "test-user4"
	userId, err := cache_level1.DB.GetUserId(userName)
	if err != nil || userId <= 0 {
		t.Fatalf("Error get user 'test-user4': %s", err)
	}

	token, _, err := jwt.New(userId, userName, "u", "")
	if err != nil {
		t.Fatalf("Error generate JWT: %s", err)
	}

	req.Header.Add("Authorization", "BEARER "+string(token))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actions.GetUserBattery)

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

	if resp["value"] != "100.00" {
		t.Fatalf("Bad value of battery. Expected '100.00', got '%s'", resp["value"])
	}
}
*/