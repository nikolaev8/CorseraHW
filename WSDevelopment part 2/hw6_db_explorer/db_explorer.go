package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Handler struct {
	Database *sql.DB
	Schema   map[string][]ColumnInfo
}

func NewHandler(db *sql.DB, schema map[string][]ColumnInfo) *Handler {
	return &Handler{
		Database: db,
		Schema:   schema,
	}
}

//Response тип для ответа
type Response map[string]interface{}

//SendResponse запаковывает ответ в JSON и отправляет
func (ans *Response) SendResponse(w http.ResponseWriter, status int) {
	ansString, err := json.Marshal(ans)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error marshal answer json")
		return
	}
	w.WriteHeader(status)
	_, err = w.Write(ansString)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("Can not write to ResponseWriter")
		return
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("Get URL path:", r.URL.Path)

	if r.URL.Path == "/" {
		tables := getDatabaseTableNames(h.Database)
		ans := Response{"response": Response{"tables": tables}}
		ans.SendResponse(w, http.StatusOK)
		return
	}

	tableName := strings.Split(r.URL.Path, "/")[1]
	_, ok := h.Schema[tableName]
	if !ok {
		ans := Response{"error": "unknown table"}
		ans.SendResponse(w, http.StatusNotFound)
		log.Println("Get unknown table name:", tableName)
		return
	}

	handler := &TableHandler{
		Database:     h.Database,
		TableName:    tableName,
		TableColumns: h.Schema[tableName],
	}

	if r.URL.Path == "/"+tableName && r.Method == http.MethodGet {
		handler.SelectQuery(w, r)
	}

	if (r.URL.Path == "/"+tableName+"/" || r.URL.Path == "/"+tableName) && r.Method == http.MethodPut {
		handler.CreateRow(w, r)
	}

	getRowPattern := regexp.MustCompile("/" + tableName + "/" + `(\d+)`)
	if getRowPattern.MatchString(r.URL.Path) && r.Method == http.MethodGet {
		handler.GetRow(w, r)
	}

	if getRowPattern.MatchString(r.URL.Path) && r.Method == http.MethodPost {
		handler.UpdateRow(w, r)
	}

	if getRowPattern.MatchString(r.URL.Path) && r.Method == http.MethodDelete {
		handler.DeleteRow(w, r)
	}

	return
}

type TableHandler struct {
	Database     *sql.DB
	TableName    string
	TableColumns []ColumnInfo
}

func (t *TableHandler) getIDColumnName() string {
	for _, c := range t.TableColumns {
		if c.IsID() {
			return c.Field
		}
	}
	return ""
}

