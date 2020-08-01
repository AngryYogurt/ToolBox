package model

import (
	"fmt"
	"path/filepath"
)

type DownloadTask struct {
	CharacterName string     `json:"character_name"`
	CharacterID   string     `json:"character_id"`
	GetProductURL string     `json:"get_product_url"`
	Product       *Product   `json:"product"`
	Monitor       *Monitor   `json:"monitor"`
	MonitorURL    string     `json:"monitor_url"`
	AwsURL        string     `json:"aws_url"`
	FilePath      string     `json:"file_path"`
	Animation     *Animation `json:"animation"`

	IsDone bool  `json:"is_done"`
	Error  error `json:"error"`

	DataDirPath string
	ExportBody  string
	Step        string
	Written     int64
}

const(
	FinalFileFormat = "%s---%s"
)

func (t *DownloadTask) GetFilePath() string {
	if len(t.FilePath) > 0 {
		return t.FilePath
	}
	fPath := filepath.Join(t.DataDirPath, t.Animation.Name)
	fPath = fmt.Sprintf(FinalFileFormat, fPath, t.Animation.Id)
	if t.Animation.Type == "Motion" {
		fPath += ".fbx"
	} else {
		fPath += ".zip"
	}
	t.FilePath = fPath
	return t.FilePath
}

func (t *DownloadTask) ToString() string {
	res := fmt.Sprintf("DTask = %+v, ", *t)
	if p := t.Product; p == nil {
		res += "prod = nil, "
	} else {
		res += fmt.Sprintf("prod=%+v, ", *p)
		if p.Details != nil {
			res += fmt.Sprintf("prod_det=%+v, ", *(p.Details))
			if p.Details.GmsHash != nil {
				res += fmt.Sprintf("prod_det_gms=%+v, ", *(p.Details.GmsHash))
			}
		}
	}

	if a := t.Animation; a == nil {
		res += "anim=nil, "
	} else {
		res += fmt.Sprintf("anim=%+v, motion_list_len=%d", *a, len(a.Motions))
	}
	return res
}
