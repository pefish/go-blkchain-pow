package pow

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"errors"
	"github.com/pefish/go-blkchain-pow/util"
	"github.com/pefish/go-logger"
	"math"
	"math/big"
	"math/rand"
	"runtime"
)

type Block struct { // 区块结构
	Header *Header
	Body   *Body
}

type Body struct { // 区块体

}

type Header struct { // 区块头。自行添加字段
	Difficulty *big.Int
}

type ProofOfWorkManager struct {
	threads int                       // 开启的线程数
	rand    *rand.Rand                // 开始的nonce由这个函数随机出来
	logger  go_logger.InterfaceLogger // 日志打印
}

type ProofOfWorkManagerOptionFunc func(options *ProofOfWorkManagerOption)

type ProofOfWorkManagerOption struct {
	threads int
	rand    *rand.Rand
}

func WithThreads(threads int) ProofOfWorkManagerOptionFunc {
	return func(option *ProofOfWorkManagerOption) {
		option.threads = threads
	}
}

func WithRand(rand *rand.Rand) ProofOfWorkManagerOptionFunc {
	return func(option *ProofOfWorkManagerOption) {
		option.rand = rand
	}
}

func NewProofOfWorkManager(logger go_logger.InterfaceLogger, opts ...ProofOfWorkManagerOptionFunc) (*ProofOfWorkManager, error) {
	option := ProofOfWorkManagerOption{}
	for _, o := range opts {
		o(&option)
	}
	if option.threads == 0 {
		option.threads = runtime.NumCPU()
	}
	if option.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			return nil, err
		}
		option.rand = rand.New(rand.NewSource(seed.Int64()))
	}

	return &ProofOfWorkManager{
		threads: option.threads,
		rand:    option.rand,
		logger:  logger,
	}, nil
}

type Result struct {
	Nonce      uint64
	AttemptNum int64
	Hash []byte
}

// 开始计算
func (proofOfWorkManager *ProofOfWorkManager) Work(block *Block, result chan *Result) error {
	// 检查设置的线程数量
	if proofOfWorkManager.threads < 0 {
		return errors.New(`threads set error`)
	}
	abort := make(chan string)
	// 多开线程开始计算
	for i := 0; i < proofOfWorkManager.threads; i++ {
		initNonce := uint64(proofOfWorkManager.rand.Int63())
		proofOfWorkManager.logger.DebugF("Thread %d: InitNonce: %d", i, initNonce)
		go func(id int, nonce uint64) {
			proofOfWorkManager.mine(block, id, nonce, abort, result)
		}(i, initNonce)
	}
	return nil
}

// 计算逻辑
func (proofOfWorkManager *ProofOfWorkManager) mine(block *Block, threadId int, nonce uint64, abort chan string, found chan *Result) {
	var (
		hash = proofOfWorkManager.hashHeader(block.Header)
		// target = 2^256 / Difficulty
		target   = new(big.Int).Div(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0)), block.Header.Difficulty)
		attempts = int64(0)
	)
	proofOfWorkManager.logger.DebugF("Thread %d: Started search for new nonces", threadId)
search:
	for {
		select {
		case <-abort: // 如果收到abort，则退出求解
			proofOfWorkManager.logger.DebugF(`Thread %d: Abort`, threadId)
			break search
		default:
			attempts++
			// 计算hash
			hash := sha256.Sum256(bytes.Join([][]byte{
				hash,
				util.MustToBuffer(nonce),
			}, []byte{}))
			realHash := hash[:]
			if attempts%1000000 == 0 {
				proofOfWorkManager.logger.DebugF(`Thread %d: attempted %d, current hash: %s`, threadId, attempts, util.BufferToHexString(realHash, true))
			}
			if new(big.Int).SetBytes(realHash).Cmp(target) <= 0 {
				proofOfWorkManager.logger.DebugF(`Thread %d: Found`, threadId)
				// 找到了正确解
				select {
				case found <- &Result{ // 这里要等待别人接收结果
					Nonce:      nonce,
					AttemptNum: attempts,
					Hash: realHash,
				}:
					proofOfWorkManager.logger.DebugF("Thread %d: Nonce found and reported", threadId)
				case <-abort: // 可能两个线程同时算出来了，这里的效果就是丢弃多余的线程求出的解
					proofOfWorkManager.logger.DebugF("Thread %d: Nonce found but discarded", threadId)
				}
				go func() { // 开启新线程通知终止计算
					proofOfWorkManager.logger.DebugF("Thread %d: Stop all calc", threadId)
					close(abort) // 所有监听abort通道的都触发
				}()
			}
			nonce++
		}
	}
}

// 对区块头进行hash（不包含nonce）
func (proofOfWorkManager *ProofOfWorkManager) hashHeader(header *Header) []byte {
	hash := sha256.Sum256(bytes.Join([][]byte{
		util.MustToBuffer(header.Difficulty.Int64()),
	}, []byte{}))
	return hash[:]
}
