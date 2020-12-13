package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)


type Ans struct {
	Error string `json:"error"`
	Response interface{}`json:"response,omitempty"`
}

	func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
			
		case "/user/profile":
			h.handlerProfile(w, r)
			
		case "/user/create":
			h.handlerCreate(w, r)
			
		default:
			w.WriteHeader(http.StatusNotFound)
			ans, err := json.Marshal(&Ans{Error: "unknown method"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			_, err := w.Write(ans)
			if err != nil {
				panic(err)
			}
			return
		}
	}
	
	func (h *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request){

		

		
		
		var qParams url.Values

		if r.Method == http.MethodPost{
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			qParams, err = url.ParseQuery(string(body))
			if err != nil {
				log.Fatal(err)
			}
		} else {
		qParams = r.URL.Query()
		}
		
		Login:=qParams.Get("login")
		
	
	if Login == "" {
		ans := &Ans{
			Error: fmt.Sprintf("%v must me not empty", strings.ToLower("Login")),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(ansErr)
		return
	}
	
	
	LoginForSet := Login
	
	
	params := ProfileParams{
		Login: LoginForSet,
		
	}
	
	res, err := h.Profile(r.Context(), params)
	if err != nil {
		switch err.(type){
		case ApiError: 
			ae := err.(ApiError)
			w.WriteHeader(ae.HTTPStatus)
			ans , err := json.Marshal(&Ans{Error: ae.Err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			ans , err := json.Marshal(&Ans{Error: err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		}
	}

	var answer interface{}

	ans, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("Error encoding json")
		return
	}

	json.Unmarshal(ans, &answer)

	fullAnswer, err := json.Marshal(&Ans{Response: answer})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("Error encoding json")
		return
	}
	w.Write(fullAnswer)
}
	
	func (h *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request){

		
		if r.Header.Get("X-Auth") != "100500" {
			w.WriteHeader(http.StatusForbidden)
			ans, err := json.Marshal(&Ans{Error: "unauthorized"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		}
		

		
		if r.Method != "POST" {
			ans, err := json.Marshal(&Ans{Error: "bad method"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write(ans)
			return
		}
		
		
		var qParams url.Values

		if r.Method == http.MethodPost{
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			qParams, err = url.ParseQuery(string(body))
			if err != nil {
				log.Fatal(err)
			}
		} else {
		qParams = r.URL.Query()
		}
		
		Login:=qParams.Get("login")
		
	
	if Login == "" {
		ans := &Ans{
			Error: fmt.Sprintf("%v must me not empty", strings.ToLower("Login")),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(ansErr)
		return
	}
			if len(Login) < 10 {
			ans := &Ans{
				Error: fmt.Sprintf("login len must be >= 10"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
		}
		
		
		Name:=qParams.Get("full_name")
		
		Status:=qParams.Get("status")
		
				if Status == "" {
					Status = "user"
				}
				
	in := false
	items := []string{"user","moderator","admin"}
	for _, it := range items {
		if it == Status {
			in = true
		}
	}

	if !in {
		ans := &Ans{
			Error: fmt.Sprintf("status must be one of [user, moderator, admin]"),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(ansErr)
		return
	}
	
		Age:=qParams.Get("age")
		
		_, err := strconv.Atoi(Age)
		if err != nil {
			ans := &Ans{
				Error: fmt.Sprintf("age must be int"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
			}
		
			convertedMin, err := strconv.Atoi(Age)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if convertedMin < 0 {
			ans := &Ans{
				Error: fmt.Sprintf("age must be >= 0"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
		}
		
		
			convertedMax, err := strconv.Atoi(Age)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if convertedMax > 128 {
			ans := &Ans{
				Error: fmt.Sprintf("age must be <= 128"),
			}
			
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
		}
		
		
	
	
	LoginForSet := Login
	
	
	NameForSet := Name
	
	
	StatusForSet := Status
	
	 AgeForSet, err := strconv.Atoi(Age)
	if err != nil {
		panic("Can not convert to int")
	}
	
	
	params := CreateParams{
		Login: LoginForSet,
		Name: NameForSet,
		Status: StatusForSet,
		Age: AgeForSet,
		
	}
	
	res, err := h.Create(r.Context(), params)
	if err != nil {
		switch err.(type){
		case ApiError: 
			ae := err.(ApiError)
			w.WriteHeader(ae.HTTPStatus)
			ans , err := json.Marshal(&Ans{Error: ae.Err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			ans , err := json.Marshal(&Ans{Error: err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		}
	}

	var answer interface{}

	ans, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("Error encoding json")
		return
	}

	json.Unmarshal(ans, &answer)

	fullAnswer, err := json.Marshal(&Ans{Response: answer})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("Error encoding json")
		return
	}
	w.Write(fullAnswer)
}
	
	func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
			
		case "/user/create":
			h.handlerCreate(w, r)
			
		default:
			w.WriteHeader(http.StatusNotFound)
			ans, err := json.Marshal(&Ans{Error: "unknown method"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		}
	}
	
	func (h *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request){

		
		if r.Header.Get("X-Auth") != "100500" {
			w.WriteHeader(http.StatusForbidden)
			ans, err := json.Marshal(&Ans{Error: "unauthorized"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		}
		

		
		if r.Method != "POST" {
			ans, err := json.Marshal(&Ans{Error: "bad method"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.WriteHeader(http.StatusNotAcceptable)
			w.Write(ans)
			return
		}
		
		
		var qParams url.Values

		if r.Method == http.MethodPost{
			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			qParams, err = url.ParseQuery(string(body))
			if err != nil {
				log.Fatal(err)
			}
		} else {
		qParams = r.URL.Query()
		}
		
		Username:=qParams.Get("username")
		
	
	if Username == "" {
		ans := &Ans{
			Error: fmt.Sprintf("%v must me not empty", strings.ToLower("Username")),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(ansErr)
		return
	}
			if len(Username) < 3 {
			ans := &Ans{
				Error: fmt.Sprintf("username len must be >= 3"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
		}
		
		
		Name:=qParams.Get("account_name")
		
		Class:=qParams.Get("class")
		
				if Class == "" {
					Class = "warrior"
				}
				
	in := false
	items := []string{"warrior","sorcerer","rouge"}
	for _, it := range items {
		if it == Class {
			in = true
		}
	}

	if !in {
		ans := &Ans{
			Error: fmt.Sprintf("class must be one of [warrior, sorcerer, rouge]"),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(ansErr)
		return
	}
	
		Level:=qParams.Get("level")
		
		_, err := strconv.Atoi(Level)
		if err != nil {
			ans := &Ans{
				Error: fmt.Sprintf("level must be int"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
			}
		
			convertedMin, err := strconv.Atoi(Level)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if convertedMin < 1 {
			ans := &Ans{
				Error: fmt.Sprintf("level must be >= 1"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
		}
		
		
			convertedMax, err := strconv.Atoi(Level)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if convertedMax > 50 {
			ans := &Ans{
				Error: fmt.Sprintf("level must be <= 50"),
			}
			
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ansErr)
			return
		}
		
		
	
	
	UsernameForSet := Username
	
	
	NameForSet := Name
	
	
	ClassForSet := Class
	
	 LevelForSet, err := strconv.Atoi(Level)
	if err != nil {
		panic("Can not convert to int")
	}
	
	
	params := OtherCreateParams{
		Username: UsernameForSet,
		Name: NameForSet,
		Class: ClassForSet,
		Level: LevelForSet,
		
	}
	
	res, err := h.Create(r.Context(), params)
	if err != nil {
		switch err.(type){
		case ApiError: 
			ae := err.(ApiError)
			w.WriteHeader(ae.HTTPStatus)
			ans , err := json.Marshal(&Ans{Error: ae.Err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			ans , err := json.Marshal(&Ans{Error: err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			w.Write(ans)
			return
		}
	}

	var answer interface{}

	ans, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("Error encoding json")
		return
	}

	json.Unmarshal(ans, &answer)

	fullAnswer, err := json.Marshal(&Ans{Response: answer})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("Error encoding json")
		return
	}
	w.Write(fullAnswer)
}
	