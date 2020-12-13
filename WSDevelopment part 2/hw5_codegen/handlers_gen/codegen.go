// go build gen/* && ./codegen.exe pack/unpack.go  pack/marshaller.go
// go run pack/*
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"os"
	"reflect"
	"strings"
)

type MethodMeta struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

type GenMethod struct {
	MethodName string
	StructName string
	Meta       MethodMeta
	Params     []*MethodParam
}

type MethodParam struct {
	ParamName string
	ParamType string
}

type GenStruct struct {
	Name   string
	Fields []*Field
}

type Field struct {
	Name      string
	Type      string
	AVComment string
}

type GenObj struct {
	ParentStructName string
	Methods          []*GenMethod
}

func (f *Field) genRequiredValidation(file *os.File) {

	reqTemplateInt := template.Must(template.New("reqTplString").Parse(`
	if {{.Name}} == "0" {
		ans := &Ans{
			Error: fmt.Sprintf("%v must me not empty", {{.Name}}),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(ansErr)
		if err != nil {
			panic(err)
		}
		return
	}`))

	reqTemplateString := template.Must(template.New("reqTplInt").Parse(`
	
	if {{.Name}} == "" {
		ans := &Ans{
			Error: fmt.Sprintf("%v must me not empty", strings.ToLower("{{.Name}}")),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(ansErr)
		if err != nil {
			panic(err)
		}
		return
	}`))

	switch f.Type {
	case "int":
		err := reqTemplateInt.Execute(file, f)
		if err != nil {
			panic(err)
		}
	case "string":
		err := reqTemplateString.Execute(file, f)
		if err != nil {
			panic(err)
		}
	}
}

func (f *Field) genParamnameValidation(file *os.File, paramName string) {

	if paramName == "" {
		_, err := file.WriteString(fmt.Sprintf(`
		%v:=qParams.Get("%v")
		`, f.Name, strings.ToLower(f.Name)))
		if err != nil {
			panic(err)
		}
	} else {
		_, err := file.WriteString(fmt.Sprintf(`
		%v:=qParams.Get("%v")
		`, f.Name, paramName))
		if err != nil {
			panic(err)
		}
	}

	if f.Type == "int" {
		_, err := file.WriteString(fmt.Sprintf(`
		_, err := strconv.Atoi(%v)
		if err != nil {
			ans := &Ans{
				Error: fmt.Sprintf("%v must be int"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write(ansErr)
			if err != nil {
				panic(err)
			}
			return
			}
		`, f.Name, strings.ToLower(f.Name)))
		if err != nil {
			panic(err)
		}
	}
}

func (f *Field) genEnumValidation(file *os.File, items []string) {

	itemsForErr := strings.Join(items, ", ")
	for i, item := range items {
		items[i] = "\"" + item + "\""
	}

	itemsString := strings.Join(items, ",")

	_, err := file.WriteString(fmt.Sprintf(`
	in := false
	items := []string{%v}
	for _, it := range items {
		if it == %v {
			in = true
		}
	}

	if !in {
		ans := &Ans{
			Error: fmt.Sprintf("%v must be one of [%v]"),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(ansErr)
		if err != nil {
			panic(err)
		}
		return
	}
	`, itemsString, f.Name, strings.ToLower(f.Name), itemsForErr))
	if err != nil {
		panic(err)
	}
}

func (f *Field) genDefaultValidation(file *os.File, defaultValue interface{}) {

	switch defaultValue.(type) {
	case int:
		_, err := file.WriteString(fmt.Sprintf(`
			if %v == 0 {
				%v = %v
			}
			`, f.Name, f.Name, defaultValue.(int)))
		if err != nil{
			panic(err)
		}
	case string:
		_, err := file.WriteString(fmt.Sprintf(`
				if %v == "" {
					%v = "%v"
				}
				`, f.Name, f.Name, defaultValue.(string)))
		if err != nil {
			panic(err)
		}
	}
}

