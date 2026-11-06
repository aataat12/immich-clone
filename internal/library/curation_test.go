package library

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 bolt "go.etcd.io/bbolt"
 "os"
 "path/filepath"
 "reflect"
 "strings"
 "testing"
 "time"
)

func curationEdit(t *testing.T,s *Store,in CurationEdit)any{t.Helper();v,e:=s.EditCuration(in);if e!=nil{t.Fatal(in.Action,e)};return v}
func seedCuration(t *testing.T,s *Store) (string,string){
 t.Helper();assets,e:=s.List();if e!=nil||len(assets)<2{t.Fatal(e,"need two assets")}
 m,e:=s.AssetOrganization(assets[0].ID);if e!=nil{t.Fatal(e)}
 curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:m.AssetID,Added:m.Added,Expected:m.Token,Favorite:true,Title:"Résumé <family>",Caption:"A & B\nFamily reunion",Notes:"Private fixture note\n第二行"})
 id:=fmt.Sprintf("%064x",90001);empty:=fmt.Sprintf("%064x",90002)
 curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:id,Title:"Trip",Description:"Retained family album"})
 curationEdit(t,s,CurationEdit{Action:"create_album",AlbumID:empty,Title:"Empty album"})
 for _,asset:=range assets{p,e:=s.AlbumPage(id,"",100);if e!=nil{t.Fatal(e)};curationEdit(t,s,CurationEdit{Action:"add_member",AlbumID:id,AssetID:asset.ID,Added:asset.Added,Expected:p.Album.Token})}
 p,_:=s.AlbumPage(id,"",100);curationEdit(t,s,CurationEdit{Action:"move_member",AlbumID:id,AssetID:assets[len(assets)-1].ID,OtherID:assets[0].ID,Expected:p.Album.Token});p,_=s.AlbumPage(id,"",100);curationEdit(t,s,CurationEdit{Action:"set_cover",AlbumID:id,AssetID:assets[0].ID,Expected:p.Album.Token})
 return id,empty
}

func TestCurationLazyAtomicVersionAndConflicts(t *testing.T){
 for _,schema:=range []string{"1","2"}{t.Run(schema,func(t *testing.T){s,root:=openTest(t);a,_,e:=s.Put("synthetic.png",bytes.NewReader(fixture(t)));if e!=nil{t.Fatal(e)};s.db.Update(func(tx *bolt.Tx)error{return tx.Bucket([]byte("meta")).Put([]byte("schema"),[]byte(schema))})
  m,_:=s.AssetOrganization(a.ID);curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:m.Token});if automaticSchemaValue(t,s)!=schema{t.Fatal("no-op promoted")}
  // Abort after the ordinary mutation primitives; both format and data roll back.
  e=s.db.Update(func(tx *bolt.Tx)error{if e:=ensureCuration(tx);e!=nil{return e};if e:=curationWrite(tx,assetOrganizationBucket,a.ID,AssetOrganization{AssetID:a.ID,Added:a.Added,Revision:1,Favorite:true});e!=nil{return e};return errors.New("synthetic abort")});if e==nil||automaticSchemaValue(t,s)!=schema{t.Fatal("atomic promotion")}
  v:=curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:m.Token,Favorite:true}).(AssetOrganization);if automaticSchemaValue(t,s)!="3"{t.Fatal("first write")}
  if _,e=s.EditCuration(CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:m.Token,Title:"lost update"});!errors.Is(e,ErrCurationConflict){t.Fatal("stale write",e)}
  curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:v.Token})
  if e=s.db.Update(requireAutomaticCatalog);e!=nil||automaticSchemaValue(t,s)!="3"{t.Fatal("automatic helper lowered format",e)}
  s.Close();again,e:=Open(root);if e!=nil{t.Fatal(e)};defer again.Close();if automaticSchemaValue(t,again)!="3"{t.Fatal("reopen lowered format")};if _,e=again.EditCuration(CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:v.Token});!errors.Is(e,ErrCurationConflict){t.Fatal("restart draft",e)}
 })}
}

