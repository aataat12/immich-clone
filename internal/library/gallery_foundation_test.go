package library

import (
 "encoding/json"
 "errors"
 "fmt"
 bolt "go.etcd.io/bbolt"
 "testing"
 "time"
)

// Disposable deterministic catalogs, no personal data or decoder dependency.
// Same-name/time ties, visible/hidden, unknown MIME, missing capture, Unicode,
// manual/automatic assignments, suggestions and dismissed AI tags are deliberate.
func seedGalleryFoundation(t *testing.T,s *Store,n int) {
 t.Helper()
 e:=s.db.Update(func(tx *bolt.Tx)error {
  for i:=1;i<=n;i++ {
   id:=fmt.Sprintf("%064x",i)
   a:=Asset{ID:id,Name:fmt.Sprintf("Family-%03d.jpg",i),MIME:"image/jpeg",Size:1,Added:time.Date(2024,11,3,5,i%60,0,0,time.UTC),Preview:"failed",PreviewRevision:3,Hidden:i%7==0}
   if i%2==0 {a.Photo.Captured="2024-02-29T12:00:00"}
   if i%3==0 {a.MIME="video/mp4"}
   if i%5==0 {a.MIME="application/octet-stream";a.Name="unknown.jpg"}
   if i==1 {a.Name="Résumé <family>.jpg";a.Sources=[]string{"Trip/Family.jpg","Other/Family.jpg"}}
   if e:=saveAsset(tx,a);e!=nil{return e}
   tags:=[]string{"Trip"}
   if i%2==0 {tags=append(tags,"Even")}
   if e:=writeJSON(tx.Bucket([]byte(tagsBucket)),id,tags);e!=nil{return e}
   if e:=writeJSON(tx.Bucket([]byte(analysisBucket)),id,Analysis{AssetID:id,Status:"ready",Provider:"synthetic",Objects:[]string{"cat","blocked"}});e!=nil{return e}
   if e:=writeJSON(tx.Bucket([]byte("dismissed_ai_tags_v1")),id,[]string{"blocked"});e!=nil{return e}
   names:=[]string{"Alice"}
   if i%2==0 {names=append(names,"Bob")}
   if i%3==0 {names=[]string{"Alicia"}}
   for j,name:=range names {
    faceID:=fmt.Sprintf("%064x",100000+i*10+j)
    if e:=writeJSON(tx.Bucket([]byte(facesBucket)),faceID,StoredFace{ID:faceID,AssetID:id,Box:[]float64{.1,.1,.2,.2},Provider:"synthetic"});e!=nil{return e}
    label:=FaceLabel{Name:name,Source:"manual"}
    if i%4==0 {label.Source="automatic";label.Policy=automaticNamePolicyID;label.Score=.99;if e:=requireAutomaticCatalog(tx);e!=nil{return e}}
    if i%11==0 {label.Dismissed=true}
    if e:=writeJSON(tx.Bucket([]byte(labelsBucket)),faceID,label);e!=nil{return e}
   }
  }
  return nil
 })
 if e!=nil {t.Fatal(e)}
}