// selectRowValidation получает на вход считанную строку таблицы неизвестной длины и с неизвестными
// типами и преобразует ее в мапу пустых интерфейсов с верными типами данных
func (t *TableHandler) selectRowValidation(row []interface{}) (map[string]interface{}, error) {

	log.Println("----------SELECT VALIDATION START----------")
	defer log.Println("----------SELECT VALIDATION END----------")
	log.Println("Got row:", row)

	if len(row) != len(t.TableColumns) {
		log.Println("Got row of wrong legth != table columns")
		return nil, fmt.Errorf("Validation failed")
	}

	rowMap := make(map[string]interface{})

	for idx, column := range t.TableColumns {
		colValue := row[idx]

		if colValue == nil {
			log.Printf("Element %v of type %T\n", colValue, colValue)
			rowMap[column.Field] = nil
			continue
		}

		switch {
		case strings.Contains(column.Type, "int"):
			switch colValue.(type) {
			case sql.NullInt32:
				log.Printf("Element %v of type %T converted to nil\n", colValue, colValue)
				rowMap[column.Field] = nil
			case sql.NullFloat64:
				log.Printf("Element %v of type %T converted to nil\n", colValue, colValue)
				rowMap[column.Field] = nil
			case int64:
				rowMap[column.Field] = colValue.(int64)
				log.Printf("Element %v of type %T converted to int64: %v\n", colValue, colValue, colValue.(int64))
			case int32:
				rowMap[column.Field] = colValue.(int32)
				log.Printf("Element %v of type %T converted to int32: %v\n", colValue, colValue, colValue.(int32))
			default:
				log.Printf("Validation failed: param %v of type %T, expected int\n", colValue, colValue)
				return nil, fmt.Errorf("Validation failed: param %v of type %T, expected int", colValue, colValue)
			}
		case strings.Contains(column.Type, "float"):
			switch colValue.(type) {
			case sql.NullFloat64:
				rowMap[column.Field] = nil
				log.Printf("Element %v of type %T converted to nil\n", colValue, colValue)
			case float32:
				rowMap[column.Field] = colValue.(float32)
				log.Printf("Element %v of type %T converted to float32: %v\n", colValue, colValue, colValue.(float32))
			case float64:
				rowMap[column.Field] = colValue.(float64)
				log.Printf("Element %v of type %T converted to float64: %v\n", colValue, colValue, colValue.(float64))
			default:
				log.Printf("Validation failed: param %v of type %T, expected float\n", colValue, colValue)
				return nil, fmt.Errorf("Validation failed: param %v of type %T, expected float", colValue, colValue)
			}
		default:
			switch colValue.(type) {
			case sql.NullString:
				rowMap[column.Field] = nil
				log.Printf("Element %v of type %T converted to nil\n", colValue, colValue)
			case string:
				rowMap[column.Field] = colValue.(string)
				log.Printf("Element %v of type %T converted to string: %v\n", colValue, colValue, colValue.(string))
			case []byte:
				rowMap[column.Field] = string(colValue.([]byte))
				log.Printf("Element %v of type %T converted to []byte and then to string: %v\n", colValue, colValue, string(colValue.([]byte)))
			default:
				log.Printf("Validation failed: param %v of type %T, string/ []byte\n", colValue, colValue)
				return nil, fmt.Errorf("Validation failed: param %v of type %T, expected string/ []byte", colValue, colValue)
			}
		}
	}
	return rowMap, nil
}

func (t *TableHandler) SelectQuery(w http.ResponseWriter, r *http.Request) {
	log.Println("----------STARTING SELECT----------")
	defer log.Println("----------END SELECT----------")
	params := r.URL.Query()
	l := params.Get("limit")

	limit, err := strconv.Atoi(l)
	if err != nil || l == "" {
		limit = 5
		log.Println("Got bad limit, set default value 5")
	}
	off := params.Get("offset")

	offset, err := strconv.Atoi(off)
	if err != nil || off == "" {
		offset = 0
		log.Println("Got bad offset, set default value 0")
	}

	colNames := make([]string, len(t.TableColumns))
	for i, c := range t.TableColumns {
		colNames[i] = c.Field
	}

	queryString := fmt.Sprintf("SELECT %s FROM %s", strings.Join(colNames, ", "), t.TableName)
	stmnt, err := t.Database.Prepare(queryString)
	log.Println("Execute querty: ", queryString)

	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		log.Println("Error prepare query:", err)
		return
	}
	defer func() {
		err := stmnt.Close()
		if err != nil {
			panic(err)
		}
	}()

	table, err := stmnt.Query()
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		log.Println("Error execute query:", err)
		return
	}

	defer func(){
		err := table.Close()
		panic(err)
	}()

	var rows []map[string]interface{}

	for table.Next() {
		row := make([]interface{}, len(t.TableColumns))
		rowPointer := make([]interface{}, len(t.TableColumns))

		for idx := range row {
			rowPointer[idx] = &row[idx]
		}

		err = table.Scan(rowPointer...)
		if err != nil {
			resp := Response{"error": err}
			resp.SendResponse(w, http.StatusInternalServerError)
			return
		}

		rowMap, err := t.selectRowValidation(row)
		if err != nil {
			resp := Response{"error": err}
			resp.SendResponse(w, http.StatusInternalServerError)
			return
		}
		rows = append(rows, rowMap)
	}

	if len(rows) < offset {
		resp := Response{"error": "too few records for this offset"}
		resp.SendResponse(w, http.StatusBadRequest)
		log.Println("Got too large offset")
		return
	}

	if len(rows) < limit+offset {
		limit = len(rows) - offset
	}

	fullAns := Response{"response": Response{"records": rows[offset : limit+offset]}}
	fullAns.SendResponse(w, http.StatusOK)
	log.Println("Successfully sent select answer")
	return
}

