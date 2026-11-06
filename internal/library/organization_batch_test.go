package library

import("bytes";"errors";"fmt";"reflect";"testing";"time";"strings";bolt "go.etcd.io/bbolt")

func prepareBatchFixture(t *testing.T,s *Store,action,album string,limit int)OrganizationBatch{t.Helper();q:=GalleryQuery{Sort:"name",Visibility:"library"};p,e:=s.QueryGalleryPage(q,"",100);if e!=nil{t.Fatal(e)};targets:=[]OrganizationTarget{};for _,a:=range p.Assets{if len(targets)==limit{break};targets=append(targets,OrganizationTarget{ID:a.ID,Added:a.Added})};b,e:=s.PrepareOrganizationBatch(OrganizationBatchRequest{Action:action,AlbumID:album,Query:q,Snapshot:p.Snapshot,Targets:targets});if e!=nil{t.Fatal(e)};return b}
func completeBatchFixture(t *testing.T,s *Store,b OrganizationBatch)OrganizationBatch{t.Helper();for b.Status!="done"{step:=0;for _,x:=range b.Targets{if x.Outcome!="pending"{step++}};var e error;b,e=s.ContinueOrganizationBatch(b.ID,step);if e!=nil{t.Fatal(e)}};return b}
func TestOrganizationBatchReceiptsRestartAndConflicts(t *testing.T){
 s,root:=openTest(t);seedGalleryFoundation(t,s,187)
 b:=prepareBatchFixture(t,s,"favorite_on","",70)
 if b.Status!="paused"||len(b.Targets)!=70{t.Fatal("preview")}
 first,e:=s.ContinueOrganizationBatch(b.ID,0);if e!=nil{t.Fatal(e)}
 n:=0;for _,x:=range first.Targets{if x.Outcome=="saved"{n++}};if n!=25{t.Fatal("chunk",n)}
 // Discard that response, close, then replay its exact continuation.
 s.Close();s,e=Open(root);if e!=nil{t.Fatal(e)};defer s.Close()
 again,e:=s.ContinueOrganizationBatch(b.ID,0);if e!=nil||!reflect.DeepEqual(first,again){t.Fatal("lost response replay progressed",e)}
 pending:=again.Targets[25];if e=s.SetTags(pending.ID,[]string{"Other tab"});e!=nil{t.Fatal(e)}
 result:=completeBatchFixture(t,s,again);if result.Targets[25].Outcome!="conflict"{t.Fatal("stale target overwritten")}
 o,_:=s.AssetOrganization(pending.ID);if o.Favorite{t.Fatal("conflict overwritten")}
 q:=GalleryQuery{Sort:"name",Visibility:"library",Favorite:"true"};p,e:=s.QueryGalleryPage(q,"",100);if e!=nil||p.Count!=69{t.Fatal("favorite count",p.Count,e)}
 if e=s.db.View(validateCuration);e!=nil{t.Fatal("receipt validation",e)}
 // Expired/unknown IDs cannot create or replay a mutation.
 s.db.Update(func(tx *bolt.Tx)error{old,_:=readOrganizationBatch(tx,b.ID);old.Created=time.Now().Add(-31*24*time.Hour);return curationWrite(tx,organizationBatchesBucket,b.ID,old)})
 if _,e=s.ContinueOrganizationBatch(b.ID,0);!errors.Is(e,ErrBatchExpired){t.Fatal("expired",e)}
 if _,e=s.ContinueOrganizationBatch(fmt.Sprintf("%064x",99999),0);!errors.Is(e,ErrBatchExpired){t.Fatal("unknown",e)}
}
func TestOrganizationBatchTagsAlbumsManualOrderAndRollback(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187);album:=fmt.Sprintf("%064x",88001);other:=fmt.Sprintf("%064x",88002)
 curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:album,Title:"Family <trip>"});curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:other,Title:"Empty"})
 b:=completeBatchFixture(t,s,prepareBatchFixture(t,s,"add_album",album,70));for _,x:=range b.Targets{if x.Outcome!="saved"{t.Fatal(x.Outcome)}}
 duplicate:=completeBatchFixture(t,s,prepareBatchFixture(t,s,"add_album",album,70));for _,x:=range duplicate.Targets{if x.Outcome!="skipped"{t.Fatal("duplicate")}}
 q:=GalleryQuery{Sort:"manual",Visibility:"library",Album:album};p,e:=s.QueryGalleryPage(q,"",60);if e!=nil||p.Count!=70||len(p.Assets)!=60{t.Fatal("manual page",p.Count,e)}
 anchor,e:=s.GalleryAnchorPage(q,b.Targets[60].ID,p.Snapshot,60);if e!=nil||anchor.Number!=2||!anchor.Page.HasPrevious{t.Fatal("manual anchor",e)}
 n,e:=s.GalleryNeighbors(q,b.Targets[60].ID,p.Snapshot);if e!=nil||n.Previous.ID!=b.Targets[59].ID{t.Fatal("manual neighbors",e)}
 back,e:=s.QueryGalleryPage(q,anchor.Page.Previous,60);if e!=nil||back.Assets[0].ID!=p.Assets[0].ID{t.Fatal("manual reverse",e)}
 a,_:=s.AlbumPage(album,"",100);curationEdit(t,s,CurationEdit{Action:"move_member",AlbumID:album,Expected:a.Album.Token,AssetID:b.Targets[60].ID,OtherID:b.Targets[59].ID})
 a,_=s.AlbumPage(album,"",100);curationEdit(t,s,CurationEdit{Action:"set_cover",AlbumID:album,Expected:a.Album.Token,AssetID:b.Targets[0].ID});s.SetHidden(b.Targets[0].ID,true)
 rows,_,e:=s.AlbumSummaries("",100);if e!=nil{t.Fatal(e)};for _,row:=range rows{if row.Album.ID==album&&(row.Visible!=69||row.Hidden!=1||row.Cover==nil||row.Cover.Hidden||row.Cover.ID==b.Targets[0].ID){t.Fatal("cover fallback")}}
 // A failed transaction rolls back both target mutation and its receipt state.
 before,_:=s.OrganizationExport();sentinel:=errors.New("synthetic write failure");e=s.db.Update(func(tx *bolt.Tx)error{asset,_:=curationAsset(tx,b.Targets[1].ID);if _,e:=applyOrganizationTarget(tx,OrganizationBatch{Action:"favorite_on"},asset);e!=nil{return e};return sentinel});if !errors.Is(e,sentinel){t.Fatal(e)};after,_:=s.OrganizationExport();raw1,_:=jsonCuration(before);raw2,_:=jsonCuration(after);if !bytes.Equal(raw1,raw2){t.Fatal("partial metadata transaction")}
 a,_=s.AlbumPage(album,"",100);curationEdit(t,s,CurationEdit{Action:"delete_album",AlbumID:album,Expected:a.Album.Token});if _,e=s.Get(b.Targets[0].ID);e!=nil{t.Fatal("album deletion removed media")};empty,_:=s.AlbumPage(other,"",100);if empty.Count!=0{t.Fatal("other album changed")};if e=s.db.View(validateCuration);e!=nil{t.Fatal(e)}
}