func TestGalleryFoundationRepresentativeCatalogs(t *testing.T) {
 for _,size:=range []int{0,1,187} {
  t.Run(fmt.Sprint(size),func(t *testing.T){s,_:=openTest(t);seedGalleryFoundation(t,s,size)
   queries:=[]GalleryQuery{
    {},{Text:"TRIP"},{Filename:"unknown",Media:"photo"},{People:GalleryLabels{All:[]string{"Alice","Bob"}}},
    {People:GalleryLabels{Any:[]string{"Alice","Alicia"},Exclude:[]string{"Bob"}}},
    {Tags:GalleryLabels{All:[]string{"trip","cat"},Exclude:[]string{"even"}}},{Tags:GalleryLabels{Any:[]string{"blocked"}}},
    {DateBasis:"capture",From:"2024-02-29T00:00:00",Until:"2024-03-01T00:00:00"},{NoCapture:true},
    {DateBasis:"upload",From:"2024-11-03T01:00:00-04:00",Until:"2024-11-03T01:00:00-05:00"},{Text:"résumé"},
   }
   for _,visibility:=range []string{"library","hidden"} {for _,order:=range []string{"capture","upload","name"} {for _,q:=range queries {
    q.Sort,q.Visibility=order,visibility
    after:="";var all []Asset;var snapshot string;var count uint64
    for {
     p,e:=s.QueryGalleryPage(q,after,17);if e!=nil{t.Fatal(q,e)}
     if snapshot!=""&&snapshot!=p.Snapshot {t.Fatal("different page snapshots")}
     snapshot,count=p.Snapshot,p.Count;all=append(all,p.Assets...)
     if p.Next==""{break};after=p.Next
    }
    if uint64(len(all))!=count {t.Fatal("count/paging diverged",q,count,len(all))}
    seen:=map[string]bool{}
    for i,a:=range all {
     if seen[a.ID] {t.Fatal("duplicate tie row")};seen[a.ID]=true
     n,e:=s.GalleryNeighbors(q,a.ID,snapshot);if e!=nil||n.Count!=count||n.Current.ID!=a.ID {t.Fatal("neighbors/current",e)}
     if (i==0)!=(n.Previous==nil)||(i==len(all)-1)!=(n.Next==nil) {t.Fatal("endpoints")}
     if i>0&&n.Previous.ID!=all[i-1].ID {t.Fatal("previous mismatch")}
     if i+1<len(all)&&n.Next.ID!=all[i+1].ID {t.Fatal("next mismatch")}
    }
   }}}
  })
 }
}

func TestGalleryFoundationExactSemanticsAndInvalidation(t *testing.T) {
 s,root:=openTest(t);seedGalleryFoundation(t,s,20)
 q:=GalleryQuery{Sort:"name",Visibility:"library",People:GalleryLabels{All:[]string{" Alice ","BOB","alice"}},Tags:GalleryLabels{Any:[]string{"Trip"}}}
 p,e:=s.QueryGalleryPage(q,"",1);if e!=nil||p.Count!=6||len(p.Assets)!=1||p.Next==""{t.Fatal(p,e)}
 for _,check:=range []struct{q GalleryQuery;want uint64}{
  {GalleryQuery{Media:"photo"},9},{GalleryQuery{Media:"video"},5},
  {GalleryQuery{NoCapture:true},9},{GalleryQuery{People:GalleryLabels{All:[]string{"Alice"}}},11},
  {GalleryQuery{Tags:GalleryLabels{Any:[]string{"blocked"}}},0},
  {GalleryQuery{DateBasis:"capture",From:"2024-02-29T00:00:00",Until:"2024-03-01T00:00:00"},9},
  {GalleryQuery{DateBasis:"capture",Until:"2024-02-29T12:00:00"},0},
  {GalleryQuery{DateBasis:"upload",From:"2024-11-03T01:00:00-04:00",Until:"2024-11-03T01:00:00-05:00"},18},
 } {check.q.Sort="name";check.q.Visibility="library";p,e:=s.QueryGalleryPage(check.q,"",100);if e!=nil||p.Count!=check.want{t.Fatal(check.q,p.Count,check.want,e)}}
 canonical:=q;canonical.People.All=[]string{"bob","alice"}
 if _,e=s.QueryGalleryPage(canonical,p.Next,1);e!=nil{t.Fatal("equivalent queries",e)}
 changed:=q;changed.Media="photo"
 if _,e=s.QueryGalleryPage(changed,p.Next,1);!errors.Is(e,ErrGalleryInput){t.Fatal("cross-query cursor",e)}
 if _,e=s.GalleryNeighbors(changed,p.Assets[0].ID,p.Snapshot);!errors.Is(e,ErrGalleryInput){t.Fatal("cross-query viewer",e)}
 if e=s.SetTags(p.Assets[0].ID,[]string{"changed"});e!=nil{t.Fatal(e)}
 if _,e=s.QueryGalleryPage(q,p.Next,1);!errors.Is(e,ErrGalleryChanged){t.Fatal("stale filtered page",e)}
 if _,e=s.GalleryNeighbors(q,p.Assets[0].ID,p.Snapshot);!errors.Is(e,ErrGalleryChanged){t.Fatal("stale viewer",e)}
 // Unfiltered preview-only writes preserve the fast-path ordering snapshot.
 unfiltered:=GalleryQuery{Sort:"name",Visibility:"library"}
 p,e=s.QueryGalleryPage(unfiltered,"",1);if e!=nil{t.Fatal(e)}
 a:=p.Assets[0];a.Error="preview changed";if e=s.save(a);e!=nil{t.Fatal(e)}
 if _,e=s.QueryGalleryPage(unfiltered,p.Next,1);e!=nil{t.Fatal("unfiltered preview invalidation",e)}
 if e=s.SetHidden(a.ID,true);e!=nil{t.Fatal(e)}
 if _,e=s.GalleryNeighbors(unfiltered,a.ID,p.Snapshot);!errors.Is(e,ErrGalleryChanged){t.Fatal("hidden anchor",e)}
 fresh,e:=s.QueryGalleryPage(q,"",1);if e!=nil{t.Fatal(e)}
 s.Close();again,e:=Open(root);if e!=nil{t.Fatal(e)};defer again.Close()
 if _,e=again.QueryGalleryPage(q,fresh.Next,1);!errors.Is(e,ErrGalleryChanged){t.Fatal("restart epoch",e)}
 // No new schema or data is needed by a browsing-only feature.
 if e=again.db.View(func(tx *bolt.Tx)error{if string(tx.Bucket([]byte("meta")).Get([]byte("schema")))!="2"{t.Fatal("catalog version changed")};return nil});e!=nil{t.Fatal(e)}
}

