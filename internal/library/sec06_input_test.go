package library



import (

 "bytes"

 "encoding/binary"

 "encoding/json"
 "os"
 "path/filepath"

 "strings"

 "testing"

 bolt "go.etcd.io/bbolt"

)



func TestSEC06MalformedExifDiscardsPartialFields(t *testing.T) {

 b:=make([]byte,38)

 copy(b,"II")

 binary.LittleEndian.PutUint16(b[2:],42)

 binary.LittleEndian.PutUint32(b[4:],8)

 binary.LittleEndian.PutUint16(b[8:],2)

 binary.LittleEndian.PutUint16(b[10:],0x112)

 binary.LittleEndian.PutUint16(b[12:],3)

 binary.LittleEndian.PutUint32(b[14:],1)

 binary.LittleEndian.PutUint16(b[18:],6)

 binary.LittleEndian.PutUint16(b[22:],0x10f)

 binary.LittleEndian.PutUint16(b[24:],2)

 binary.LittleEndian.PutUint32(b[26:],256)

 binary.LittleEndian.PutUint32(b[30:],0xffffffff)

 m:=readTIFF(b)

 if m != (PhotoMetadata{Orientation:1,Warning:"invalid_exif"}) { t.Fatalf("partial EXIF trusted: %+v",m) }

}



func TestSEC06OrganizationRejectsControlsBeforeTrimming(t *testing.T) {

 s,_:=openTest(t)

 a,_,e:=s.Put("normal.txt",strings.NewReader("synthetic original"));if e!=nil {t.Fatal(e)}

 for _,value:=range []string{"\ncat","cat\r","\tcat","\x00cat",string([]byte{0xff})} {

  if e=s.SetTags(a.ID,[]string{value});e==nil {t.Fatalf("control tag accepted: %q",value)}

  if e=s.LabelFaces([]string{strings.Repeat("a",64)},value); e!=ErrOrganizationInput {t.Fatalf("control label reached lookup: %q: %v",value,e)}

 }

 if e=s.SetTags(a.ID,[]string{"  café  ","CAFÉ","<script>"});e!=nil {t.Fatal(e)}

 var tags []string

 if e=s.db.View(func(tx *bolt.Tx)error{return readJSON(tx.Bucket([]byte(tagsBucket)),a.ID,&tags)}); e!=nil {t.Fatal(e)}

 if len(tags)!=2 || tags[1]!="café" {t.Fatalf("valid Unicode/dedup parity: %q",tags)}

}



func TestSEC06CatalogInputValidation(t *testing.T) {

 for _,tc:=range []struct{name,bucket string; value any}{

  {"asset key mismatch","assets",Asset{ID:strings.Repeat("b",64)}},

  {"asset control","assets",Asset{ID:strings.Repeat("a",64),Name:"bad\x00name"}},
  {"preview status size","assets",Asset{ID:strings.Repeat("a",64),Preview:strings.Repeat("x",17)}},
  {"preview revision","assets",Asset{ID:strings.Repeat("a",64),PreviewRevision:-1}},

  {"source traversal","assets",Asset{ID:strings.Repeat("a",64),Sources:[]string{"../escape"}}},

  {"orientation","assets",Asset{ID:strings.Repeat("a",64),Photo:PhotoMetadata{Orientation:9}}},

  {"tag count",tagsBucket,make([]string,33)},

  {"tag control",tagsBucket,[]string{"bad\n"}},

  {"label control",labelsBucket,FaceLabel{Name:"bad\n"}},

  {"analysis identity",analysisBucket,Analysis{AssetID:strings.Repeat("b",64),Status:"ready"}},

 } {

  t.Run(tc.name,func(t *testing.T){

   s,_:=openTest(t)

   if e:=s.db.Update(func(tx *bolt.Tx)error{v,e:=json.Marshal(tc.value);if e!=nil{return e};return tx.Bucket([]byte(tc.bucket)).Put([]byte(strings.Repeat("a",64)),v)});e!=nil{t.Fatal(e)}

   if e:=s.db.View(validateCatalogInput);e==nil{t.Fatal("malformed catalog accepted")}

  })

 }

 s,_:=openTest(t)

 a,_,e:=s.PutWithPath("café <safe>.txt","Trip/café <safe>.txt",bytes.NewBufferString("synthetic valid media"));if e!=nil{t.Fatal(e)}

 if e=s.SetTags(a.ID,[]string{"café","<script>"});e!=nil{t.Fatal(e)}

 if e=s.db.View(validateCatalogInput);e!=nil{t.Fatalf("valid display-data parity: %v",e)}

 if e=s.db.Update(func(tx *bolt.Tx)error{return tx.Bucket([]byte(tagsBucket)).Put([]byte(a.ID),bytes.Repeat([]byte(" "),(1<<20)+1))});e!=nil{t.Fatal(e)}

 if e=s.db.View(validateCatalogInput);e==nil{t.Fatal("oversized record accepted")}

}



func TestSEC06CatalogMaximumSourceAliases(t *testing.T) {

 s,_:=openTest(t)

 id:=strings.Repeat("a",64)

 a:=Asset{ID:id, Name:"normal.bin", Size:1}

 for i:=0;i<4096;i++ {a.Sources=append(a.Sources,strings.Repeat("x",1010)+"/"+strings.Repeat("y",10))}

 if e:=s.db.Update(func(tx *bolt.Tx)error {v,e:=json.Marshal(a);if e!=nil{return e};return tx.Bucket([]byte("assets")).Put([]byte(id),v)});e!=nil{t.Fatal(e)}

 if e:=s.db.View(validateCatalogInput);e!=nil{t.Fatalf("documented maximum aliases refused: %v",e)}

}



