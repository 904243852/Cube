package handler

import (
	. "cube/internal"
	"cube/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func HandleSource(w http.ResponseWriter, r *http.Request) {
	var (
		data       interface{}
		returnless bool
		err        error
	)
	switch r.Method {
	case http.MethodPost:
		err = handleSourcePost(r)
	case http.MethodDelete:
		err = handleSourceDelete(r)
	case http.MethodPatch:
		err = handleSourcePatch(r)
	case http.MethodGet:
		data, returnless, err = handleSourceGet(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		toError(w, err)
		return
	}
	if !returnless {
		toSuccess(w, data)
	}
}

func handleSourceGet(w http.ResponseWriter, r *http.Request) (interface{}, bool, error) {
	// 初始化返回对象
	var data struct {
		Sources []model.Source `json:"sources"`
		Total   int            `json:"total"`
	}

	// 解析 URL 入参
	p := &QueryParams{r.URL.Query()}
	name, stype := p.Get("name"), p.GetOrDefault("type", "%")
	from, size := p.GetIntOrDefault("from", 0), p.GetIntOrDefault("size", 10)
	sort := p.Get("sort")

	// 初始化排序字段
	orders := "rowid desc"
	if ok, _ := regexp.MatchString("^(rowid|name|last_modified_date) (asc|desc)$", sort); ok {
		orders = sort
	}

	// 查询总数
	if err := Db.QueryRow("select count(1) from source where name like ? and type like ?", "%"+name+"%", stype).Scan(&data.Total); err != nil { // 调用 QueryRow 方法后，须调用 Scan 方法，否则连接将不会被释放
		return data, false, err
	}

	// 分页查询
	columns := "rowid, name, type, lang, content, compiled, active, method, url, cron, last_modified_date"
	if p.Has("basic") { // 不返回 content、compiled 字段，用于列表查询
		columns = "rowid, name, type, lang, '' content, '' compiled, active, method, url, cron, last_modified_date"
	}
	rows, err := Db.Query("select "+columns+" from source where name like ? and type like ? order by "+orders+" limit ?, ?", "%"+name+"%", stype, from, size)
	if err != nil {
		return data, false, err
	}
	defer rows.Close()
	for rows.Next() {
		source := model.Source{}
		rows.Scan(&source.Id, &source.Name, &source.Type, &source.Lang, &source.Content, &source.Compiled, &source.Active, &source.Method, &source.Url, &source.Cron, &source.LastModifiedDate)
		if source.Type == "daemon" { // 如果是 daemon，写入状态
			source.Status = fmt.Sprintf("%v", Cache.Daemons[source.Name] != nil)
		}
		data.Sources = append(data.Sources, source)
	}

	if p.Has("bulk") { // 导出为文件
		w.Header().Set("Content-Disposition", "attachment;filename=\"sources-"+strconv.FormatInt(time.Now().UnixMilli(), 10)+".json\"")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.Encode(data.Sources)
		return nil, true, nil
	}

	return data, false, err
}

func handleSourcePost(r *http.Request) error {
	// 读取请求消息体
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return err
	}

	if _, bulk := r.URL.Query()["bulk"]; !bulk {
		// 转换为 source 对象
		var source model.Source
		if err = json.Unmarshal(body, &source); err != nil {
			return err
		}

		// 校验类型
		if ok, _ := regexp.MatchString("^(module|controller|daemon|crontab|template|resource)$", source.Type); !ok {
			return errors.New("type of source is required, it must be module, controller, daemon, crontab, template or resource")
		}
		// 校验名称
		if source.Type == "module" {
			if ok, _ := regexp.MatchString("^(node_modules/)?\\w{2,32}$", source.Name); !ok {
				return errors.New("name of module is required, it must be a letter, number or underscore with a length of 2 to 32, it can also start with 'node_modules/'")
			}
		} else {
			if ok, _ := regexp.MatchString("^\\w{2,32}$", source.Name); !ok {
				return errors.New("name of " + source.Type + " is required, it must be a letter, number, or underscore with a length of 2 to 32")
			}
		}

		// 单个新增或修改，新增的均为去激活状态，无需刷新缓存
		if _, err := Db.Exec(strings.Join([]string{
			"update source set content = ?, compiled = ?, last_modified_date = datetime('now', 'localtime') where name = ? and type = ?",                         // 先尝试更新，再尝试新增
			"insert or ignore into source (name, type, lang, content, compiled, url, last_modified_date) values(?, ?, ?, ?, ?, ?, datetime('now', 'localtime'))", // 这里不用 insert or replace，replace 是替换整条记录
		}, ";"), source.Content, source.Compiled, source.Name, source.Type, source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Name); err != nil {
			return err
		}

		// 新增或更新路由
		if source.Type == "controller" && source.Url != "" {
			Cache.SetRoute(source.Name, source.Url)
		}

		// 清空 module 缓存以重建
		Cache.Modules = make(map[string]*goja.Program)
	} else { // 批量导入
		// 将请求入参转换为 source 对象数组
		var sources []model.Source
		if err = json.Unmarshal(body, &sources); err != nil {
			return err
		}

		if len(sources) == 0 {
			return errors.New("nothing added or modified")
		}

		// 批量新增或修改
		stmt, err := Db.Prepare("insert or replace into source (rowid, name, type, lang, content, compiled, active, method, url, cron, last_modified_date) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, source := range sources {
			if _, err = stmt.Exec(source.Id, source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Active, source.Method, source.Url, source.Cron, source.LastModifiedDate.String()); err != nil {
				return err
			}
		}

		Cache.InitRoutes()
		// 批量导入后，需要清空 module 缓存以重建
		Cache.Modules = make(map[string]*goja.Program)
		// 启动守护任务
		RunDaemons("")
		// 启动定时任务
		RunCrontabs("")
	}

	return nil
}