func TestGalleryFoundationValidation(t *testing.T) {
 valid:=GalleryQuery{Sort:"capture",Visibility:"library"}
 for _,change:=range []func(*GalleryQuery){
  func(q *GalleryQuery){q.Media="unknown"},func(q *GalleryQuery){q.Text=string([]byte{255})},
  func(q *GalleryQuery){q.People.All=[]string{""}},func(q *GalleryQuery){q.From="2024-01-01T00:00:00"},
  func(q *GalleryQuery){q.DateBasis="capture";q.From="2024-02-30T00:00:00"},
  func(q *GalleryQuery){q.DateBasis="upload";q.From="2024-01-01T00:00:00"},
  func(q *GalleryQuery){q.DateBasis="capture";q.From="2025-01-01T00:00:00";q.Until="2024-01-01T00:00:00"},
  func(q *GalleryQuery){q.NoCapture=true;q.DateBasis="capture"},
 } {q:=valid;change(&q);if _,e:=q.Normalize();!errors.Is(e,ErrGalleryInput){b,_:=json.Marshal(q);t.Fatal(string(b),e)}}
}

func TestGalleryFoundationFilteredScale(t *testing.T) {
 s,_:=openTest(t);seedGallery(t,s,50000)
 q:=GalleryQuery{Sort:"name",Visibility:"library",Filename:"Synthetic-049",Media:"photo"}
 start:=time.Now();p,e:=s.QueryGalleryPage(q,"",60);if e!=nil||p.Count!=858||len(p.Assets)!=60||p.Next==""{t.Fatal(p.Count,e)}
 cold:=time.Since(start);start=time.Now()
 allocs:=testing.AllocsPerRun(3,func(){if _,e:=s.QueryGalleryPage(q,p.Next,60);e!=nil{panic(e)}})
 if allocs>5000 {t.Fatal("warm page builds full catalog",allocs)}
 t.Logf("synthetic 50k metadata rows: cold filtered query %s, three warm pages %s, warm page allocations %.0f; no media ingestion or speed guarantee",cold,time.Since(start),allocs)
}
