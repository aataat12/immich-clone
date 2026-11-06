package library

import("bytes";"encoding/json";"errors";"fmt";"reflect";"testing";"time";"strings";bolt "go.etcd.io/bbolt")

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

func TestOrganizationBatchRejectedPreviewCannotPublishUncommittedSearchCache(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187)
 q:=GalleryQuery{Sort:"name",Visibility:"library",Favorite:"false"};p,e:=s.QueryGalleryPage(q,"",60);if e!=nil{t.Fatal(e)};a:=p.Assets[0];target:=OrganizationTarget{ID:a.ID,Added:a.Added}
 _,e=s.PrepareOrganizationBatch(OrganizationBatchRequest{Action:"favorite_on",Query:q,Snapshot:p.Snapshot,Targets:[]OrganizationTarget{target,target}});if !errors.Is(e,ErrCurationConflict){t.Fatal(e)}
 o,_:=s.AssetOrganization(a.ID);curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:o.Token,Favorite:true})
 q.Favorite="true";fresh,e:=s.QueryGalleryPage(q,"",60);if e!=nil||fresh.Count!=1||fresh.Assets[0].ID!=a.ID{t.Fatal("uncommitted preview cache survived the next commit",fresh.Count,e)}
}

func TestOrganizationBatchPreviewRechecksCurrentTargetsBeforeAcceptance(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187);q:=GalleryQuery{Sort:"name",Visibility:"library"};p,_:=s.QueryGalleryPage(q,"",60);a:=p.Assets[0]
 s.SetTags(a.ID,[]string{"Latest custom tag"})
 in:=OrganizationBatchRequest{Action:"favorite_on",Query:q,Snapshot:p.Snapshot,Targets:[]OrganizationTarget{{ID:a.ID,Added:a.Added}}}
 b,e:=s.PrepareOrganizationBatch(in);if e!=nil{t.Fatal(e)};result:=completeBatchFixture(t,s,b);if result.Targets[0].Outcome!="saved"{t.Fatal("preview did not freeze current organization")}
 q=GalleryQuery{Sort:"name",Visibility:"library",Text:"Latest custom tag"};p,_=s.QueryGalleryPage(q,"",60);s.SetTags(a.ID,[]string{"Latest custom tag","Another client"});in.Query=q;in.Snapshot=p.Snapshot
 b,e=s.PrepareOrganizationBatch(in);if e!=nil||!b.Refreshed{t.Fatal("changed result snapshot was not explicitly refreshed for review",e)}
 s.SetHidden(a.ID,true);p,_=s.QueryGalleryPage(GalleryQuery{Sort:"name",Visibility:"library"},"",60);in.Query=GalleryQuery{Sort:"name",Visibility:"library"};in.Snapshot=p.Snapshot;if _,e=s.PrepareOrganizationBatch(in);!errors.Is(e,ErrCurationConflict){t.Fatal("out-of-scope selection accepted",e)}
}

func TestOrganizationBatchCancelKeepsCommittedWorkAndCannotResumeCanceledTargets(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187);b:=prepareBatchFixture(t,s,"favorite_on","",70);first,e:=s.ContinueOrganizationBatch(b.ID,0);if e!=nil{t.Fatal(e)}
 canceled,e:=s.CancelOrganizationBatch(b.ID,25);if e!=nil||canceled.Status!="done"{t.Fatal(e)};saved,notAttempted:=0,0;for i,target:=range canceled.Targets{switch target.Outcome{case "saved":saved++;case "canceled":notAttempted++;default:t.Fatal(target.Outcome)};o,_:=s.AssetOrganization(target.ID);if o.Favorite!=(i<25){t.Fatal("cancellation changed committed or unattempted metadata")}}
 if saved!=25||notAttempted!=45||first.Targets[25].Outcome!="pending"{t.Fatal("cancellation accounting")}
 replay,e:=s.CancelOrganizationBatch(b.ID,25);if e!=nil||!reflect.DeepEqual(replay,canceled){t.Fatal("cancellation replay",e)};replay,e=s.ContinueOrganizationBatch(b.ID,70);if e!=nil||!reflect.DeepEqual(replay,canceled){t.Fatal("canceled work resumed",e)}
 if e=s.db.View(validateCuration);e!=nil{t.Fatal(e)}
}

func TestGalleryOrganizationSnapshotIgnoresPreviewProgressAndReceiptBookkeeping(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187);q:=GalleryQuery{Sort:"name",Visibility:"library",Text:"Trip"};p,e:=s.QueryGalleryPage(q,"",60);if e!=nil||p.Next==""{t.Fatal(e)};a:=p.Assets[0]
 a.Preview="ready";a.Error="";a.Width=120;a.Height=80;s.gate.Lock();e=s.save(a);s.gate.Unlock();if e!=nil{t.Fatal(e)}
 if _,e=s.QueryGalleryPage(q,p.Next,60);e!=nil{t.Fatal("preview processing invalidated the current results",e)}
 b,e:=s.PrepareOrganizationBatch(OrganizationBatchRequest{Action:"favorite_on",Query:q,Snapshot:p.Snapshot,Targets:[]OrganizationTarget{{ID:a.ID,Added:a.Added}}});if e!=nil{t.Fatal(e)}
 if _,e=s.QueryGalleryPage(q,p.Next,60);e!=nil{t.Fatal("receipt bookkeeping invalidated the current results",e)}
 completeBatchFixture(t,s,b)
 if _,e=s.QueryGalleryPage(q,p.Next,60);!errors.Is(e,ErrGalleryChanged){t.Fatal("favorite edit did not invalidate organization snapshot",e)}
 fresh,_:=s.QueryGalleryPage(q,"",60);s.SetTags(a.ID,[]string{"Trip","Changed metadata"});if _,e=s.GalleryNeighbors(q,a.ID,fresh.Snapshot);!errors.Is(e,ErrGalleryChanged){t.Fatal("tag edit did not invalidate viewer snapshot",e)}
}

func TestOrganizationBatchChunkBudgetFailureRollsBackTargetsAndReceipt(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187);b:=prepareBatchFixture(t,s,"favorite_on","",2)
 before,_:=s.OrganizationBatch(b.ID);rawBefore,_:=json.Marshal(before);var original uint64
 first,_:=s.AssetOrganization(b.Targets[0].ID);second,_:=s.AssetOrganization(b.Targets[1].ID);first.Revision=1;first.Favorite=true;first.Token="";second.Revision=1;second.Favorite=true;second.Token="";one,_:=json.Marshal(first);two,_:=json.Marshal(second)
 s.db.Update(func(tx *bolt.Tx)error{meta:=tx.Bucket([]byte("meta"));original=galleryNumber(meta,"curation_bytes");return setGalleryNumber(meta,"curation_bytes",maxCurationBytes-uint64(len(one)+len(two)/2))})
 if _,e:=s.ContinueOrganizationBatch(b.ID,0);!errors.Is(e,ErrCurationLimit){t.Fatal("chunk did not encounter its budget boundary",e)}
 after,_:=s.OrganizationBatch(b.ID);rawAfter,_:=json.Marshal(after);if !bytes.Equal(rawBefore,rawAfter){t.Fatal("failed chunk changed its durable outcomes")};for _,target:=range b.Targets{o,_:=s.AssetOrganization(target.ID);if o.Favorite{t.Fatal("failed chunk partially changed a target")}}
 s.db.Update(func(tx *bolt.Tx)error{return setGalleryNumber(tx.Bucket([]byte("meta")),"curation_bytes",original)})
 completeBatchFixture(t,s,b);if e:=s.db.View(validateCuration);e!=nil{t.Fatal(e)}
}
