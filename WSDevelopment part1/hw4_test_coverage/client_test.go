package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Users struct {
	XMLName xml.Name  `xml:"root"`
	Users   []RowUser `xml:"row"`
}

type RowUser struct {
	ID             int    `xml:"id"`
	Guid           string `xml:"guid"`
	IsActive       bool   `xml:"isActive"`
	Balance        string `xml:"balance"`
	Picture        string `xml:"picture"`
	Age            string `xml:"age"`
	EyeColor       string `xml:"eyeColor"`
	FirstName      string `xml:"first_name"`
	LastName       string `xml:"last_name"`
	Gender         string `xml:"gender"`
	Company        string `xml:"company"`
	Email          string `xml:"email"`
	Phone          string `xml:"phone"`
	Address        string `xml:"address"`
	About          string `xml:"about"`
	Registered     string `xml:"registered"`
	FavouriteFruit string `xml:"favoriteFruit"`
}

func (ru *RowUser) SelectQuery(query string) bool {
	return strings.Contains(ru.About, query) || strings.Contains(ru.FirstName, query) || strings.Contains(ru.LastName, query)
}

func rowToUser(ru RowUser) User {

	u := User{}
	u.Id = ru.ID
	u.Name = ru.FirstName + " " + ru.LastName
	age, err := strconv.Atoi(ru.Age)
	if err != nil {
		fmt.Println("Cant parse age: age will be set 0")
		u.Age = 0
	} else {
		u.Age = age
	}
	u.About = ru.About
	u.Gender = ru.Gender
	return u
}

func selectUsers(users Users, query string) []User {

	usersToAnswer := make([]User, 0)

	if query == "" {
		for _, rowUser := range users.Users {
			usersToAnswer = append(usersToAnswer, rowToUser(rowUser))
		}
	} else {
		for _, rowUser := range users.Users {

			if rowUser.SelectQuery(query) {
				usersToAnswer = append(usersToAnswer, rowToUser(rowUser))
			}
		}
	}

	return usersToAnswer

}

func sortUsers(users []User, orderField string, orderByVal string) ([]User, error) {

	switch orderField {
	case "Id":
		switch orderByVal {
		case "-1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Id > users[j].Id
			})
		case "0":
		case "1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Id < users[j].Id
			})
		default:
			return nil, fmt.Errorf("BadOrderByValue")
		}
	case "Age":
		switch orderByVal {
		case "-1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Age > users[j].Age
			})
		case "0":
		case "1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Age < users[j].Age
			})
		default:
			return nil, fmt.Errorf("BadOrderByValue")
		}
	case "Name":
		switch orderByVal {
		case "-1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Name > users[j].Name
			})
		case "0":
		case "1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Name < users[j].Name
			})
		default:
			return nil, fmt.Errorf("BadOrderByValue")
		}
	case "":
		switch orderByVal {
		case "-1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Name > users[j].Name
			})
		case "0":
		case "1":
			sort.SliceStable(users, func(i, j int) bool {
				return users[i].Name < users[j].Name
			})
		default:
			return nil, fmt.Errorf("BadOrderByValue")
		}

	default:
		return nil, fmt.Errorf(ErrorBadOrderField)
	}
	return users, nil
}

var filePath = "dataset.xml"

