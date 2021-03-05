package converter

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

//map for converting mysql type to golang types
var typeForMysqlToGo = map[string]string{
	"int":                         "int64",
	"integer":                     "int64",
	"tinyint":                     "int64",
	"smallint":                    "int64",
	"mediumint":                   "int64",
	"bigint":                      "int64",
	"int unsigned":                "int64",
	"integer unsigned":            "int64",
	"tinyint unsigned":            "int64",
	"smallint unsigned":           "int64",
	"mediumint unsigned":          "int64",
	"bigint unsigned":             "int64",
	"bit":                         "int64",
	"bool":                        "bool",
	"enum":                        "string",
	"set":                         "string",
	"varchar":                     "string",
	"char":                        "string",
	"tinytext":                    "string",
	"mediumtext":                  "string",
	"text":                        "string",
	"longtext":                    "string",
	"blob":                        "string",
	"tinyblob":                    "string",
	"mediumblob":                  "string",
	"longblob":                    "string",
	"date":                        "time.Time", // time.Time or string
	"datetime":                    "time.Time", // time.Time or string
	"timestamp":                   "time.Time", // time.Time or string
	"time":                        "time.Time", // time.Time or string
	"float":                       "float64",
	"double":                      "float64",
	"decimal":                     "float64",
	"binary":                      "string",
	"varbinary":                   "string",
	"json":                        "json.RawMessage",
	"nullable.int":                "sql.Int64",
	"nullable.integer":            "sql.Int64",
	"nullable.tinyint":            "sql.Int64",
	"nullable.smallint":           "sql.Int64",
	"nullable.mediumint":          "sql.Int64",
	"nullable.bigint":             "sql.Int64",
	"nullable.int unsigned":       "sql.Int64",
	"nullable.integer unsigned":   "sql.Int64",
	"nullable.tinyint unsigned":   "sql.Int64",
	"nullable.smallint unsigned":  "sql.Int64",
	"nullable.mediumint unsigned": "sql.Int64",
	"nullable.bigint unsigned":    "sql.Int64",
	"nullable.bit":                "sql.Int64",
	"nullable.bool":               "sql.NullBool",
	"nullable.enum":               "sql.NullString",
	"nullable.set":                "sql.NullString",
	"nullable.varchar":            "sql.NullString",
	"nullable.char":               "sql.NullString",
	"nullable.tinytext":           "sql.NullString",
	"nullable.mediumtext":         "sql.NullString",
	"nullable.text":               "sql.NullString",
	"nullable.longtext":           "sql.NullString",
	"nullable.blob":               "sql.NullString",
	"nullable.tinyblob":           "sql.NullString",
	"nullable.mediumblob":         "sql.NullString",
	"nullable.longblob":           "sql.NullString",
	"nullable.date":               "sql.NullTime", // time.Time or string
	"nullable.datetime":           "sql.NullTime", // time.Time or string
	"nullable.timestamp":          "sql.NullTime", // time.Time or string
	"nullable.time":               "sql.NullTime", // time.Time or string
	"nullable.float":              "sql.NullFloat64",
	"nullable.double":             "sql.NullFloat64",
	"nullable.decimal":            "sql.NullFloat64",
	"nullable.binary":             "sql.NullString",
	"nullable.varbinary":          "sql.NullString",
	"nullable.json":               "sql.NullString",
}

type Table2Struct struct {
	dsn            string
	savePath       string
	db             *sql.DB
	table          string
	prefix         string
	config         *T2tConfig
	err            error
	realNameMethod string
	enableJsonTag  bool // 是否添加json的tag, 默认不添加
	genTableTags   bool
	packageName    string // 生成struct的包名(默认为空的话, 则取名为: package model)
	tagKey         string // tag字段的key值,默认是orm
	dateToTime     bool   // 是否将 date相关字段转换为 time.Time,默认否
}

type T2tConfig struct {
	StructNameToHump bool // 结构体名称是否转为驼峰式，默认为false
	RmTagIfUcFirsted bool // 如果字段首字母本来就是大写, 就不添加tag, 默认false添加, true不添加
	TagToLower       bool // tag的字段名字是否转换为小写, 如果本身有大写字母的话, 默认false不转
	JsonTagToHump    bool // json tag是否转为驼峰，默认为false，不转换
	UcFirstOnly      bool // 字段首字母大写的同时, 是否要把其他字母转换为小写,默认false不转换
	SeperatFile      bool // 每个struct放入单独的文件,默认false,放入同一个文件
	GenNullableType  bool // 可空字段是否生成sql.Null*类型
}

func NewTable2Struct() *Table2Struct {
	return &Table2Struct{}
}

func (t *Table2Struct) Dsn(d string) *Table2Struct {
	t.dsn = d
	return t
}

func (t *Table2Struct) TagKey(r string) *Table2Struct {
	t.tagKey = r
	return t
}

func (t *Table2Struct) PackageName(r string) *Table2Struct {
	t.packageName = r
	return t
}