func (t *TableHandler) GetRow(w http.ResponseWriter, r *http.Request) {
	idCol := t.getIDColumnName()
	if idCol == "" {
		resp := Response{"error": "can not find table id column for get row in id"}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fmt.Println("Id column name: " + idCol + " for table " + t.TableName)
	id := strings.Split(r.URL.Path, "/")[2]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		resp := Response{"error": "bad value for id"}
		resp.SendResponse(w, http.StatusBadRequest)
		return
	}

	row := make([]interface{}, len(t.TableColumns))
	rowPointer := make([]interface{}, len(t.TableColumns))

	for idx := range row {
		rowPointer[idx] = &row[idx]
	}

	colNames := make([]string, len(t.TableColumns))
	for i, c := range t.TableColumns {
		colNames[i] = c.Field
	}

	stmnt, err := t.Database.Prepare(fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", strings.Join(colNames, ", "), t.TableName, idCol))
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	defer func() {
		err := stmnt.Close()
		if err != nil {
			panic(err)
		}
	}()

	st := stmnt.QueryRow(idInt)

	err = st.Scan(rowPointer...)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			fullAns := Response{"error": "record not found"}
			fullAns.SendResponse(w, http.StatusNotFound)
			return
		default:
			fullAns := Response{"error": err}
			fullAns.SendResponse(w, http.StatusInternalServerError)
			return
		}
	}

	rowMap, err := t.selectRowValidation(row)
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fullAns := Response{"response": Response{"record": rowMap}}
	fullAns.SendResponse(w, http.StatusOK)
	return
}

// ParseRequestParams вытаскивает значения колонок из тела запроса
func (t *TableHandler) ParseRequestParams(r *http.Request) (map[string]interface{}, error) {

	bd, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("can not read request body")
		return nil, fmt.Errorf("Can not read request body for parsing params")
	}
	defer func() {
		err := r.Body.Close()
		if err != nil {
			panic(err)
		}
	}()

	rowValues := make(map[string]interface{})

	err = json.Unmarshal(bd, &rowValues)
	if err != nil {
		return nil, err
	}

	if r.Method != http.MethodPost {
		delete(rowValues, t.getIDColumnName())
	}

	return rowValues, nil
}

func (t *TableHandler) getColExistChecker() map[string]*ColumnInfo {
	checker := make(map[string]*ColumnInfo)
	for idx, col := range t.TableColumns {
		checker[col.Field] = &t.TableColumns[idx]
	}
	return checker
}

func (t *TableHandler) validateParams(params map[string]interface{}) (map[string]interface{}, error) {

	vals := make(map[string]interface{})
	checker := t.getColExistChecker()

	for colName, colValue := range params {
		col, ok := checker[colName]
		if !ok {
			continue
		}

		log.Printf("Check column %v/%v: exptected type %v, got value %v of type %T\n", colName, col.Field, col.Type, colValue, colValue)
		val, err := col.Validate(colValue)
		if err != nil {
			return nil, fmt.Errorf("field %v have invalid type", colName)
		}
		vals[colName] = val
	}
	return vals, nil
}

func (t *TableHandler) completeParamsForInsert(params map[string]interface{}) map[string]interface{} {
	for _, column := range t.TableColumns {
		_, ok := params[column.Field]
		if !ok {
			switch {
			case strings.Contains(column.Type, "int"):
				log.Println("Column", column.Field, "NULL: ", column.Null)

				if column.Null == "YES" {
					params[column.Field] = sql.NullInt64{}
					continue
				}
				params[column.Field] = 0
			case strings.Contains(column.Type, "float"):
				log.Println("Column", column.Field, "NULL: ", column.Null)

				if column.Null == "YES" {
					params[column.Field] = sql.NullFloat64{}
					continue
				}
				params[column.Field] = float64(0)
			default:
				if column.Null == "YES" {
					params[column.Field] = sql.NullString{}
					continue
				}
				log.Println("Column", column.Field, "NULL: ", column.Null)
				params[column.Field] = ""
			}

		}
	}
	return params
}