func TestOrganizationBatchBoundsTagsAndReceiptRetirement(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187)
 q:=GalleryQuery{Sort:"name",Visibility:"library",Text:"Trip"};p,e:=s.QueryGalleryPage(q,"",60);if e!=nil{t.Fatal(e)}
 target:=OrganizationTarget{ID:p.Assets[0].ID,Added:p.Assets[0].Added}
 in:=OrganizationBatchRequest{Action:"add_tag",Tag:"Custom <family>",Query:q,Snapshot:p.Snapshot,Targets:[]OrganizationTarget{target}}
 before:=map[string][]byte{};s.db.View(func(tx *bolt.Tx)error{for _,name:=range []string{analysisBucket,"dismissed_ai_tags_v1",labelsBucket}{b:=tx.Bucket([]byte(name));if b!=nil{b.ForEach(func(k,v []byte)error{before[name+string(k)]=append([]byte(nil),v...);return nil})}};return nil})
 b,e:=s.PrepareOrganizationBatch(in);if e!=nil{t.Fatal("filtered snapshot preview",e)};b=completeBatchFixture(t,s,b);if b.Targets[0].Outcome!="saved"{t.Fatal("tag add")}
 p,_=s.QueryGalleryPage(q,"",60);in.Snapshot=p.Snapshot;in.Action="remove_tag";b,e=s.PrepareOrganizationBatch(in);if e!=nil{t.Fatal(e)};completeBatchFixture(t,s,b)
 s.db.View(func(tx *bolt.Tx)error{for key,want:=range before{for _,name:=range []string{analysisBucket,"dismissed_ai_tags_v1",labelsBucket}{if strings.HasPrefix(key,name)&&!bytes.Equal(want,tx.Bucket([]byte(name)).Get([]byte(strings.TrimPrefix(key,name)))){t.Fatal("recognition/correction provenance changed")}}};return nil})
 p,_=s.QueryGalleryPage(q,"",60);in.Snapshot=p.Snapshot;in.Targets=make([]OrganizationTarget,501);if _,e=s.PrepareOrganizationBatch(in);!errors.Is(e,ErrCurationInput){t.Fatal("selection bound",e)}
 in.Targets=[]OrganizationTarget{target,target};if _,e=s.PrepareOrganizationBatch(in);!errors.Is(e,ErrCurationConflict){t.Fatal("duplicate targets",e)}
 // Filled unfinished history refuses more work without evicting pending receipts.
 for i:=0;i<maxOrganizationBatches;i++{rows,_:=s.OrganizationBatches();if len(rows)==maxOrganizationBatches{break};prepareBatchFixture(t,s,"favorite_on","",1)}
 rows,_:=s.OrganizationBatches();for _,r:=range rows{if r.Status=="done"{s.db.Update(func(tx *bolt.Tx)error{old,_:=readOrganizationBatch(tx,r.ID);old.Status="paused";for i:=range old.Targets{old.Targets[i].Outcome="pending"};return curationWrite(tx,organizationBatchesBucket,r.ID,old)})}}
 p,_=s.QueryGalleryPage(q,"",60);in.Action="favorite_on";in.Tag="";in.Snapshot=p.Snapshot;in.Targets=[]OrganizationTarget{target};if _,e=s.PrepareOrganizationBatch(in);!errors.Is(e,ErrCurationLimit){t.Fatal("unfinished history was evicted",e)}
 rows,_=s.OrganizationBatches();old,e:=s.OrganizationBatch(rows[0].ID);if e!=nil{t.Fatal(e)};completeBatchFixture(t,s,old);p,_=s.QueryGalleryPage(q,"",60);in.Snapshot=p.Snapshot;if _,e=s.PrepareOrganizationBatch(in);e!=nil{t.Fatal("completed retirement",e)};if _,e=s.OrganizationBatch(old.ID);!errors.Is(e,ErrBatchExpired){t.Fatal("retired ID replay",e)}
 if e=s.db.View(validateCuration);e!=nil{t.Fatal("history byte accounting",e)}
}

func TestOrganizationBatchAlbumRecreationCannotReceiveOldEdits(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187);id:=fmt.Sprintf("%064x",89001)
 old:=curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:id,Title:"Same title"}).(Album)
 batch:=prepareBatchFixture(t,s,"add_album",id,1)
 curationEdit(t,s,CurationEdit{Action:"delete_album",AlbumID:id,Expected:old.Token})
 fresh:=curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:id,Title:"Same title"}).(Album)
 if old.Incarnation==fresh.Incarnation||old.Token==fresh.Token{t.Fatal("reused album identity accepted old token")}
 if _,e:=s.EditCuration(CurationEdit{Action:"update_album",AlbumID:id,Expected:old.Token,Title:"Old draft"});!errors.Is(e,ErrCurationConflict){t.Fatal("old album draft accepted",e)}
 result:=completeBatchFixture(t,s,batch);if result.Targets[0].Outcome!="conflict"{t.Fatal("old batch edited a recreated album")};p,_:=s.AlbumPage(id,"",60);if p.Count!=0{t.Fatal("recreated album mutated")}
}