func (f *Field) genMinValidation(file *os.File, val string) {
	switch f.Type {
	case "int":
		_, err := file.WriteString(fmt.Sprintf(`
			convertedMin, err := strconv.Atoi(%v)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if convertedMin < %v {
			ans := &Ans{
				Error: fmt.Sprintf("%v must be >= %v"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write(ansErr)
			if err != nil {
				panic(err)
			}
			return
		}
		`, f.Name, val, strings.ToLower(f.Name), val))
		if err != nil {
			panic(err)
		}
	case "string":
		_, err := file.WriteString(fmt.Sprintf(`
			if len(%v) < %v {
			ans := &Ans{
				Error: fmt.Sprintf("%v len must be >= %v"),
			}
			ansErr, err := json.Marshal(ans)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Printf("Can not marshall error json")
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write(ansErr)
			if err != nil {
				panic(err)
			}
			return
		}
		`, f.Name, val, strings.ToLower(f.Name), val))
		if err != nil {
			panic(err)
		}
	}
}

func (f *Field) genMaxValidation(file *os.File, val string) {
	switch f.Type {
	case "int":
		_, err := file.WriteString(fmt.Sprintf(`
		convertedMax, err := strconv.Atoi(%v)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if convertedMax > %v {
		ans := &Ans{
			Error: fmt.Sprintf("%v must be <= %v"),
		}
		
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(ansErr)
		if err != nil {
			panic(err)
		}
		return
		}
		
		`, f.Name, val, strings.ToLower(f.Name), val))
		if err != nil {
			panic(err)
		}
	case "string":
		_, err := file.WriteString(fmt.Sprintf(`
		if len(%v) > %v {
		ans := &Ans{
			Error: fmt.Sprintf("%v len must be <= %v"),
		}
		ansErr, err := json.Marshal(ans)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Can not marshall error json")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write(ansErr)
		if err != nil {
			panic(err)
		}
		return
		}
		`, f.Name, val, strings.ToLower(f.Name), val))
		if err != nil {
			panic(err)
		}
	}
}

func (f *Field) genParamsValidation(file *os.File) {

	validMarks := strings.Split(f.AVComment, ",")

	act := make(map[string]interface{})

	for _, tag := range validMarks {
		if tag == "required" {
			act["required"] = true
			continue
		}
		act[strings.Split(tag, "=")[0]] = strings.Split(tag, "=")[1]
	}

	pn, ok := act["paramname"]
	if !ok {
		f.genParamnameValidation(file, "")
	} else {
		f.genParamnameValidation(file, pn.(string))
	}

	_, ok = act["required"]
	if ok {
		f.genRequiredValidation(file)
	}

	df, ok := act["default"]
	if ok {
		f.genDefaultValidation(file, df)
	}

	enum, ok := act["enum"]
	if ok {
		f.genEnumValidation(file, strings.Split(enum.(string), "|"))
	}

	min, ok := act["min"]
	if ok {
		f.genMinValidation(file, min.(string))
	}

	max, ok := act["max"]
	if ok {
		f.genMaxValidation(file, max.(string))
	}
}

// ParseStructure searches for structures in the source code which fields contains tag `apivalidator`.
func ParseStructure(currType *ast.TypeSpec) (*GenStruct, bool) {

	var vs *GenStruct

	currStruct, ok := currType.Type.(*ast.StructType)
	if !ok {
		//fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
		return nil, false
	}

	strName := currType.Name.Name
	//fmt.Printf("Get structure %v\n", strName)
	fs := make([]*Field, 0)

	if len(currStruct.Fields.List) == 0 {
		return nil, false
	}

	for _, fd := range currStruct.Fields.List {
		fieldName := fd.Names[0].Name
		fieldT, ok := fd.Type.(*ast.Ident)
		if !ok {
			return nil, false
		}
		fieldType := fieldT.Name

		//fmt.Printf("Get field %v of struct %v of type %v\n", fieldName, strName, fieldType)
		if fieldType != "int" && fieldType != "string" {
			fmt.Printf("unsupported %v\n", fieldType)
			return nil, false
		}

		var strTag string
		if fd.Tag != nil {
			tag := reflect.StructTag(fd.Tag.Value[1 : len(fd.Tag.Value)-1])
			if tag.Get("apivalidator") == "" {
				return nil, false
			}
			strTag = tag.Get("apivalidator")
		}
		fs = append(fs, &Field{
			Name:      fieldName,
			Type:      fieldType,
			AVComment: strTag,
		})
	}
	vs = &GenStruct{
		Name:   strName,
		Fields: fs,
	}
	return vs, true
}