//PUT /$table - создаёт новую запись, данный по записи в теле запроса (POST-параметры)
func (t *TableHandler) CreateRow(w http.ResponseWriter, r *http.Request) {

	params, err := t.ParseRequestParams(r)
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusBadRequest)
		return
	}

	allParams := t.completeParamsForInsert(params)

	validParams, err := t.validateParams(allParams)
	if err != nil {
		resp := Response{"error": "invalid params: " + err.Error()}
		resp.SendResponse(w, http.StatusBadRequest)
		return
	}

	vals := make([]interface{}, len(validParams))
	for idx, v := range t.TableColumns {
		vals[idx] = validParams[v.Field]
	}

	valsPattern := make([]string, len(t.TableColumns))
	for id := range valsPattern {
		valsPattern[id] = "?"
	}
	valString := "(" + strings.Join(valsPattern, ", ") + ")"

	result, err := t.Database.Exec(fmt.Sprintf("INSERT INTO %v VALUES %s ", t.TableName, valString), vals...)
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	insCnt, err := result.RowsAffected()
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	insertId, err := result.LastInsertId()
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fmt.Println("Insert ", insCnt, " rows: ")
	fmt.Println("Insert ", insertId, " rowId")
	fmt.Println(vals)

	fullAns := Response{"response": Response{t.getIDColumnName(): insertId}}
	fullAns.SendResponse(w, http.StatusOK)
	return
}

