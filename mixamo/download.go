package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AngryYogurt/ToolBox/mixamo/config"
	"github.com/AngryYogurt/ToolBox/mixamo/constant"
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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var Animations []*model.Animation
var client *http.Client
var re *regexp.Regexp

var recordF, failedF *os.File
var GoroutineCount = 100
var Step = 50

func init() {
	re = regexp.MustCompile(`(?m).*filename="(.+?\..+?)".*`)
	client = &http.Client{}
	log.SetOutput(&lumberjack.Logger{
		Filename: "./mixamo/data/output.log",
		MaxSize:  50, // megabytes
	})

	failedFile := fmt.Sprintf("failed_%s.txt", time.Now().Format("0102-15_04_05"))
	failedF, _ = os.OpenFile(filepath.Join(constant.DataDir, failedFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
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
	defer recordF.Close()
	InitAnimationList()

	//start := 0
	//for start < len(Animations) {
	//	end := start + Step
	//	if end > len(Animations) {
	//		end = len(Animations)
	//	}
	//	Log(Important, fmt.Sprintf("start range %d ~ %d", start, end-1))
	//
	//	if recordF != nil {
	//		recordF.Close()
	//	}
	//	recordFile := fmt.Sprintf("record_%d_%d_%s.txt", start, end-1, time.Now().Format("0102-15_04_05"))
	//	recordF, _ = os.OpenFile(filepath.Join(config.DataDir, recordFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	//
	//	dls := genDLTaskList(Animations[start:end])
	//	Download(dls)
	//
	//	Log(Important, fmt.Sprintf("finish range %d ~ %d", start, end-1))
	//	start = end
	//	time.Sleep(5 * time.Second)
	//}
	return
}

func checkExist(fPath string) bool {
	if info, err := os.Stat(fPath); !os.IsNotExist(err) && info.Size() > 0 {
		return true
	}
	return false
}

func genDLTaskList(anims []*model.Animation) []*model.DownloadTask {
	dls := make([]*model.DownloadTask, 0)
	skipCount := 0
	for i, _ := range anims {
		a := anims[i]
		for id, ch := range config.IDCharacters {
			dt := &model.DownloadTask{
				CharacterName: ch,
				CharacterID:   id,
				GetProductURL: fmt.Sprintf(constant.GetProductURL, a.Id, id),
				Animation:     a,
				LocationDir:   filepath.Join(constant.DataDir, strings.ReplaceAll(ch, "/", " ")),
			}
			if _, err := os.Stat(dt.LocationDir); os.IsNotExist(err) {
				err := os.Mkdir(dt.LocationDir, os.ModeDir|os.ModePerm)
				if err != nil {
					log.Fatalln(err)
				}
			}
			if checkExist(dt.GetTargetPath()) {
				skipCount++
				//Log(Info, fmt.Sprintf("skip c: %s, a:%s", dt.CharacterID, dt.Animation.Id))
				continue
			}
			dls = append(dls, dt)
		}
	}
	Log(Important, fmt.Sprintf("skip %d finished tasks", skipCount))
	return dls
}

var recMu = &sync.Mutex{}
var failMu = &sync.Mutex{}

func writeRecord(line string) {
	recMu.Lock()
	defer recMu.Unlock()
	if _, err := recordF.WriteString(line); err != nil {
		Log(Error, err)
	}
}

func writeFailedRecord(line string) {
	failMu.Lock()
	defer failMu.Unlock()
	if _, err := failedF.WriteString(line); err != nil {
		Log(Error, err)
	}
}

func writeFile(data string, fileName string) {
	path, _ := filepath.Abs(constant.DataDir)
	f, _ := os.Create(filepath.Join(path, fileName))
	f.WriteString(data)
	defer f.Close()
}

func readFile(fileName string) string {
	path, _ := filepath.Abs(constant.DataDir)
	fData, err := ioutil.ReadFile(filepath.Join(path, fileName))
	if err != nil {
		log.Fatalln(err)
	}
	return string(fData)
}

func marshalAnim(f string, am map[string]*model.Animation) map[string]*model.Animation {
	animationData := readFile(f)
	tmp := make([]*model.Animation, 0)
	_ = json.Unmarshal([]byte(animationData), &tmp)
	for i, a := range tmp {
		am[a.Id+a.Name] = tmp[i]
	}
	return am
}

// Prepare animation list file, only for first time running this script
func PullAnimationList() {
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
		u := fmt.Sprintf(constant.AnimationListURL, page)
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		utils.BuildHeader(req)
		respData, err := utils.Request(client, req, 0)
		if err != nil {
			result.Err = err
			return result
		}
		animResp := &model.AnimationResult{}
		err = json.Unmarshal(respData, animResp)
		if err != nil {
			result.Err = err
			return result
		}
		result.Result = animResp
		Log(Info, fmt.Sprintf("totalPage = %d, current page = %d, result count=%d\n", totalPage, page, animResp.Pagination.NumResults))
		return result
	}), 15)
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
	writeFile(string(animData), constant.AnimationListFile)
}