func TestSEC06CatalogImportParityAndRejection(t *testing.T) {

 s,_:=openTest(t)

 m,e:=s.CreateImport([]ImportFile{{Name:"café.txt",Relative:"Trip/café.txt",Size:123}});if e!=nil{t.Fatal(e)}

 if e=s.db.View(validateCatalogInput);e!=nil{t.Fatalf("valid import rejected: %v",e)}

 if e=s.db.Update(func(tx *bolt.Tx)error {b:=tx.Bucket([]byte(importBucket)).Bucket([]byte(m.ID));var entry ImportEntry;if e:=readJSON(b,entryKey(0),&entry);e!=nil{return e};entry.Index=99;return writeJSON(b,entryKey(0),entry)});e!=nil{t.Fatal(e)}

 if e=s.db.View(validateCatalogInput);e==nil{t.Fatal("invalid import index accepted")}

}



func TestSEC06CatalogRejectsInvalidUTF8AndNestedAuthoritativeBuckets(t *testing.T) {

 for _,nested:=range []bool{false,true} {

  s,_:=openTest(t)

  if e:=s.db.Update(func(tx *bolt.Tx)error {

   b:=tx.Bucket([]byte("assets"));k:=[]byte(strings.Repeat("a",64))

   if nested {_,e:=b.CreateBucket(k);return e}

   return b.Put(k,[]byte("{\"id\":\""+string(k)+"\",\"name\":\""+string([]byte{0xff})+"\"}"))

  });e!=nil{t.Fatal(e)}

  if e:=s.db.View(validateCatalogInput);e==nil{t.Fatal("invalid catalog accepted")}

 }

}


func TestSEC06MalformedMetadataPreservesOriginalAndPreview(t *testing.T) {
 s,_:=openTest(t)
 data:=exifJPEG(t,6)
 // TIFF starts after SOI, APP1 marker/length and Exif signature.
 binary.LittleEndian.PutUint32(data[12+22+8:],0xffffffff)
 a,_,e:=s.Put("malformed-exif.jpg",bytes.NewReader(data));if e!=nil{t.Fatal(e)}
 if _,e=s.ProcessOne();e!=nil{t.Fatal(e)}
 a,e=s.Get(a.ID);if e!=nil{t.Fatal(e)}
 if a.Preview!="ready" || a.Photo.Warning!="invalid_exif" || a.Photo.Orientation!=1 || a.Width!=20 || a.Height!=10 {t.Fatalf("malformed metadata damaged valid media: %+v",a)}
 original,e:=os.ReadFile(s.Original(a.ID));if e!=nil || !bytes.Equal(data,original){t.Fatal("original changed",e)}
}

func TestSEC06CatalogEscapedMaximumAliases(t *testing.T) {
 s,_:=openTest(t)
 a:=Asset{ID:strings.Repeat("a",64),Name:"safe",Size:1}
 // encoding/json expands each '<' to six bytes; the documented alias bound
 // must remain usable even for strings requiring JSON/HTML escaping.
 for i:=0;i<4096;i++ {a.Sources=append(a.Sources,strings.Repeat("<",1024))}
 if e:=s.db.Update(func(tx *bolt.Tx)error {b,e:=json.Marshal(a);if e!=nil{return e};return tx.Bucket([]byte("assets")).Put([]byte(a.ID),b)});e!=nil {t.Fatal(e)}
 if e:=s.db.View(validateCatalogInput);e!=nil{t.Fatal("escaped aliases rejected",e)}
}

func TestSEC06CatalogOpenRefusesWithoutCatalogMutation(t *testing.T) {
 s,root:=openTest(t)
 a,_,e:=s.Put("safe.txt",bytes.NewBufferString("synthetic authoritative original"));if e!=nil{t.Fatal(e)}
 if e=s.db.Update(func(tx *bolt.Tx)error{return writeJSON(tx.Bucket([]byte(tagsBucket)),a.ID,[]string{"bad\nlabel"})});e!=nil{t.Fatal(e)}
 if e=s.Close();e!=nil{t.Fatal(e)}
 catalog:=filepath.Join(root,"catalog.db")
 before,e:=os.ReadFile(catalog);if e!=nil{t.Fatal(e)}
 reopened,e:=Open(root)
 if e==nil {reopened.Close();t.Fatal("Open accepted malformed authoritative catalog")}
 after,e:=os.ReadFile(catalog);if e!=nil{t.Fatal(e)}
 if !bytes.Equal(before,after) {t.Fatal("failed catalog validation changed catalog bytes")}
 original,e:=os.ReadFile(filepath.Join(root,"originals",a.ID))
 if e!=nil || string(original)!="synthetic authoritative original" {t.Fatal("failed validation changed original",e)}
}

func TestSEC06CatalogVideoSampleValidation(t *testing.T) {
 v:=&VideoProgress{Version:1,Policy:videoPolicy,DurationUS:100,Targets:[]int64{0},Next:1,Samples:[]VideoSample{{Provider:"safe",PositionUS:0,Width:1,Height:1,Objects:[]string{"cat"}}}}
 if !validCatalogVideo(v) {t.Fatal("valid video sample refused")}
 v.Samples[0].Objects=[]string{"bad\n"}
 if validCatalogVideo(v) {t.Fatal("video sample control tag accepted")}
 v.Samples[0].Objects=[]string{"cat"};v.Samples[0].Provider=string([]byte{255})
 if validCatalogVideo(v) {t.Fatal("invalid video provider encoding accepted")}
}
