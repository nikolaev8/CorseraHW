package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/peer"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func parseACL(aclJSON string) (map[string][]string, error) {
	var acl map[string][]string

	err := json.Unmarshal([]byte(aclJSON), &acl)
	if err != nil {
		return nil, err
	}
	return acl, nil
}

type LogChanPool struct {
	mu       sync.RWMutex
	LogChans map[string]chan *Event
}

func NewLogChanPool() *LogChanPool {
	p := make(map[string]chan *Event)
	return &LogChanPool{LogChans: p}
}

func (pool *LogChanPool) Add(consName string) (chan *Event) {
	fmt.Println("Add channel for consumer", consName)
	pool.mu.RLock()
	if _, ok := pool.LogChans[consName]; ok {
		return nil
	}
	pool.mu.RUnlock()

	ch := make(chan *Event)
	pool.mu.Lock()
	pool.LogChans[consName] = ch
	pool.mu.Unlock()
	return ch
}

func (pool *LogChanPool) Delete(consName string) {
	fmt.Println("Delete channel for consumer", consName)
	pool.mu.Lock()
	defer pool.mu.Unlock()

	delete(pool.LogChans, consName)
}

func (pool *LogChanPool) SendEvent(msg *Message) {
	fmt.Println("Send message for all subscribers")
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	for _, ch := range pool.LogChans{
		ch <- msg.msgBody
	}
}


func (pool *LogChanPool) Close() {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	for _, ch := range pool.LogChans {
		close(ch)
	}
}

func (pool *LogChanPool) DeleteAll() {
	fmt.Println("Delete all channels")
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for consID:= range pool.LogChans {
		delete(pool.LogChans, consID)
	}
}

type SyncStat struct {
	mu sync.RWMutex
	Stat
}

func (ss *SyncStat) SetTime(){
	ss.mu.Lock()
	ss.Timestamp = time.Now().Unix()
	ss.mu.Unlock()
}

func (ss *SyncStat) AddByMethod (methodName string) {
	ss.mu.Lock()
	ss.Stat.ByMethod[methodName] += 1
	ss.mu.Unlock()
}

func (ss *SyncStat) AddByConsumer (consName string) {
	ss.mu.Lock()
	ss.Stat.ByConsumer[consName] += 1
	ss.mu.Unlock()
}


func (ss *SyncStat) GetConsStat () map[string]uint64{
	ans := make(map[string]uint64)
	ss.mu.RLock()
	st := ss.Stat.ByConsumer
	for k, v := range st{
		ans[k] = v
	}
	ss.mu.RUnlock()
	return ans
}

func (ss *SyncStat) GetMethodStat () map[string]uint64{
	ans := make(map[string]uint64)
	ss.mu.RLock()
	st := ss.Stat.ByMethod
	for k, v := range st{
		ans[k] = v
	}
	ss.mu.RUnlock()
	return ans
}

func (ss *SyncStat) Diff (consStat, methStat map[string]uint64) *Stat {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	ans := NewStat()
	for meth := range ss.ByMethod {
		if d := ss.ByMethod[meth] - methStat[meth]; d == 0 {
			continue
		} else {
			ans.ByMethod[meth] = d
		}
	}

	for cons := range ss.ByConsumer {
		if d := ss.ByConsumer[cons] - consStat[cons]; d == 0 {
			continue
		} else {
			ans.ByConsumer[cons] = d
		}
	}
	ans.Timestamp = time.Now().Unix()

	return ans
}


func NewStat() *Stat {
	byMethod := make(map[string]uint64)
	byConsumer := make(map[string]uint64)

	ss := &Stat{}
	ss.ByMethod = byMethod
	ss.ByConsumer = byConsumer
	return ss
}

func NewSyncStat() *SyncStat{
	byMethod := make(map[string]uint64)
	byConsumer := make(map[string]uint64)

	ss := &SyncStat{}
	ss.Stat.ByMethod = byMethod
	ss.Stat.ByConsumer = byConsumer
	return ss
}

func StartMyMicroservice(ctx context.Context, addr string, acl string) error {

	access, err := parseACL(acl)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	server := grpc.NewServer()

	logChan := make(chan *Message, 5)

	stat := NewSyncStat()
	readLogChans := NewLogChanPool()

	RegisterAdminServer(server, NewAdmin(access, readLogChans, stat, logChan))
	RegisterBizServer(server, NewBiz(access, logChan))

	go func() {
		fmt.Println("starting server at", addr)
		err := server.Serve(lis)
		if err != nil{
			panic("Can not start server")
		}
	}()

	// начинаем читать логи и считать статистику
	go func() {
		for {
			select {
			case msg := <-logChan:
				readLogChans.SendEvent(msg)

				stat.SetTime()
				stat.AddByConsumer(msg.msgBody.GetConsumer())
				stat.AddByMethod(msg.msgBody.GetMethod())

			case <-ctx.Done():
				close(logChan)
				readLogChans.Close()
				readLogChans.DeleteAll()
				server.Stop()
				//err := lis.Close()
				//if err != nil {
				//	panic("Can not close connection")
				//}
				return
			}
		}
	}()
	return nil
}

type Message struct {
	consumerID string
	msgBody *Event
}

type Admin struct {
	AccessList map[string][]string
	mu         sync.RWMutex
	logChans   *LogChanPool
	stat       *SyncStat
	adminLogW  chan *Message
}