// Step 1: get animation list
func InitAnimationList() {
	readFile(constant.AllAnimationListFile)
	am := make(map[string]*model.Animation)
	am = marshalAnim(constant.AnimationListFile, am)
	am = marshalAnim(constant.AnimationListFile2, am)
	am = marshalAnim(constant.AnimationListFile3, am)
	am = marshalAnim(constant.AnimationListFile4, am)
	am = marshalAnim(constant.AnimationListFile5, am)
	Animations = make([]*model.Animation, 0, len(am))
	for k, _ := range am {
		if am[k] == nil {
			fmt.Println()
		}
		Animations = append(Animations, am[k])
	}

	sort.Slice(Animations, func(i, j int) bool {
		return Animations[i].Name+Animations[i].Id < Animations[j].Name+Animations[j].Id
	})

	animData, err := json.MarshalIndent(Animations, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}
	writeFile(string(animData), constant.AllAnimationListFile)
}

func getTotalPages() int {
	page := 1
	u := fmt.Sprintf(constant.AnimationListURL, page)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	utils.BuildHeader(req)
	animResp := &model.AnimationResult{}
	respData, err := utils.Request(client, req, 0)
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(respData, animResp)
	if err != nil {
		log.Fatalln(err)
	}
	return animResp.Pagination.NumPages
}

func Download(dls []*model.DownloadTask) {
	tps := make([]*task_manager.TaskParam, 0)
	for i := 0; i < len(dls); i++ {
		var tp interface{} = *(dls[i])
		tps = append(tps, &tp)
	}
	Log(Info, len(tps), "tasks, Start!")
	tm := task_manager.NewTaskManager(200*time.Millisecond, task_manager.NewTask(tps, handleDownload), GoroutineCount)
	tm.Start().Wait()
	result := tm.GetTaskResult()
	for k, v := range result {
		if v.Err != nil {
			dt, _ := (*k).(model.DownloadTask)
			writeFailedRecord(fmt.Sprintf("%s|%s", dt.CharacterID, dt.Animation.Id))
			Log(Error, fmt.Sprintf("err=%v, failed task: %s", v.Err, dt.ToString()))
		}
	}
}

func handleDownload(p *task_manager.TaskParam) (result *task_manager.TaskResult) {
	var err error
	result = &task_manager.TaskResult{}
	d, ok := (*p).(model.DownloadTask)
	dt := &d
	result.Result = dt
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
	Log(Info, fmt.Sprintf("finish proc task=%s|%s", dt.CharacterID, dt.Animation.Id))
	return result
}

func downloadAws(dt *model.DownloadTask) error {
	resp, err := client.Get(dt.AwsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		Log(Error, err, resp)
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(dt.GetTempPath())
	if err != nil {
		Log(Error, err)
		return err
	}
	defer out.Close()

	dt.Written, err = io.Copy(out, resp.Body)
	if err != nil || dt.Written <= 0 {
		err = fmt.Errorf("download aws error, err=%s, written=%d", err, dt.Written)
		Log(Error, err)
		return err
	}
	err = os.Rename(dt.GetTempPath(), dt.GetTargetPath())
	return err
}

func monitor(dt *model.DownloadTask) error {
	var err error
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf(constant.MonitorURL, dt.CharacterID), nil)
	utils.BuildHeader(req)
	for {
		switch dt.Monitor.Status {
		case "completed":
			dt.AwsURL = dt.Monitor.JobResult
			eUrl, _ := url.PathUnescape(dt.AwsURL)
			fPath := filepath.Join(dt.LocationDir, re.FindStringSubmatch(eUrl)[1])
			fExt := filepath.Ext(fPath)
			target := fmt.Sprintf(model.FinalFileFormat, fPath[:len(fPath)-len(fExt)], dt.Animation.Id) + fExt
			if checkExist(target) {
				err = fmt.Errorf("conflict target file, target=%s", target)
				Log(Error, err)
				return err
			}
			return nil
		case "processing":
			time.Sleep(5 * time.Second)
			respData, err := utils.Request(client, req, 0)
			if err != nil {
				dt.Error = err
				return err
			}
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
	respData, err := utils.Request(client, req, 0)
	if err != nil {
		dt.Error = err
		return err
	}
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
	req, _ := http.NewRequest(http.MethodPost, constant.ExportAnimationURL, bytes.NewBuffer(body))
	utils.BuildHeader(req)
	respData, err := utils.Request(client, req, 0)
	if err != nil {
		dt.Error = err
		return err
	}
	exp := &model.Monitor{}
	err = json.Unmarshal(respData, exp)
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
			Fps:      constant.Fps,
			Reducekf: "0",
		},
	}
	if b.Type == "Motion" {
		b.GmsHash = convertGmsHash(t.Product.Details.GmsHash)
		b.Preferences.Skin = constant.WithSkin
	} else {
		b.GmsHash = convertGmsHashs(t.Product.Details.Motions)
		b.Preferences.MeshMotionpack = constant.MeshMotionpack
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
