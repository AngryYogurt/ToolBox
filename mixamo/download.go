package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AngryYogurt/ToolBox/mixamo/config"
	"github.com/AngryYogurt/ToolBox/mixamo/model"
	"github.com/AngryYogurt/ToolBox/mixamo/utils"
	"github.com/AngryYogurt/ToolBox/task_manager"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var Animations []*model.Animation
var client *http.Client
var re *regexp.Regexp

var RecordFile = "record.txt"
var recordF *os.File
var GoroutineCount = 200
var Step = 200

func init() {
	re = regexp.MustCompile(`(?m).*filename="(.+?\..+?)".*`)
	client = &http.Client{}
	log.SetOutput(&lumberjack.Logger{
		Filename: "./mixamo/data/output.log",
		MaxSize:  50, // megabytes
	})

}

const (
	Info = iota
	Error
	Important
)

func Log(level int, v ...interface{}) {
	lv := "unknown"
	switch level {
	case Info:
		lv = "Info"
	case Error:
		lv = "Error"
	case Important:
		lv = "Important"
	}
	log.Println(lv, v)
}

func main() {
	var err error
	defer recordF.Close()

	InitAnimationList()
	// Start

	start, step := 0, Step
	for start < len(Animations) {
		end := start + step
		if recordF != nil {
			recordF.Close()
		}
		RecordFile = fmt.Sprintf("record_%d_%d.txt", start, end-1)
		recordF, err = os.OpenFile(filepath.Join(config.DataDir, RecordFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			Log(Error, err)
		}

		if end > len(Animations) {
			end = len(Animations)
		}
		anims := Animations[start : start+step]
		dls := genDLTaskList(anims)
		initCharacterDirs(dls)
		Download(dls)
		start = end
		Log(Important, fmt.Sprintf("finish range %d ~ %d", start, end-1))
	}
	return
}

func initCharacterDirs(dls []*model.DownloadTask) {
	for i, _ := range dls {
		path := filepath.Join(config.DataDir, strings.ReplaceAll(dls[i].CharacterName, "/", " "))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.Mkdir(path, os.ModeDir|os.ModePerm)
			if err != nil {
				log.Fatalln(err)
			}
		}
		dls[i].DataDirPath = path
	}
}

var mu = &sync.Mutex{}

func writeRecord(line string) {
	mu.Lock()
	defer mu.Unlock()
	if _, err := recordF.WriteString(line); err != nil {
		Log(Error, err)
	}
}

func writeFile(data string, fileName string) {
	path, _ := filepath.Abs(config.DataDir)
	f, _ := os.Create(filepath.Join(path, fileName))
	f.WriteString(data)
	defer f.Close()
}

func readFile(fileName string) string {
	path, _ := filepath.Abs(config.DataDir)
	fData, err := ioutil.ReadFile(filepath.Join(path, fileName))
	if err != nil {
		log.Fatalln(err)
	}
	return string(fData)
}

// Step 1: get animation list
func InitAnimationList() {
	animationData := readFile(config.AnimationListFile)
	if len(animationData) > 0 {
		err := json.Unmarshal([]byte(animationData), &Animations)
		if err == nil {
			return
		} else {
			log.Fatalln(err)
		}
	}
	totalPage := getTotalPages()
	tps := make([]*task_manager.TaskParam, 0)
	for i := 1; i <= totalPage; i++ {
		var tp interface{} = i
		tps = append(tps, &tp)
	}
	tm := task_manager.NewTaskManager(5*time.Second, task_manager.NewTask(tps, func(p *task_manager.TaskParam) *task_manager.TaskResult {
		var err error
		result := &task_manager.TaskResult{}
		page, ok := (*p).(int)
		if !ok {
			result.Err = fmt.Errorf("format param error")
			return result
		}
		u := fmt.Sprintf(config.AnimationListURL, page)
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		utils.BuildHeader(req)
		respData := utils.Request(client, req)
		animResp := &model.AnimationResult{}
		err = json.Unmarshal(respData, animResp)
		if err != nil {
			result.Err = err
		}
		result.Result = animResp
		Log(Info, "totalPage = %d, current page = %d, result count=%d\n", totalPage, page, animResp.Pagination.NumResults)
		return result
	}), 2)
	tm.Start().Wait()
	results := tm.GetTaskResult()
	animRes := make([]*model.AnimationResult, totalPage)
	for k, v := range results {
		page, _ := (*k).(int)
		r, _ := (v.Result).(*model.AnimationResult)
		animRes[page-1] = r
	}
	for _, v := range animRes {
		Animations = append(Animations, v.Result...)
	}
	animData, err := json.Marshal(Animations)
	if err != nil {
		log.Fatalln(err)
	}
	writeFile(string(animData), config.AnimationListFile)
}

