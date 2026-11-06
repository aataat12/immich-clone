package library

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"immichclone/internal/recognition"
	"math"
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func automaticTestFace(id int, asset string, axis int) StoredFace {
	v := make([]float64, 128)
	v[axis] = 1
	return StoredFace{ID: fmt.Sprintf("%064x", id), AssetID: asset, Box: []float64{.1, .1, .2, .2}, Embedding: v, Provider: recognition.AnalysisID}
}

func TestAutomaticNamePolicyRequiresIndependentTrustedReferences(t *testing.T) {
	p := candidateAutomaticNamePolicy()
	subject := automaticTestFace(9, "query", 0)
	a := reviewReference{automaticTestFace(1, "first", 0), FaceLabel{Name: "Alice"}}
	b := reviewReference{automaticTestFace(2, "second", 0), FaceLabel{Name: "Alice", Source: "manual"}}
	if name, _ := matchAutomaticFace(subject, []reviewReference{a,b}, p); name != "Alice" { t.Fatal("strong independently confirmed references rejected") }
	tests := map[string][]reviewReference{
		"single reference": {a},
		"same original": {a, {automaticTestFace(2, "first", 0), b.l}},
		"self source": {a, {automaticTestFace(2, "query", 0), b.l}},
		"automatic reference": {a, {b.f, FaceLabel{Name:"Alice", Source:"automatic"}}},
		"dismissed reference": {a, {b.f, FaceLabel{Name:"Alice", Dismissed:true}}},
		"competing identity": {a,b,{automaticTestFace(3,"other",0),FaceLabel{Name:"Bob"}}},
	}
	for title, refs := range tests { t.Run(title, func(t *testing.T) { if name,_ := matchAutomaticFace(subject,refs,p); name != "" { t.Fatal("unsafe assignment") } }) }
	bad := b; bad.f.Provider = "unrelated-provider"
	if name,_ := matchAutomaticFace(subject,[]reviewReference{a,bad},p); name != "" { t.Fatal("incompatible provider accepted") }
	bad = b; bad.f.Embedding = append([]float64(nil), b.f.Embedding...); bad.f.Embedding[0] = math.NaN()
	if name,_ := matchAutomaticFace(subject,[]reviewReference{a,bad},p); name != "" { t.Fatal("nonfinite embedding accepted") }
	if name,_ := matchAutomaticFace(subject,[]reviewReference{a,b},productionAutomaticNamePolicy()); name != "" { t.Fatal("unvalidated production gate bypassed") }
}

func seedAutomaticNames(t *testing.T) (*Store, string, []StoredFace) {
	t.Helper()
	s, root := openTest(t)
	assets := []Asset{}
	for i:=0; i<5; i++ {
		data := append(append([]byte(nil),fixture(t)...),byte(i))
		a,_,e := s.Put(fmt.Sprintf("synthetic-%d.png",i),bytes.NewReader(data)); if e != nil { t.Fatal(e) }
		assets = append(assets,a)
	}
	faces := []StoredFace{
		automaticTestFace(1,assets[0].ID,0), automaticTestFace(2,assets[1].ID,0),
		automaticTestFace(3,assets[2].ID,1), automaticTestFace(4,assets[3].ID,1),
		automaticTestFace(5,assets[4].ID,0), automaticTestFace(6,assets[4].ID,1),
	}
	if e := s.db.Update(func(tx *bolt.Tx) error {
		for i,f := range faces {
			if e := writeJSON(tx.Bucket([]byte(facesBucket)),f.ID,f); e != nil { return e }
			if i<4 { name:="Alice"; if i>=2 {name="Bob"}; if e:=writeJSON(tx.Bucket([]byte(labelsBucket)),f.ID,FaceLabel{Name:name}); e!=nil{return e} }
		}
		return applyAutomaticNames(tx,faces[4:],candidateAutomaticNamePolicy())
	}); e != nil { t.Fatal(e) }
	return s,root,faces
}

func automaticLabel(t *testing.T,s *Store,id string) FaceLabel {
	t.Helper(); var l FaceLabel
	if e:=s.db.View(func(tx *bolt.Tx) error{return readJSON(tx.Bucket([]byte(labelsBucket)),id,&l)}); e!=nil {t.Fatal(e)}
	return l
}

