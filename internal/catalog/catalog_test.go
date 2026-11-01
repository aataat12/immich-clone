package catalog

import (
 "bytes"
 "image"
 "image/color"
 "image/jpeg"
 "image/png"
 "path/filepath"
 "testing"
 "time"
 bolt "go.etcd.io/bbolt"
)

func TestDatabaseRollbackRestartAndMedia(t *testing.T) {
 path := filepath.Join(t.TempDir(), "probe.db")
 db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
 if err != nil { t.Fatal(err) }
 err = db.Update(func(tx *bolt.Tx) error { b,e:=tx.CreateBucketIfNotExists([]byte("probe"));if e!=nil{return e};return b.Put([]byte("original"),[]byte("preserved")) })
 if err!=nil {t.Fatal(err)}
 tx,err:=db.Begin(true);if err!=nil {t.Fatal(err)}
 if err=tx.Bucket([]byte("probe")).Put([]byte("original"), []byte("uncommitted"));err!=nil {t.Fatal(err)}
 if err=tx.Rollback();err!=nil {t.Fatal(err)}
 if err=db.Close();err!=nil {t.Fatal(err)}
 db,err=bolt.Open(path,0600,nil);if err!=nil {t.Fatal(err)};defer db.Close()
 err=db.View(func(tx *bolt.Tx) error {if string(tx.Bucket([]byte("probe")).Get([]byte("original")))!="preserved" {t.Fatal("rollback/reopen lost value")};return nil});if err!=nil {t.Fatal(err)}
 img:=image.NewRGBA(image.Rect(0,0,64,48));img.Set(4,4,color.RGBA{255,80,0,255})
 for _,format:=range []string{"png","jpeg"} {
 var b bytes.Buffer
 if format=="png" {err=png.Encode(&b,img)} else {err=jpeg.Encode(&b,img,nil)}
 if err!=nil {t.Fatal(err)}
 decoded,_,e:=image.Decode(&b);if e!=nil||decoded.Bounds().Dx()!=64 {t.Fatalf("%s decode: %v",format,e)}
 }
}
