package library

import (
 "encoding/json"
 "fmt"
 bolt "go.etcd.io/bbolt"
 "os"
 "path/filepath"
 "reflect"
 "testing"
)

// Explicit disposable-runner fixture; never a production migration switch.
func TestCurationPublishedFixture(t *testing.T){
 root:=os.Getenv("IMMICH_CURATION_ROOT");stage:=os.Getenv("IMMICH_CURATION_STAGE")
 if root==""{t.Skip("requires explicit disposable CI fixture")};if os.Getenv("GITHUB_ACTIONS")!="true"||!filepath.IsAbs(root){t.Fatal("disposable CI path required")}
 switch stage{
 case "create1","create2":s,subject:=automaticSchemaFixture(t,root);if stage=="create2"{if e:=s.db.Update(func(tx *bolt.Tx)error{return applyAutomaticNames(tx,[]StoredFace{subject},candidateAutomaticNamePolicy())});e!=nil{t.Fatal(e)}};s.Close()
 case "promote3":s,e:=Open(root);if e!=nil{t.Fatal(e)};defer s.Close();seedCuration(t,s);if automaticSchemaValue(t,s)!="3"{t.Fatal("format not promoted")}
 case "inspect":s,e:=OpenReadOnly(root);if e!=nil{t.Fatal(e)};defer s.Close();if e=s.Verify();e!=nil{t.Fatal(e)};out,e:=s.OrganizationExport();if e!=nil{t.Fatal(e)};raw,e:=os.ReadFile(os.Getenv("IMMICH_CURATION_EXPECTED"));if e!=nil{t.Fatal(e)};var expected,actual any;if e=json.Unmarshal(raw,&expected);e!=nil{t.Fatal(e)};b,_:=json.Marshal(out["curation"]);json.Unmarshal(b,&actual);if !reflect.DeepEqual(expected,actual){t.Fatal("complete restored organization mismatch")};if automaticSchemaValue(t,s)!="3"{t.Fatal("restored schema")}
 default:t.Fatal(fmt.Sprint("unknown fixture stage: ",stage))
 }
}