func TestAutomaticNamesMultiplePeopleCorrectionsAndUndo(t *testing.T) {
	s,_,faces := seedAutomaticNames(t)
	for i,name:=range []string{"Alice","Bob"} {
		l:=automaticLabel(t,s,faces[4+i].ID)
		if l.Name!=name || l.Source!="automatic" || l.Policy!=automaticNamePolicyID || !l.CanUndo {t.Fatal("automatic provenance missing")}
	}
	id:=faces[4].ID
	if e:=s.LabelFaces([]string{id},"Alice"); e!=nil {t.Fatal(e)}
	if l:=automaticLabel(t,s,id); !trustedFaceLabel(l) {t.Fatal("same-name confirmation did not promote reference")}
	if e:=s.UndoFace(id); e!=nil {t.Fatal(e)}
	if l:=automaticLabel(t,s,id); l.Source!="automatic" || trustedFaceLabel(l) {t.Fatal("undo lost previous provenance")}
	if e:=s.LabelFaces([]string{id},"Corrected person"); e!=nil {t.Fatal(e)}
	if e:=s.db.Update(func(tx *bolt.Tx)error{return applyAutomaticNames(tx,faces[4:],candidateAutomaticNamePolicy())}); e!=nil{t.Fatal(e)}
	if l:=automaticLabel(t,s,id); l.Name!="Corrected person" || l.Source!="manual" {t.Fatal("explicit correction overwritten")}
	if e:=s.LabelFaces([]string{id},""); e!=nil{t.Fatal(e)}
	if e:=s.db.Update(func(tx *bolt.Tx)error{return applyAutomaticNames(tx,faces[4:],candidateAutomaticNamePolicy())}); e!=nil{t.Fatal(e)}
	if l:=automaticLabel(t,s,id); l.Name!=""{t.Fatal("explicit cleared name reassigned")}
	if e:=s.UndoFace(faces[5].ID); e!=nil{t.Fatal(e)}
	if e:=s.db.Update(func(tx *bolt.Tx)error{return applyAutomaticNames(tx,faces[4:],candidateAutomaticNamePolicy())}); e!=nil{t.Fatal(e)}
	if l:=automaticLabel(t,s,faces[5].ID); l.Name!=""{t.Fatal("undone automatic name reassigned")}
}

func TestAutomaticNamesDoNotBecomeReferencesAndSurviveRestore(t *testing.T) {
	s,root,faces := seedAutomaticNames(t)
	page,e:=s.OrganizationPage("","",""); if e!=nil{t.Fatal(e)}
	auto:=0; for _,f:=range page.Faces {if f.Source=="automatic"{auto++}}
	if auto!=2 {t.Fatal("API omitted automatic provenance")}
	if e:=s.db.View(func(tx *bolt.Tx)error{
		index,e:=s.organizationIndexFor(tx);if e!=nil{return e}
		if len(index.references)!=4 {t.Fatal("automatic names entered reference index")};return nil
	});e!=nil{t.Fatal(e)}
	export,e:=s.OrganizationExport();if e!=nil{t.Fatal(e)}
	encoded,_:=json.Marshal(export)
	if !bytes.Contains(encoded,[]byte(`"name_source":"automatic"`))||bytes.Contains(encoded,[]byte(`"embedding":[`)){t.Fatal("export provenance/embedding contract")}
	var backup bytes.Buffer; if e=s.Backup(&backup);e!=nil{t.Fatal(e)}
	s.Close()
	s,e=Open(root);if e!=nil{t.Fatal(e)}
	if l:=automaticLabel(t,s,faces[4].ID);l.Source!="automatic"{t.Fatal("restart lost provenance")}
	s.Close()
	archive:=filepath.Join(t.TempDir(),"backup.zip");if e=os.WriteFile(archive,backup.Bytes(),0600);e!=nil{t.Fatal(e)}
	target:=filepath.Join(t.TempDir(),"restored");if e=Restore(archive,target);e!=nil{t.Fatal(e)}
	if e=os.Rename(root,root+"-unavailable");e!=nil{t.Fatal(e)}
	restored,e:=Open(target);if e!=nil{t.Fatal(e)};defer restored.Close()
	if l:=automaticLabel(t,restored,faces[5].ID);l.Name!="Bob"||l.Source!="automatic"||l.Policy!=automaticNamePolicyID{t.Fatal("independent restore lost provenance")}
	if e=restored.Verify();e!=nil{t.Fatal(e)}
}

func TestAutomaticNamesReanalysisKeepsExistingCorrections(t *testing.T) {
	s,id,face:=seedReviewedPhoto(t)
	s.SetRecognizer(proposalAnalysis,true)
	before:=automaticLabel(t,s,face)
	if e:=s.ControlReanalysis(id,"start");e!=nil{t.Fatal(e)}
	if _,e:=s.ProcessAnalysis(context.Background());e!=nil{t.Fatal(e)}
	if e:=s.ControlReanalysis(id,"apply");e!=nil{t.Fatal(e)}
	after:=automaticLabel(t,s,face)
	if before!=after{t.Fatal("reanalysis altered explicit correction or dismissal")}
}

func TestAutomaticNamingGateLeavesNewAnalysisUnassigned(t *testing.T) {
	s,_,_:=seedAutomaticNames(t)
	s.SetRecognizer(fakeAnalysis,true)
	data:=append(append([]byte(nil),fixture(t)...),byte(99))
	a,_,e:=s.Put("new-import.png",bytes.NewReader(data));if e!=nil{t.Fatal(e)}
	if e=s.QueueAnalysis(a.ID);e!=nil{t.Fatal(e)}
	if _,e=s.ProcessAnalysis(context.Background());e!=nil{t.Fatal(e)}
	page,e:=s.OrganizationPage("","","");if e!=nil{t.Fatal(e)}
	found:=false
	for _,f:=range page.Faces{if f.AssetID==a.ID{found=true;if f.Name!=""||f.Source!=""{t.Fatal("unvalidated automatic naming ran during new-import analysis")}}}
	if !found{t.Fatal("new-import analysis did not publish face")}
	if page.AutomaticNaming{t.Fatal("API claims automatic naming enabled")}
}
