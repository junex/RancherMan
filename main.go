package main

import (
	"RancherMan/rancher"
	"log"
)

func main() {
	//// 创建数据库管理器实例
	dm, err := rancher.NewDatabaseManager("")
	if err != nil {
		log.Fatal(err)
	}
	defer dm.Close()

	ru := rancher.NewRancherUtils(dm)
	ru.UseNamespace("dev-tongguling-integration", "")
	ru.List()
}
