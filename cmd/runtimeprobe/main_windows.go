package main

import (
 "context"
 "encoding/json"
 "flag"
 "fmt"
 "net"
 "net/http"
 "os"
 "path/filepath"
 "time"
 bolt "go.etcd.io/bbolt"
 "golang.org/x/sys/windows/svc"
)

type probe struct { data string }
func(p *probe) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool,uint32) {
 status<-svc.Status{State:svc.StartPending}
 db,err:=bolt.Open(filepath.Join(p.data,"probe.db"),0600,&bolt.Options{Timeout:time.Second});if err!=nil{return false,1};defer db.Close()
 err=db.Update(func(tx *bolt.Tx)error{_,e:=tx.CreateBucketIfNotExists([]byte("probe"));return e});if err!=nil{return false,2}
 mux:=http.NewServeMux()
 mux.HandleFunc("/health",func(w http.ResponseWriter,r *http.Request){
 var previous string
 err:=db.Update(func(tx *bolt.Tx)error{b:=tx.Bucket([]byte("probe"));previous=string(b.Get([]byte("state")));return b.Put([]byte("state"),[]byte("persisted"))})
 if err!=nil {http.Error(w,"database error",500);return}
 w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(map[string]string{"status":"ok","previous":previous})
 })
 listener,err:=net.Listen("tcp","127.0.0.1:18769");if err!=nil{return false,3}
 server:=&http.Server{Handler:mux,ReadHeaderTimeout:5*time.Second}
 done:=make(chan error,1);go func(){done<-server.Serve(listener)}()
 status<-svc.Status{State:svc.Running,Accepts:svc.AcceptStop|svc.AcceptShutdown}
 for {select {
 case err:=<-done: if err!=http.ErrServerClosed{return false,4};return false,0
 case request:=<-requests:
 switch request.Cmd {
 case svc.Interrogate:status<-request.CurrentStatus
 case svc.Stop,svc.Shutdown:
 status<-svc.Status{State:svc.StopPending};ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel();if server.Shutdown(ctx)!=nil{return false,5};return false,0
 }
 }}
}
func main(){data:=flag.String("data","","probe data directory");flag.Parse();if *data=="" {fmt.Fprintln(os.Stderr,"data required");os.Exit(1)};if err:=svc.Run("ImmichCloneRuntimeProbe",&probe{*data});err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}}