func getTotalPages() int {
	page := 1
	u := fmt.Sprintf(config.AnimationListURL, page)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	utils.BuildHeader(req)
	animResp := &model.AnimationResult{}
	respData := utils.Request(client, req)
	err := json.Unmarshal(respData, animResp)
	if err != nil {
		log.Fatalln(err)
	}
	return animResp.Pagination.NumPages
}

func genDLTaskList(anims []*model.Animation) []*model.DownloadTask {
	dls := make([]*model.DownloadTask, 0)
	for i, _ := range anims {
		a := anims[i]
		for id, ch := range config.IDCharacters {
			dls = append(dls, &model.DownloadTask{
				CharacterName: ch,
				CharacterID:   id,
				GetProductURL: fmt.Sprintf(config.GetProductURL, a.Id, id),
				Animation:     a,
			})
		}
	}
	return dls
}

func Download(dls []*model.DownloadTask) {
	tps := make([]*task_manager.TaskParam, 0)
	for i := 0; i < len(dls); i++ {
		var tp interface{} = *(dls[i])
		tps = append(tps, &tp)
	}
	tm := task_manager.NewTaskManager(2*time.Second, task_manager.NewTask(tps, handleDownload), GoroutineCount)
	tm.Start().Wait()
	result := tm.GetTaskResult()
	for k, v := range result {
		if v.Err != nil {
			dt, _ := (result[k].Result).(model.DownloadTask)
			Log(Error, fmt.Sprintf("err=%v, failed task: %s", v.Err, dt.ToString()))

		}
	}
}

func checkExist(dt *model.DownloadTask) bool {
	fPath := filepath.Join(dt.DataDirPath, dt.Animation.Name)
	if dt.Animation.Type == "Motion" {
		fPath += ".fbx"
	} else {
		fPath += ".zip"
	}
	if info, err := os.Stat(fPath); !os.IsNotExist(err) && info.Size() > 0 {
		// file exist
		err = fmt.Errorf("duplicated download")
		return true
	}
	return false
}

func handleDownload(p *task_manager.TaskParam) *task_manager.TaskResult {
	var err error
	result := &task_manager.TaskResult{}
	d, ok := (*p).(model.DownloadTask)
	dt := &d
	if checkExist(dt) {
		return result
	}
	Log(Info, fmt.Sprintf("start c: %s, a:%s", dt.CharacterID, dt.Animation.Id))
	if !ok {
		result.Err = fmt.Errorf("format param error")
		dt.Error = result.Err
		return result
	}
	// Step 2: get product gms hash
	dt.Step = "start getProduct"
	err = getProduct(dt)
	if err != nil {
		result.Err = err
		return result
	}
	// Step 3: export animation from server to aws
	dt.Step = "start exportAnim"
	err = exportAnim(dt)
	if err != nil {
		result.Err = err
		return result
	}
	// Step 4: monitor export status
	dt.Step = "start monitor"
	err = monitor(dt)
	if err != nil {
		result.Err = err
		return result
	}
	// Step 5: download from aws
	dt.Step = "start downloadAws"
	err = downloadAws(dt)
	if err != nil {
		result.Err = err
		return result
	}
	writeRecord(fmt.Sprintf("%s|%s\n", dt.CharacterID, dt.Animation.Id))
	dt.IsDone = true
	Log(Info, fmt.Sprintf("finish proc task=%s", dt.ToString()))
	result.Result = dt
	return result
}

func downloadAws(dt *model.DownloadTask) error {
	resp, err := client.Get(dt.AwsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		Log(Error, err, resp)
		return err
	}
	defer resp.Body.Close()

	if info, err := os.Stat(dt.FilePath); !os.IsNotExist(err) && info.Size() > 0 {
		// file exist
		err = fmt.Errorf("duplicated download")
		return err
	}
	out, err := os.Create(dt.FilePath)
	if err != nil {
		Log(Error, err)
	}
	dt.Written, err = io.Copy(out, resp.Body)
	if err != nil || dt.Written <= 0 {
		err = fmt.Errorf("download aws error, err=%s, written=%d", err, dt.Written)
		Log(Error, err)
	}
	defer out.Close()
	return err
}

