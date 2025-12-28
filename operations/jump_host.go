package operations

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
	"RancherMan/rancher"
)

// jumpHostProgressListener 跳板机进度监听器
type jumpHostProgressListener struct {
	infoArea *widget.Entry
}

// NewJumpHostProgressListener 创建跳板机进度监听器
func NewJumpHostProgressListener(infoArea *widget.Entry) rancher.ProgressListener {
	return &jumpHostProgressListener{
		infoArea: infoArea,
	}
}

func (l *jumpHostProgressListener) OnProgress(currentFolder string, current, total int) {
	// 直接更新 UI
	l.infoArea.SetText(fmt.Sprintf("正在扫描... %d/%d\n当前目录:%s", current, total, currentFolder))
}

func (l *jumpHostProgressListener) OnComplete() {
	// 直接更新 UI
	l.infoArea.SetText("更新跳板机完成")
}

func (l *jumpHostProgressListener) OnBatchResult(configs []rancher.SSHUploadConfig) {
	// 将 SSHUploadConfig 转换为 UploadConfig
	var uploadConfigs []rancher.UploadConfig
	for _, config := range configs {
		uploadConfig := rancher.UploadConfig{
			Dir:    config.Dir,
			Script: config.Script,
			Jar:    config.Jar,
			Image:  config.Image,
		}
		uploadConfigs = append(uploadConfigs, uploadConfig)
	}

	// 插入数据库
	gDb.InsertUploadConfigs(uploadConfigs)
}
