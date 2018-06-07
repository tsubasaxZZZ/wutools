package main

import (
	"log"

	"github.com/tsubasaxZZZ/wutools/common"
)

func main() {
	kbList := kb.NewKBList([]int{9999999, 4088891, 4103720, 4093120, 4091461, 4091664, 4093110})
	log.Println(*kbList)
}
