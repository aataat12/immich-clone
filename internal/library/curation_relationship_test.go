package library

import (
 "bytes"
 "context"
 "errors"
 "fmt"
 "testing"
)

func TestCurationManyAlbumsAndReimportGuard(t *testing.T){s,_:=openTest(t);data:=fixture(t);a,_,e:=s.Put("first.png",bytes.NewReader(data));if e!=nil{t.Fatal(e)};old,_:=s.AssetOrganization(a.ID);curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:old.Token,Favorite:true})
 ids:=[]string{fmt.Sprintf("%064x",91001),fmt.Sprintf("%064x",91002),fmt.Sprintf("%064x",91003)}
 for _,id:=range ids{curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:id,Title:"Synthetic album"})}
 for _,id:=range ids[:2]{p,_:=s.AlbumPage(id,"",10);v:=curationEdit(t,s,CurationEdit{Action:"add_member",AlbumID:id,AssetID:a.ID,Added:a.Added,Expected:p.Album.Token}).(Album);curationEdit(t,s,CurationEdit{Action:"add_member",AlbumID:id,AssetID:a.ID,Added:a.Added,Expected:v.Token});p,_=s.AlbumPage(id,"",10);if p.Count!=1{t.Fatal("duplicate membership")}}
 p,_:=s.AlbumPage(ids[0],"",10);curationEdit(t,s,CurationEdit{Action:"delete_album",AlbumID:ids[0],Expected:p.Album.Token});p,_=s.AlbumPage(ids[1],"",10);if p.Count!=1{t.Fatal("album deletion removed another membership")};if e=s.Verify();e!=nil{t.Fatal(e)}
 if e=s.SetHidden(a.ID,true);e!=nil{t.Fatal(e)};if _,e=s.DeleteHidden(context.Background(),a.ID,a.ID,a.Added);e!=nil{t.Fatal(e)}
 again,_,e:=s.Put("reimport.png",bytes.NewReader(data));if e!=nil||again.ID!=a.ID||again.Added.Equal(a.Added){t.Fatal("reimport fixture",e)}
 if _,e=s.EditCuration(CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:old.Token,Favorite:true});!errors.Is(e,ErrCurationConflict){t.Fatal("stale incarnation",e)}
 m,_:=s.AssetOrganization(a.ID);if m.Favorite{t.Fatal("deleted organization resurrected")};p,_=s.AlbumPage(ids[1],"",10);if p.Count!=0{t.Fatal("dangling membership")};p,_=s.AlbumPage(ids[2],"",10);if p.Count!=0{t.Fatal("empty album lost")}
}
