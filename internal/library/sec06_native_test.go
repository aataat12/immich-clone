package library
import (
 "bytes"
 "context"
 "immichclone/internal/media"
 "os"
 "path/filepath"
 "testing"
 "time"
)
func TestSEC06NativeMalformedMediaAndRecovery(t *testing.T) {
 provider,fixtures:=os.Getenv("IMMICH_COMPAT_PROVIDER"),os.Getenv("IMMICH_COMPAT_FIXTURES")
 if provider==""||fixtures=="" {t.Skip("requires pinned actual media provider and synthetic video")}
 p:=media.New(provider);if !p.Available(){t.Fatal("pinned provider unavailable")}
 s,_:=openTest(t);s.SetMediaPreview(p.Thumbnail)
 for _,brand:=range []string{"heic","isom","qt  "} {
  t.Run(brand,func(t *testing.T){
   bad:=[]byte{0,0,0,24,'f','t','y','p'};bad=append(bad,[]byte(brand)...);bad=append(bad,make([]byte,12)...)
   a,_,err:=s.Put("- hostile & <data>.media",bytes.NewReader(bad));if err!=nil{t.Fatal(err)}
   ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
   start:=time.Now()
   if _,err=s.ProcessOneContext(ctx);err!=nil{t.Fatal(err)}
   a,err=s.Get(a.ID);if err!=nil||a.Preview!="failed"{t.Fatal("native malformed preview",err,a.Preview)}
   if _,err=p.VideoInfo(ctx,s.Original(a.ID));err==nil {t.Fatal("malformed native probe accepted")}
   if _,_,err=p.FrameAt(ctx,s.Original(a.ID),0);err==nil{t.Fatal("malformed native frame accepted")}
   if hash,err:=fileHash(s.Original(a.ID));err!=nil||hash!=a.ID{t.Fatal("malformed original altered",err)}
   t.Logf("actual native malformed brand=%q wall=%s",brand,time.Since(start))
  })
 }
 valid,err:=os.ReadFile(filepath.Join(fixtures,"audio-first.mp4"));if err!=nil{t.Fatal(err)}
 a,_,err:=s.Put("valid video.mp4",bytes.NewReader(valid));if err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
 if _,err=s.ProcessOneContext(ctx);err!=nil{t.Fatal(err)}
 a,err=s.Get(a.ID);if err!=nil||a.Preview!="ready"{t.Fatal("native post-failure preview recovery",err,a.Preview)}
 if _,err=p.VideoInfo(ctx,s.Original(a.ID));err!=nil{t.Fatal("native post-failure probe recovery",err)}
 if _,_,err=p.FrameAt(ctx,s.Original(a.ID),0);err!=nil{t.Fatal("native post-failure frame recovery",err)}
 cancelled,stop:=context.WithCancel(context.Background());stop()
 if _,_,_,err=p.Thumbnail(cancelled,s.Original(a.ID));err==nil{t.Fatal("native cancellation ignored")}
 if _,_,err=p.FrameAt(ctx,s.Original(a.ID),0);err!=nil{t.Fatal("native cancellation recovery",err)}
 if hash,err:=fileHash(s.Original(a.ID));err!=nil||hash!=a.ID{t.Fatal("valid original altered",err)}
}