// ParseMethod find methods with comment having prefix `apigen:api`. Returns methods for codegen.
func ParseMethod(mg *ast.FuncDecl) (*GenMethod, bool) {

	if mg.Recv == nil {
		//fmt.Printf("Skip %v, this is function\n", mg.Name.Name)
		return nil, false
	}

	if mg.Doc == nil {
		//fmt.Printf("Skip method %v: no any prefix\n", mg.Name.Name)
		return nil, false
	}

	var mm MethodMeta

	validCommentExist := false

	CommentLoop:
	for _, comment := range mg.Doc.List {
		if strings.HasPrefix(comment.Text, "// apigen:api ") {
			methodApiInfo := strings.Replace(comment.Text, "// apigen:api ", "", 1)
			err := json.Unmarshal([]byte(methodApiInfo), &mm)
			if err != nil {
				fmt.Printf("Error occured unmarshalling comment's JSON: %v\n", methodApiInfo)
				continue CommentLoop
			}
			validCommentExist = true
			break CommentLoop
		}
	}

	var methodToGen *GenMethod

	if !validCommentExist {
		fmt.Printf("No valid comment on method %v\n", mg.Name.Name)
		return nil, false
	}

	structNameExpr, ok := mg.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return nil, false
	}
	structNameIdent, ok := structNameExpr.X.(*ast.Ident)
	if !ok {
		return nil, false
	}

	structName := structNameIdent.Name

	params := mg.Type.Params.List
	paramVals := make([]*MethodParam, 0)

ParamLoop:
	for paramNum, p := range params {

		var paramType string
		paramName := p.Names[0].Name

		switch p.Type.(type) {
		case *ast.Ident:
			pt := p.Type.(*ast.Ident)
			paramType = pt.Name
		case *ast.SelectorExpr:
			pt := p.Type.(*ast.SelectorExpr)
			paramType = pt.Sel.Name
		default:
			fmt.Printf("Can not convert to *ast.Ident or *ast.SelectorExpr from %v of type %T\n", p.Type, p.Type)
		}

		if paramNum == 0 {
			if paramType == "Context" {
				continue ParamLoop
			}
		}

		paramVals = append(paramVals, &MethodParam{
			ParamName: paramName,
			ParamType: paramType,
		})
	}
	methodToGen = &GenMethod{
		MethodName: mg.Name.Name,
		StructName: structName,
		Meta:       mm,
		Params:     paramVals,
	}

	j, _ := json.Marshal(methodToGen)
	fmt.Printf("parsed method: \n\n%v\n", string(j))
	return methodToGen, true
}

