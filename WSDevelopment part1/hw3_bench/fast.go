package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

//easyjson:json
type userProfile struct {
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
}

func easyjsonB4ad3f7dDecodeHw3Bench(in *jlexer.Lexer, out *userProfile) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "email":
			out.Email = string(in.String())
		case "name":
			out.Name = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonB4ad3f7dEncodeHw3Bench(out *jwriter.Writer, in userProfile) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"browsers\":"
		out.RawString(prefix[1:])
		if in.Browsers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Browsers {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"email\":"
		out.RawString(prefix)
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"name\":"
		out.RawString(prefix)
		out.String(string(in.Name))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v userProfile) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonB4ad3f7dEncodeHw3Bench(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v userProfile) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonB4ad3f7dEncodeHw3Bench(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *userProfile) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonB4ad3f7dDecodeHw3Bench(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *userProfile) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonB4ad3f7dDecodeHw3Bench(l, v)
}

func FastSearch(out io.Writer) {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileScanner := bufio.NewScanner(file)

	seenBrowsers := make([]string, 0, 100)
	uniqueBrowsers := 0

	var i int

	fmt.Println(out, "found users:")

	user := &userProfile{}

	for {

		ok := fileScanner.Scan()
		if !ok {
			break
		}

		isAndroid := false
		isMSIE := false

		err := user.UnmarshalJSON(fileScanner.Bytes())
		if err != nil {
			panic(err)
		}

		for _, browser := range user.Browsers {

			notSeenBefore := true

			switch {

			case strings.Contains(browser, "Android"):
				isAndroid = true
			case strings.Contains(browser, "MSIE"):
				isMSIE = true
			default:
				continue
			}

			for _, item := range seenBrowsers {
				if item == browser {
					notSeenBefore = false
					break
				}
			}

			if notSeenBefore {
				// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
				seenBrowsers = append(seenBrowsers, browser)
				uniqueBrowsers++
			}

		}

		if isAndroid && isMSIE {
			email := strings.ReplaceAll(user.Email, "@", " [at] ")
			_, err := fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, email)
			if err != nil {
				panic(err)
			}
		}
		i++
	}
	_, err = fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
	if err != nil {
		panic(err)
	}
}