func handleSourceDelete(r *http.Request) error {
	r.ParseForm()
	name := r.Form.Get("name")
	if name == "" {
		return errors.New("parameter name is required")
	}
	stype := r.Form.Get("type")
	if stype == "" {
		return errors.New("parameter type is required")
	}

	res, err := Db.Exec("delete from source where name = ? and type = ?", name, stype)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("source does not existed")
	}

	// 删除路由
	if stype == "controller" {
		delete(Cache.Routes, name)
	}

	return nil
}

func handleSourcePatch(r *http.Request) error {
	// 读取请求消息体
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return err
	}
	// 转换为 source 对象
	var source model.Source
	if err = json.Unmarshal(body, &source); err != nil {
		return err
	}

	if source.Type == "controller" || source.Type == "resource" {
		// 校验 url 不能重复
		var count int
		if err = Db.QueryRow("select count(1) from source where type = ? and url = ? and active = true and name != ?", source.Type, source.Url, source.Name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("url already existed")
		}
	}

	// 修改
	res, err := Db.Exec("update source set active = ?, method = ?, url = ?, cron = ?, last_modified_date = datetime('now', 'localtime') where name = ? and type = ?", source.Active, source.Method, source.Url, source.Cron, source.Name, source.Type)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("source does not existed")
	}

	// 更新路由
	if source.Type == "controller" {
		Cache.SetRoute(source.Name, source.Url)
	}

	// 清空 module 缓存以重建
	Cache.Modules = make(map[string]*goja.Program)
	// 如果是 daemon，需要启动或停止
	if source.Type == "daemon" {
		if source.Active {
			if Cache.Daemons[source.Name] == nil && source.Status == "true" {
				RunDaemons(source.Name)
			}
			if Cache.Daemons[source.Name] != nil && source.Status == "false" {
				Cache.Daemons[source.Name].Interrupt("Daemon stopped")
			}
		}
	}
	// 如果是 crontab，需要启动或停止
	if source.Type == "crontab" {
		id, ok := Cache.Crontabs[source.Name]
		if !ok && source.Active {
			RunCrontabs(source.Name)
		}
		if ok && !source.Active {
			Crontab.Remove(id)
			delete(Cache.Crontabs, source.Name)
		}
	}

	return nil
}