//POST /$table/$id - обновляет запись, данные приходят в теле запроса (POST-параметры)
func (t *TableHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {

	idCol := t.getIDColumnName()
	if idCol == "" {
		resp := Response{"error": "cat not find id column"}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fmt.Println("Id column name: " + idCol + " for table " + t.TableName)
	id := strings.Split(r.URL.Path, "/")[2]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Println("Get bad id value")
		resp := Response{"error": "bad id value"}
		resp.SendResponse(w, http.StatusBadRequest)
		return
	}

	paramsToUpdate, err := t.ParseRequestParams(r)
	if err != nil {
		resp := Response{"error": err}
		resp.SendResponse(w, http.StatusBadRequest)
		return
	}

	if _, ok := paramsToUpdate[idCol]; ok {
		fullAns := Response{"error": fmt.Sprintf("field %s have invalid type", idCol)}
		fullAns.SendResponse(w, http.StatusBadRequest)
		return
	}

	setRow := make([]string, 0)
	setVal := make([]interface{}, 0)

	validParams, err := t.validateParams(paramsToUpdate)
	if err != nil {
		log.Println("Validation failed:", err)
		fullAns := Response{"error": err.Error()}
		fullAns.SendResponse(w, http.StatusBadRequest)
		return
	}

	for colName, value := range validParams {
		setRow = append(setRow, fmt.Sprintf("`%v` = ?", colName))
		setVal = append(setVal, value)
	}

	setRowString := strings.Join(setRow, ", ")
	setVal = append(setVal, idInt)

	result, err := t.Database.Exec(fmt.Sprintf("UPDATE %s SET %v WHERE %s = ?", t.TableName, setRowString, idCol), setVal...)
	if err != nil {
		log.Println("Validation error:", err)
		fullAns := Response{"error": err}
		fullAns.SendResponse(w, http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		fullAns := Response{"error": err}
		fullAns.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fullAns := Response{"response": Response{"updated": affected}}
	fullAns.SendResponse(w, http.StatusOK)
	return
}

func (t *TableHandler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	idCol := t.getIDColumnName()
	if idCol == "" {
		resp := Response{"error": "cat not find id column"}
		resp.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fmt.Println("Id column name: " + idCol + " for table " + t.TableName)
	id := strings.Split(r.URL.Path, "/")[2]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		resp := Response{"error": "bad id value"}
		resp.SendResponse(w, http.StatusBadRequest)
		return
	}

	result, err := t.Database.Exec("DELETE FROM "+t.TableName+" WHERE "+idCol+"= ?", idInt)
	if err != nil {
		fullAns := Response{"error": err}
		fullAns.SendResponse(w, http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		fullAns := Response{"error": err}
		fullAns.SendResponse(w, http.StatusInternalServerError)
		return
	}

	fullAns := Response{"response": Response{"deleted": affected}}
	fullAns.SendResponse(w, http.StatusOK)
	return
}

func getDatabaseTableNames(db *sql.DB) []string {
	tables := make([]string, 0)
	allTables, _ := db.Query("SHOW TABLES;")
	for allTables.Next() {
		var table string
		err := allTables.Scan(&table)
		if err != nil {
			panic(err)
		}
		tables = append(tables, table)
	}

	err := allTables.Close()
	if err != nil {
		panic(err)
	}


	return tables
}

type ColumnInfo struct {
	Field      string
	Type       string
	Collation  sql.NullString
	Null       string
	Key        string
	Default    sql.NullString
	Extra      string
	Priveleges string
	Comment    string
}

func (ci *ColumnInfo) IsID() bool {
	if ci.Key == "PRI" || ci.Extra == "auto_increment" {
		return true
	}
	return false
}

func (ci *ColumnInfo) Validate(value interface{}) (interface{}, error) {

	if ci.IsID() {
		return nil, nil
	}

	switch {
	case strings.Contains(ci.Type, "int"):
		switch value.(type) {
		case int:
			return value.(int), nil
		case int64:
			return value.(int64), nil
		case int32:
			return value.(int32), nil
		case nil:
			if ci.Null == "YES" {
				return sql.NullInt32{}, nil
			} else {
				return 0, nil
			}
		case sql.NullInt32:
			if ci.Null == "YES" {
				return sql.NullInt32{}, nil
			} else {
				return 0, nil
			}
		default:
			return nil, fmt.Errorf("Transaction declined: insertion of value %v of type %T prohibited for column %v. Expected type int", value, value, ci.Field)
		}
	case strings.Contains(ci.Type, "float"):
		switch value.(type) {
		case float64:
			return value.(float64), nil
		case float32:
			return value.(float32), nil
		case nil:
			if ci.Null == "YES" {
				return sql.NullFloat64{}, nil
			} else {
				return float64(0), nil
			}
		case sql.NullFloat64:
			if ci.Null == "YES" {
				return sql.NullFloat64{}, nil
			} else {
				return float64(0), nil
			}
		default:
			return nil, fmt.Errorf("Transaction declined: insertion of value %v of type %T prohibited for column %v. Expected float", value, value, ci.Field)
		}
	default:
		switch value.(type) {
		case string:
			return value.(string), nil
		case []uint8:
			return string([]byte(value.([]uint8))), nil
		case nil:
			if ci.Null == "YES" {
				return sql.NullString{}, nil
			}
			return "", fmt.Errorf("Try to insert null value on column %v which not null", ci.Field)
		case sql.NullString:
			if ci.Null == "YES" {
				return sql.NullString{}, nil
			}
			return "", nil
		default:
			return nil, fmt.Errorf("Transaction declined: insertion of value %v of type %T prohibited for column %v. Expected string", value, value, ci.Field)
		}
	}
}

func getTableColumnsInfo(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	columns := []ColumnInfo{}
	colQuery, err := db.Query("SHOW FULL COLUMNS FROM " + tableName)
	if err != nil {
		return nil, err
	}
	for colQuery.Next() {
		var column ColumnInfo

		err := colQuery.Scan(
			&column.Field,
			&column.Type,
			&column.Collation,
			&column.Null,
			&column.Key,
			&column.Default,
			&column.Extra,
			&column.Priveleges,
			&column.Comment,
		)
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, nil
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {

	dbDescr := make(map[string][]ColumnInfo)

	tableNames := getDatabaseTableNames(db)
	fmt.Println("Get database with tables: ", tableNames)
	for _, table := range tableNames {
		cols, err := getTableColumnsInfo(db, table)
		if err != nil {
			panic(err)
		}
		dbDescr[table] = cols
	}

	h := NewHandler(db, dbDescr)

	return h, nil
}