var (
	serveTmp = template.Must(template.New("serve").Parse(`
	func (h *{{.ParentStructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
			{{range .Methods}}
		case "{{.Meta.Url}}":
			h.handler{{.MethodName}}(w, r)
			{{end}}
		default:
			w.WriteHeader(http.StatusNotFound)
			ans, err := json.Marshal(&Ans{Error: "unknown method"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			_, err = w.Write(ans)
			if err != nil {
				panic(err)
			}
			return
		}
	}
	`))

	wrapperHead = template.Must(template.New("wrTmp").Parse(`
	func (h *{{.StructName}}) handler{{.MethodName}}(w http.ResponseWriter, r *http.Request){

		{{if .Meta.Auth}}
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
		{{end}}

		{{if ne .Meta.Method ""}}
		if r.Method != "{{.Meta.Method}}" {
			ans, err := json.Marshal(&Ans{Error: "bad method"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.WriteHeader(http.StatusNotAcceptable)
			_, err = w.Write(ans)
			if err != nil {
				panic(err)
			}
			return
		}
		{{end}}
		
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
		`))

	stuctFiller = template.Must(template.New("fillTmp").Parse(`
	{{range .Fields}}
	{{ if eq .Type "int"}} {{.Name}}ForSet, err := strconv.Atoi({{.Name}})
	if err != nil {
		panic("Can not convert to int")
	}{{else}}
	{{.Name}}ForSet := {{.Name}}{{end}}
	{{ end }}
	
	params := {{.Name}}{
		{{range .Fields}}{{.Name}}: {{.Name}}ForSet,
		{{end}}
	}
	`))

	wrapperTail = template.Must(template.New("wrTailTmp").Parse(`
	res, err := h.{{.MethodName}}(r.Context(), params)
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
			_, err = w.Write(ans)
			if err != nil {
				panic(err)
			}
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			ans , err := json.Marshal(&Ans{Error: err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Fatal("Error encoding json")
				return
			}
			_, err = w.Write(ans)
			if err != nil {
				panic(err)
			}
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
	_, err = w.Write(fullAnswer)
	if err != nil {
		panic(err)
	}
}
	`))
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	methods := make([]*GenMethod, 0)
	structs := make(map[string]*GenStruct)

	for _, f := range node.Decls {

		switch f.(type) {
		case *ast.GenDecl:
			gd, _ := f.(*ast.GenDecl)

		SpecsLoop:
			for _, spec := range gd.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					//fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
					continue SpecsLoop
				}

				s, ok := ParseStructure(currType)
				if !ok {
					//fmt.Print("Error convert to GenStruct\n")
					continue SpecsLoop
				}
				structs[s.Name] = s
			}
		case *ast.FuncDecl:
			mg, _ := f.(*ast.FuncDecl)

			m, ok := ParseMethod(mg)
			if !ok {
				fmt.Printf("Error convert %v to GenMethod\n", mg.Name.Name)
				continue
			}
			methods = append(methods, m)
		}
	}

	mainStructNames := make(map[string]*struct{})

	var emptyStruct struct{}
	for _, elem := range methods {
		mainStructNames[elem.StructName] = &emptyStruct
	}

	mainStructs := make([]*GenObj, 0)

	for mainStructName := range mainStructNames {
		strMethods := make([]*GenMethod, 0)
		for _, method := range methods {
			if method.StructName == mainStructName {
				strMethods = append(strMethods, method)
			}
		}
		mainStructs = append(mainStructs, &GenObj{
			ParentStructName: mainStructName,
			Methods:          strMethods,
		})
	}

	// Начинаем генерацию кода

	out, _ := os.Create(os.Args[2])

	_, err = fmt.Fprintln(out, `package `+node.Name.Name)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprintln(out) // empty line
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprintln(out, `import (
							"net/http"
							"net/url"
							"strconv"
							"encoding/json"
							"fmt"
							"io/ioutil"
							"log"
							"strings"
							)`)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprintln(out) // empty line
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprintln(out, `type Ans struct {
		Error string`+" `json:\"error\"`"+"\n"+
		`Response interface{}`+"`json:\"response,omitempty\"`"+"\n}")
	if err != nil {
		panic(err)
	}

	for _, ms := range mainStructs {

		err := serveTmp.Execute(out, ms)
		if err != nil {
			panic(err)
		}
	MethodLoop:
		for _, method := range ms.Methods {
			err := wrapperHead.Execute(out, method)
			if err != nil {
				panic(err)
			}

			for _, param := range method.Params {
				strToGen, ok := structs[param.ParamType]
				if !ok {
					fmt.Printf("Can not find struct type %v for method %v", param.ParamType, method.MethodName)
					continue MethodLoop
				}
				for _, f := range strToGen.Fields {
					f.genParamsValidation(out)
				}
				err := stuctFiller.Execute(out, strToGen)
				if err != nil {
					panic(err)
				}
			}
			err = wrapperTail.Execute(out, method)
			if err != nil {
				panic(err)
			}
		}
	}

}

// go build gen/* && ./codegen.exe pack/unpack.go  pack/marshaller.go
// go run pack/*