func (adm *Admin) checkAccess(methodName string, consName string) bool {

	acc, ok := adm.AccessList[consName]
	if !ok {
		return false
	}

	approved := false
	for _, acs := range acc {
		if acs != "/main.Admin/*" && acs != "/main.Admin/"+methodName {
			continue
		}
		approved = true
		break
	}

	if !approved {
		return false
	}

	return true
}

func NewAdmin(acl map[string][]string, lc *LogChanPool, s *SyncStat, admLog chan *Message) *Admin {
	return &Admin{AccessList: acl,
		logChans:  lc,
		stat:      s,
		mu:        sync.RWMutex{},
		adminLogW: admLog,
	}
}

func (adm *Admin) Logging(nthg *Nothing, inStream Admin_LoggingServer) error {
	ctx := inStream.Context()

	consName, err := getConsumerName(ctx)
	if err != nil {
		return status.Error(codes.InvalidArgument, "Unknown consumer")
	}

	approved := adm.checkAccess("Logging", consName)
	if !approved {
		return status.Error(codes.Unauthenticated, "Access denied")
	}

	consID := consName + strconv.Itoa(rand.Int())

	p, ok := peer.FromContext(ctx)
	if !ok {
		return fmt.Errorf("Can not read host")
	}

	ev := &Message{
		consumerID: consID,
		msgBody:
		&Event{
			Timestamp: time.Now().Unix(),
			Method: "/main.Admin/Logging",
			Consumer: consName,
			Host: p.Addr.String(),
		},
	}

	adm.logChans.SendEvent(ev)

	lc := adm.logChans.Add(consID)

	if lc == nil {
		return status.Error(codes.Internal, "Can not add consumer channel to pool")
	}

	for {
		err := inStream.Send(<-lc)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (adm *Admin) Statistics(interval *StatInterval, inStream Admin_StatisticsServer) error {
	ctx := inStream.Context()

	ticker := time.NewTicker(time.Second * time.Duration(interval.IntervalSeconds))

	consName, err := getConsumerName(ctx)
	if err != nil {
		return status.Error(codes.InvalidArgument, "Unknown consumer")
	}

	approved := adm.checkAccess("Statistics", consName)

	if !approved {
		return status.Error(codes.Unauthenticated, "Access denied")
	}

	consID := consName + strconv.Itoa(rand.Int())

	p, ok := peer.FromContext(ctx)
	if !ok {
		return fmt.Errorf("Can not read host")
	}

	ev := &Message{
		consumerID: consID,
		msgBody:
		&Event{
			Timestamp: time.Now().Unix(),
			Method: "/main.Admin/Statistics",
			Consumer: consName,
			Host: p.Addr.String(),
		},
	}

	adm.logChans.SendEvent(ev)
	adm.stat.AddByMethod("/main.Admin/Statistics")
	adm.stat.AddByConsumer(consName)

	var lastIterConsStat, lastIterMethodStat map[string]uint64

	lastIterConsStat = adm.stat.GetConsStat()
	lastIterMethodStat =  adm.stat.GetMethodStat()

	for {
		select {
		case <-ticker.C:
			ans := adm.stat.Diff(lastIterConsStat, lastIterMethodStat)
			err := inStream.Send(ans)

			lastIterConsStat = adm.stat.GetConsStat()
			lastIterMethodStat = adm.stat.GetMethodStat()

			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

type Biz struct {
	AccessList map[string][]string
	logChan    chan *Message
}

func getConsumerName(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "Can not find consumer metadata")
	}

	consumer := md.Get("consumer")
	if len(consumer) != 1 {
		return "", status.Errorf(codes.Unauthenticated, "Incorrect consumer metadata")
	}

	fmt.Println("Consumer name", consumer[0])
	return consumer[0], nil
}

func (biz *Biz) checkAccess(methodName string, consName string) bool {
	fmt.Println("Check access for user", consName, "for method", methodName)
	acc, ok := biz.AccessList[consName]
	if !ok {
		fmt.Println("Access denied")
		return false
	}

	approved := false
	for _, acs := range acc {
		if acs != "/main.Biz/*" && acs != "/main.Biz/"+methodName {
			continue
		}
		approved = true
		break
	}

	if !approved {
		fmt.Println("Access denied")
		return false
	}
	fmt.Println("Access approved")

	return true
}

func NewBiz(acl map[string][]string, lc chan *Message) *Biz {
	return &Biz{acl, lc}
}

func (biz *Biz) callMethod(ctx context.Context, nthg *Nothing, methodName string) (*Nothing, error) {
	fmt.Println("Run biz", methodName)
	consName, err := getConsumerName(ctx)
	if err != nil {
		return &Nothing{}, err
	}

	approved := biz.checkAccess(methodName, consName)

	if !approved {
		return &Nothing{}, status.Error(codes.Unauthenticated, "Access denied")
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		fmt.Println("Не могу вытащить хост")
		return &Nothing{}, nil
	}

	ans := &Message{
		consumerID: consName,
		msgBody: &Event{
			Timestamp: time.Now().Unix(),
			Consumer:  consName,
			Method:    "/main.Biz/" + methodName,
			Host:      p.Addr.String(),
		},
	}

	biz.logChan <- ans

	return &Nothing{}, nil
}

func (biz *Biz) Check(ctx context.Context, nthg *Nothing) (*Nothing, error) {
	return biz.callMethod(ctx, nthg, "Check")
}

func (biz *Biz) Add(ctx context.Context, nthg *Nothing) (*Nothing, error) {
	return biz.callMethod(ctx, nthg, "Add")
}

func (biz *Biz) Test(ctx context.Context, nthg *Nothing) (*Nothing, error) {
	return biz.callMethod(ctx, nthg, "Test")
}


