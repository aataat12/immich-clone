package library

import (
 "errors"
 "testing"
 "time"
)

func TestGalleryAnchorPagesAndReverseCursors(t *testing.T){
 s,_:=openTest(t);seedGalleryFoundation(t,s,187)
 for _,order:=range []string{"capture","upload","name"}{for _,visibility:=range []string{"library","hidden"}{for _,text:=range []string{"","Alice","Trip"}{
  q:=GalleryQuery{Sort:order,Visibility:visibility,Text:text};var pages []GalleryPage;after:=""
  for {p,e:=s.QueryGalleryPage(q,after,17);if e!=nil{t.Fatal(e)};pages=append(pages,p);if p.Next==""{break};after=p.Next}
  for i,p:=range pages {for _,a:=range p.Assets {
   anchor,e:=s.GalleryAnchorPage(q,a.ID,p.Snapshot,17);if e!=nil||anchor.Number!=uint64(i+1)||len(anchor.Page.Assets)!=len(p.Assets){t.Fatal("anchor",i,e)}
   for j,a:=range p.Assets{if anchor.Page.Assets[j].ID!=a.ID{t.Fatal("wrong anchored page")}}
   if anchor.Page.HasPrevious!=(i>0){t.Fatal("previous state")}
   if i>0{previous,e:=s.QueryGalleryPage(q,anchor.Page.Previous,17);if e!=nil||previous.Assets[0].ID!=pages[i-1].Assets[0].ID{t.Fatal("direct jump previous",e)}}
  }}
 }}}
 q:=GalleryQuery{Sort:"name",Visibility:"library",Text:"trip"};p,_:=s.QueryGalleryPage(q,"",1)
 if e:=s.SetTags(p.Assets[0].ID,[]string{"changed"});e!=nil{t.Fatal(e)}
 if _,e:=s.GalleryAnchorPage(q,p.Assets[0].ID,p.Snapshot,60);!errors.Is(e,ErrGalleryChanged){t.Fatal("stale anchor",e)}
 if _,e:=s.GalleryAnchorPage(q,p.Assets[0].ID,"",60);!errors.Is(e,ErrGalleryChanged){t.Fatal("ineligible anchor",e)}
}

func TestGalleryAnchorScale(t *testing.T){s,_:=openTest(t);seedGallery(t,s,50000);q:=GalleryQuery{Sort:"name",Visibility:"library"};p,_:=s.QueryGalleryPage(q,"",1);id:=p.Assets[0].ID;start:=time.Now();a,e:=s.GalleryAnchorPage(q,id,p.Snapshot,60);if e!=nil||len(a.Page.Assets)>60{t.Fatal(e)};t.Logf("50k metadata anchor, bounded 60-row result: %s",time.Since(start));}