func (t *Table2Struct) RealNameMethod(r string) *Table2Struct {
	t.realNameMethod = r
	return t
}

func (t *Table2Struct) SavePath(p string) *Table2Struct {
	t.savePath = p
	return t
}

func (t *Table2Struct) DB(d *sql.DB) *Table2Struct {
	t.db = d
	return t
}

func (t *Table2Struct) Table(tab string) *Table2Struct {
	t.table = tab
	return t
}

func (t *Table2Struct) Prefix(p string) *Table2Struct {
	t.prefix = p
	return t
}

func (t *Table2Struct) EnableJsonTag(p bool) *Table2Struct {
	t.enableJsonTag = p
	return t
}

func (t *Table2Struct) EnableTableTags(p bool) *Table2Struct {
	t.genTableTags = p
	return t
}

func (t *Table2Struct) DateToTime(d bool) *Table2Struct {
	t.dateToTime = d
	return t
}

func (t *Table2Struct) Config(c *T2tConfig) *Table2Struct {
	t.config = c
	return t
}

func (t *Table2Struct) Run() error {
	if t.config == nil {
		t.config = new(T2tConfig)
	}
	// 链接mysql, 获取db对象
	t.dialMysql()
	if t.err != nil {
		return t.err
	}

	// 获取表和字段的shcema
	tableColumns, err := t.getColumns()
	if err != nil {
		return err
	}

	// 包名
	var packageName string
	if t.packageName == "" {
		packageName = "package model\n\n"
	} else {
		packageName = fmt.Sprintf("package %s\n\n", t.packageName)
	}

	// 组装struct
	var structContent string
	for tableRealName, item := range tableColumns {
		// 去除前缀
		if t.prefix != "" {
			tableRealName = tableRealName[len(t.prefix):]
		}
		tableName := tableRealName
		structName := tableName
		if t.config.StructNameToHump {
			structName = t.camelCase(structName)
		}

		switch len(tableName) {
		case 0:
		case 1:
			tableName = strings.ToUpper(tableName[0:1])
		default:
			// 字符长度大于1时
			tableName = strings.ToUpper(tableName[0:1]) + tableName[1:]
		}
		depth := 1
		structContent += "type " + structName + " struct {\n"
		for _, v := range item {
			//structContent += tab(depth) + v.ColumnName + " " + v.Type + " " + v.Json + "\n"
			// 字段注释
			var clumnComment string
			if v.ColumnComment != "" {
				clumnComment = fmt.Sprintf(" // %s", v.ColumnComment)
			}
			structContent += fmt.Sprintf("%s%s %s %s%s\n",
				tab(depth), v.ColumnName, v.Type, v.Tag, clumnComment)
		}
		structContent += tab(depth-1) + "}\n\n"

		// 添加 method 获取真实表名
		if t.realNameMethod != "" {
			structContent += fmt.Sprintf("func (%s) %s() string {\n",
				structName, t.realNameMethod)
			structContent += fmt.Sprintf("%sreturn \"%s\"\n",
				tab(depth), tableRealName)
			structContent += "}\n\n"
		}
	}

	// 如果有引入 time.Time, 则需要引入 time 包
	var importContent string
	if strings.Contains(structContent, "time.Time") {
		importContent = "import \"time\"\n\n"
	}

	// 添加json类型支持
	if strings.Contains(structContent, "json.RawMessage") {
		importContent += "import \"encoding/json\"\n\n"
	}

	// 支持sql.Null*
	if strings.Contains(structContent, "sql.") {
		importContent += "import \"database/sql\"\n\n"
	}

	// 写入文件struct
	var savePath = t.savePath
	// 是否指定保存路径
	if savePath == "" {
		savePath = "model.go"
	}
	filePath := fmt.Sprintf("%s", savePath)
	f, err := os.Create(filePath)
	if err != nil {
		log.Println("Can not write file")
		return err
	}
	defer f.Close()

	f.WriteString(packageName + importContent + structContent)

	cmd := exec.Command("gofmt", "-w", filePath)
	cmd.Run()

	log.Println("gen model finish!!!")

	return nil
}

func (t *Table2Struct) dialMysql() {
	if t.db == nil {
		if t.dsn == "" {
			t.err = errors.New("dsn数据库配置缺失")
			return
		}
		t.db, t.err = sql.Open("mysql", t.dsn)
	}
	return
}

type column struct {
	ColumnName    string
	Type          string
	Nullable      string
	TableName     string
	ColumnComment string
	Tag           string
}