func SearchServer(w http.ResponseWriter, r *http.Request) {

	urlParams := r.URL.Query()

	if r.Header.Get("AccessToken") != "1234" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	query := urlParams.Get("query")
	orderField := urlParams.Get("order_field")
	orderByValue := urlParams.Get("order_by")

	offset, err := strconv.Atoi(urlParams.Get("offset"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	xmlFile, err := os.Open(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func(){
		err := xmlFile.Close()
		if err != nil {
			panic(err)
		}
	}()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var users Users

	err = xml.Unmarshal(byteValue, &users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	usersToAnswer := selectUsers(users, query)

	ans, err := sortUsers(usersToAnswer, orderField, orderByValue)

	var errAns *SearchErrorResponse
	if err != nil {
		switch err.Error() {
		case "BadOrderByValue":
			errAns = &SearchErrorResponse{Error: "BadOrderByValue"}
		case ErrorBadOrderField:
			errAns = &SearchErrorResponse{Error: "ErrorBadOrderField"}
		}
		ansJson, err := json.Marshal(errAns)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(ansJson)
		if err != nil {
			panic(err)
		}
		return
	}

	var ansLen int
	if len(ans) < offset {
		ansLen = len(ans)
	} else {
		ansLen = offset
	}
	byteAns, err := json.Marshal(ans[:ansLen])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(byteAns)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type TestCase struct {
	ID      int
	Result  *SearchResponse
	Request *SearchRequest
	isError bool
}

func TestFindUsers(t *testing.T) {
	cases := []TestCase{
		TestCase{
			ID: 1,
			Result: &SearchResponse{
				Users: []User{
					User{
						Id:     27,
						Name:   "Rebekah Sutton",
						Age:    26,
						About:  "Aliqua exercitation ad nostrud et exercitation amet quis cupidatat esse nostrud proident. Ullamco voluptate ex minim consectetur ea cupidatat in mollit reprehenderit voluptate labore sint laboris. Minim cillum et incididunt pariatur amet do esse. Amet irure elit deserunt quis culpa ut deserunt minim proident cupidatat nisi consequat ipsum.\n",
						Gender: "female",
					},
				},
				NextPage: false,
			},
			Request: &SearchRequest{Limit: 25,
				Offset:     1,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)

		if err != nil && !item.isError {
			t.Errorf("Unexpected error: %v", err)
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func TestRequestLimitOffset(t *testing.T) {
	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: -1,
				Offset:     1,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
		TestCase{
			ID:     2,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     -1,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
		TestCase{
			ID:     3,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     10,
				Query:      "Rebekah",
				OrderField: "orknwdfkjawnf",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for _, item := range cases {

		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)

		if err != nil && !item.isError {
			t.Errorf("[%v] Unexpected error", item.ID)
		}

		if err == nil && item.isError {
			t.Errorf("[%v] Expected error, got nil", item.ID)
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", item.ID, resp, item.Result)
		}

	}

}

func TestLargeLimit(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

	RequestLarge := &SearchRequest{Limit: 30,
		Offset:     1,
		Query:      "c",
		OrderField: "Name",
		OrderBy:    1,
	}

	RequestSmall := &SearchRequest{Limit: 25,
		Offset:     1,
		Query:      "c",
		OrderField: "Name",
		OrderBy:    1,
	}

	respLarge, _ := cl.FindUsers(*RequestLarge)

	respSmall, _ := cl.FindUsers(*RequestSmall)

	if !reflect.DeepEqual(respLarge, respSmall) {
		t.Errorf("Not equal answers when large limit set")
	}

}

func TestLargeAnswer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

	RequestLarge := &SearchRequest{Limit: 1,
		Offset:     2,
		Query:      "c",
		OrderField: "Name",
		OrderBy:    1,
	}

	RequestSmall := &SearchRequest{Limit: 1,
		Offset:     1,
		Query:      "c",
		OrderField: "Name",
		OrderBy:    1,
	}

	respLarge, _ := cl.FindUsers(*RequestLarge)

	respSmall, _ := cl.FindUsers(*RequestSmall)

	if !reflect.DeepEqual(respLarge.Users, respSmall.Users) {
		t.Errorf("Not equal answers when large limit set")
	}

}

func TestBadAuth(t *testing.T) {
	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     1,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1235", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)
		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func TestBadRequest(t *testing.T) {

	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     1,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    2,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)
		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func TimeOutSimulate(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 2)
}

func TestTimeout(t *testing.T) {

	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     10,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(TimeOutSimulate))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)
		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func TestBadUrl(t *testing.T) {

	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     10,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(TimeOutSimulate))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ""}

		resp, err := cl.FindUsers(*item.Request)

		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func InternalServerSimulate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func TestInternalServer(t *testing.T) {

	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     10,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(InternalServerSimulate))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)
		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func ReturnBrokenJson(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("brokenJson"))
	if err != nil {
		panic(err)
	}
}

func TestBrokenJson(t *testing.T) {

	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     10,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(ReturnBrokenJson))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)
		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}

func ReturnErrorBrokenJson(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write([]byte("BrokenErrorJson"))
	if err != nil {
		panic(err)
	}
}

func TestBrokenErrorJson(t *testing.T) {

	cases := []TestCase{
		TestCase{
			ID:     1,
			Result: nil,
			Request: &SearchRequest{Limit: 10,
				Offset:     10,
				Query:      "Rebekah",
				OrderField: "Name",
				OrderBy:    1,
			},
			isError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(ReturnErrorBrokenJson))
	defer ts.Close()

	for caseNum, item := range cases {
		cl := &SearchClient{AccessToken: "1234", URL: ts.URL}

		resp, err := cl.FindUsers(*item.Request)
		if err != nil && !item.isError {
			t.Errorf("Unexpected error")
		}

		if err == nil && item.isError {
			t.Errorf("Expected error, got nil")
		}

		if !reflect.DeepEqual(item.Result, resp) {
			t.Errorf("[%v] Wrong result: \n\n %v \n Expected: \n\n %v", caseNum, resp, item.Result)
		}

	}
}
