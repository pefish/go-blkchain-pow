package main

import (
	"fmt"
	"github.com/pefish/go-blkchain-pow/pow"
	"github.com/pefish/go-blkchain-pow/util"
	"github.com/pefish/go-logger"
	"math/big"
)

func main() {
	forever := make(chan string)

	go_logger.Logger.Init(`test`, ``) // 初始化日志打印
	powManager, err := pow.NewProofOfWorkManager(go_logger.Logger, pow.WithThreads(5)) // 设置5个线程
	if err != nil {
		panic(err)
	}
	result := make(chan *pow.Result)
	err = powManager.Work(&pow.Block{
		Header: &pow.Header{
			Difficulty: new(big.Int).Exp(big.NewInt(2), big.NewInt(8 * 3), big.NewInt(0)), // 难度初始化，应当是根据出块时间自动调整的，这里先设置固定值
		},
	}, result)
	if err != nil {
		panic(err)
	}
	select {
	case result_ := <- result: // 监听结果
		fmt.Println(result_.Nonce, result_.AttemptNum, util.BufferToHexString(result_.Hash, true))
	}


	<- forever // 进程挂起
}