func monitor(dt *model.DownloadTask) error {
	var err error
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf(config.MonitorURL, dt.CharacterID), nil)
	utils.BuildHeader(req)
	for {
		switch dt.Monitor.Status {
		case "completed":
			dt.AwsURL = dt.Monitor.JobResult
			eUrl, _ := url.PathUnescape(dt.AwsURL)
			dt.FilePath = filepath.Join(dt.DataDirPath, re.FindStringSubmatch(eUrl)[1])
			return nil
		case "processing":
			time.Sleep(5 * time.Second)
			respData := utils.Request(client, req)
			exp := &model.Monitor{}
			err = json.Unmarshal(respData, exp)
			if err != nil {
				dt.Error = fmt.Errorf("monitor processing err, err=%v", err)
				return err
			}
			dt.Monitor = exp
		default:
			s := fmt.Sprintf("unexpected monitor status: %s, msg: %s", dt.Monitor.Status, dt.Monitor.Message)
			Log(Error, s)
			err = fmt.Errorf(s)
			dt.Error = err
			return err
		}
	}
}

func getProduct(dt *model.DownloadTask) error {
	prod := &model.Product{}
	var err error
	req, _ := http.NewRequest(http.MethodGet, dt.GetProductURL, nil)
	utils.BuildHeader(req)
	respData := utils.Request(client, req)
	err = json.Unmarshal(respData, prod)
	if err != nil {
		dt.Error = fmt.Errorf("get product err, err=%v", err)
		return err
	}
	dt.Product = prod
	return nil
}

func exportAnim(dt *model.DownloadTask) error {
	body := getExportBody(dt)
	req, _ := http.NewRequest(http.MethodPost, config.ExportAnimationURL, bytes.NewBuffer(body))
	utils.BuildHeader(req)
	respData := utils.Request(client, req)
	exp := &model.Monitor{}
	err := json.Unmarshal(respData, exp)
	if err != nil {
		dt.Error = fmt.Errorf("export anim err, err=%v", err)
		return err
	}
	dt.Monitor = exp
	return nil
}

func convertGmsHash(g *model.GmsHash) []*model.GmsHash {
	pVals := make([]string, 0)
	params, _ := g.Params.([]interface{})
	for i, _ := range params {
		pparams, _ := params[i].([]interface{})
		pv, _ := pparams[1].(int64)
		pVals = append(pVals, strconv.FormatInt(pv, 10))
	}
	newG := *g
	newG.Params = strings.Join(pVals, ",")
	return []*model.GmsHash{&newG}
}

func convertGmsHashs(dms []*model.DetailMotion) []*model.GmsHash {
	res := make([]*model.GmsHash, 0)
	for i, v := range dms {
		t := convertGmsHash(dms[i].GmsHash)
		t[0].Name = v.Name
		res = append(res, t[0])
	}
	return res
}

func getExportBody(t *model.DownloadTask) []byte {
	if len(t.ExportBody) > 0 {
		return []byte(t.ExportBody)
	}
	b := &ExpBody{
		CharacterID: t.CharacterID,
		Type:        t.Animation.Type,
		ProductName: t.Product.Name,
		Preferences: &Preferences{
			Format:   "fbx7",
			Fps:      config.Fps,
			Reducekf: "0",
		},
	}
	if b.Type == "Motion" {
		b.GmsHash = convertGmsHash(t.Product.Details.GmsHash)
		b.Preferences.Skin = config.WithSkin
	} else {
		b.GmsHash = convertGmsHashs(t.Product.Details.Motions)
		b.Preferences.MeshMotionpack = config.MeshMotionpack
	}
	res, _ := json.Marshal(b)
	t.ExportBody = string(res)
	return []byte(t.ExportBody)
}

type ExpBody struct {
	GmsHash     []*model.GmsHash `json:"gms_hash"`
	CharacterID string           `json:"character_id"`
	Type        string           `json:"type"`
	ProductName string           `json:"product_name"`
	Preferences *Preferences     `json:"preferences"`
}

type Preferences struct {
	Format         string `json:"format"`
	Skin           string `json:"skin"`
	Fps            string `json:"fps"`
	Reducekf       string `json:"reducekf"`
	MeshMotionpack string `json:"mesh_motionpack"`
}