func (c *column) genTags(sqlTagKey string, enableJSONTag bool, jsonTag string, genTableTags bool) string {
	tagList := make(map[string]string)
	tagList[sqlTagKey] = c.Tag
	if enableJSONTag {
		tagList["json"] = jsonTag
	}
	if genTableTags {
		commentSegments := strings.Split(c.ColumnComment, "|")
		if len(commentSegments) > 1 {
			tagList["t_name"] = commentSegments[0]
		} else {
			tagList["t_name"] = c.ColumnName
		}
		switch c.Type {
		case "int64":
			tagList["t_fm"] = "SINT"
			tagList["t_vt"] = "INT"
			tagList["t_order"] = "true"
		case "bool":
			tagList["t_fm"] = "SBOOL"
			tagList["t_vt"] = "INT"
		case "time.Time":
			tagList["t_fm"] = "STIME"
			tagList["t_vt"] = "TIME"
			tagList["t_order"] = "true"
		case "float64":
			tagList["t_fm"] = "SDOUBLE"
			tagList["t_vt"] = "DOUBLE"
			tagList["t_order"] = "true"
		case "string":
			tagList["t_fm"] = "SSTR"
			tagList["t_vt"] = "STRING"
		default:
			tagList["t_ignore"] = "true"
		}
		if _, ok := tagList["t_fm"]; ok {
			if c.Nullable == "YES" {
				tagList["t_fm"] += "|SNULLABLE"
			}
		}
	}
	var tagKeys []string
	// sort keys
	for k := range tagList {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)
	var tags []string
	for _, k := range tagKeys {
		v := tagList[k]
		if v != "" {
			tags = append(tags, fmt.Sprintf("%s:\"%s\"", k, v))
		} else {
			tags = append(tags, fmt.Sprintf("%s", k))
		}
	}
	return "`" + strings.Join(tags, " ") + "`"
}

// Function for fetching schema definition of passed table
func (t *Table2Struct) getColumns(table ...string) (tableColumns map[string][]column, err error) {
	// 根据设置,判断是否要把 date 相关字段替换为 string
	if t.dateToTime == false {
		typeForMysqlToGo["date"] = "string"
		typeForMysqlToGo["datetime"] = "string"
		typeForMysqlToGo["timestamp"] = "string"
		typeForMysqlToGo["time"] = "string"
	}
	tableColumns = make(map[string][]column)
	// sql
	var sqlStr = `SELECT COLUMN_NAME,DATA_TYPE,IS_NULLABLE,TABLE_NAME,COLUMN_COMMENT
		FROM information_schema.COLUMNS 
		WHERE table_schema = DATABASE()`
	// 是否指定了具体的table
	if t.table != "" {
		sqlStr += fmt.Sprintf(" AND TABLE_NAME = '%s'", t.prefix+t.table)
	}
	// sql排序
	sqlStr += " order by TABLE_NAME asc, ORDINAL_POSITION asc"

	rows, err := t.db.Query(sqlStr)
	if err != nil {
		log.Println("Error reading table information: ", err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		col := column{}
		err = rows.Scan(&col.ColumnName, &col.Type, &col.Nullable, &col.TableName, &col.ColumnComment)

		if err != nil {
			log.Println(err.Error())
			return
		}

		//col.Json = strings.ToLower(col.ColumnName)
		col.Tag = col.ColumnName
		col.ColumnName = t.camelCase(col.ColumnName)
		if t.config.GenNullableType {
			col.Type = "nullable." + col.Type
		}
		col.Type = typeForMysqlToGo[col.Type]
		jsonTag := col.Tag
		// 字段首字母本身大写, 是否需要删除tag
		if t.config.RmTagIfUcFirsted &&
			col.ColumnName[0:1] == strings.ToUpper(col.ColumnName[0:1]) {
			col.Tag = "-"
		} else {
			// 是否需要将tag转换成小写
			if t.config.TagToLower {
				col.Tag = strings.ToLower(col.Tag)
				jsonTag = col.Tag
			}

			if t.config.JsonTagToHump {
				jsonTag = t.camelCase(jsonTag)
			}

			//if col.Nullable == "YES" {
			//	col.Json = fmt.Sprintf("`json:\"%s,omitempty\"`", col.Json)
			//} else {
			//}
		}
		if t.tagKey == "" {
			t.tagKey = "orm"
		}
		col.Tag = col.genTags(t.tagKey, t.enableJsonTag, jsonTag, t.genTableTags)
		//columns = append(columns, col)
		if _, ok := tableColumns[col.TableName]; !ok {
			tableColumns[col.TableName] = []column{}
		}
		tableColumns[col.TableName] = append(tableColumns[col.TableName], col)
	}
	return
}

func (t *Table2Struct) camelCase(str string) string {
	// 是否有表前缀, 设置了就先去除表前缀
	if t.prefix != "" {
		str = strings.Replace(str, t.prefix, "", 1)
	}
	var text string
	//for _, p := range strings.Split(name, "_") {
	for _, p := range strings.Split(str, "_") {
		// 字段首字母大写的同时, 是否要把其他字母转换为小写
		switch len(p) {
		case 0:
		case 1:
			text += strings.ToUpper(p[0:1])
		default:
			// 字符长度大于1时
			if t.config.UcFirstOnly == true {
				text += strings.ToUpper(p[0:1]) + strings.ToLower(p[1:])
			} else {
				text += strings.ToUpper(p[0:1]) + p[1:]
			}
		}
	}
	return text
}
func tab(depth int) string {
	return strings.Repeat("\t", depth)
}