func TestCurationExportRestoreDeletionAndOrdering(t *testing.T){
 s,root:=openTest(t);for i:=0;i<3;i++{if _,_,e:=s.Put(fmt.Sprint(i)+".png",bytes.NewReader(append(fixture(t),byte(i))));e!=nil{t.Fatal(e)}}
 id,empty:=seedCuration(t,s);p,_:=s.AlbumPage(id,"",1);if p.Next==""||p.Count!=3{t.Fatal("member paging")};second,e:=s.AlbumPage(id,p.Next,1);if e!=nil||second.Members[0].AssetID==p.Members[0].AssetID{t.Fatal("duplicate member paging",e)}
 before,e:=s.OrganizationExport();if e!=nil{t.Fatal(e)};expected,_:=json.Marshal(before["curation"])
 var archive bytes.Buffer;if e=s.Backup(&archive);e!=nil{t.Fatal(e)};zip:=filepath.Join(t.TempDir(),"backup.zip");if e=os.WriteFile(zip,archive.Bytes(),0600);e!=nil{t.Fatal(e)}
 s.Close();offline,e:=OpenReadOnly(root);if e!=nil{t.Fatal(e)};ro,e:=offline.OrganizationExport();if e!=nil||!reflect.DeepEqual(ro["curation"],before["curation"]){t.Fatal("read-only export",e)};if e=offline.Verify();e!=nil{t.Fatal(e)};offline.Close()
 if e=os.Rename(root,root+".offline");e!=nil{t.Fatal(e)};restored:=filepath.Join(t.TempDir(),"independent");if e=Restore(zip,restored);e!=nil{t.Fatal(e)};r,e:=Open(restored);if e!=nil{t.Fatal(e)};defer r.Close();out,e:=r.OrganizationExport();actual,_:=json.Marshal(out["curation"]);if e!=nil||!bytes.Equal(expected,actual){t.Fatal("independent restore",e)};if e=r.Verify();e!=nil{t.Fatal(e)}
 all,_:=r.List();asset:=all[0];if e=r.SetHidden(asset.ID,true);e!=nil{t.Fatal(e)};if _,e=r.DeleteHidden(context.Background(),asset.ID,asset.ID,asset.Added);e!=nil{t.Fatal(e)}
 p,e=r.AlbumPage(id,"",100);if e!=nil||p.Count!=2||p.Album.Cover!=""{t.Fatal("deletion links/cover",p,e)};empt,e:=r.AlbumPage(empty,"",100);if e!=nil||empt.Count!=0{t.Fatal("empty album lost",e)}
 curationEdit(t,r,CurationEdit{Action:"delete_album",AlbumID:id,Expected:p.Album.Token});if e=r.Verify();e!=nil{t.Fatal("album deletion removed media",e)}
}

func TestCurationValidationAndSemanticFailure(t *testing.T){
 s,_:=openTest(t);a,_,_:=s.Put("a.png",bytes.NewReader(fixture(t)));m,_:=s.AssetOrganization(a.ID)
 for _,text:=range []string{string([]byte{255}),"bad\x00text",strings.Repeat("x",257)}{if _,e:=s.EditCuration(CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:m.Token,Title:text});!errors.Is(e,ErrCurationInput){t.Fatal("invalid text",e)}}
 v:=curationEdit(t,s,CurationEdit{Action:"set_asset",AssetID:a.ID,Added:a.Added,Expected:m.Token,Notes:strings.Repeat("<",64<<10)}).(AssetOrganization)
 if e:=s.db.View(validateCatalogInput);e!=nil{t.Fatal("valid escaped large text rejected",e)}
 if e:=s.db.Update(func(tx *bolt.Tx)error{if e:=curationWrite(tx,assetOrganizationBucket,a.ID,AssetOrganization{AssetID:a.ID,Added:time.Time{},Revision:v.Revision});e!=nil{return e};if e:=validateCatalogInput(tx);e==nil{return errors.New("validator accepted stale incarnation")};return errors.New("rollback fixture")});e==nil{t.Fatal("validation fixture")}
 if e:=s.db.View(validateCatalogInput);e!=nil{t.Fatal("failed fixture committed",e)}
}
