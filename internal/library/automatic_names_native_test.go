package library

import (
	"encoding/json"
	"fmt"
	"testing"
)

type automaticQualityRecord struct {
	subject, index int
	face *StoredFace
}

// This separate frozen policy screen reuses native inference already performed
// by TestActualNativeCrossPhotoQuality. No per-photo predictions or embeddings
// are logged. The reviewed ORL corpus is a screen, not unseen general-library
// evidence, and passing never changes the production enablement constant.
func evaluateAutomaticNativeQuality(t *testing.T, all []automaticQualityRecord) {
	t.Helper()
	p:=candidateAutomaticNamePolicy()
	refs:=[]reviewReference{}
	detected:=0
	for _,r:=range all {
		if r.face==nil {continue};detected++
		if r.subject>=11&&r.subject<=30&&r.index<=2 {
			f:=*r.face;f.ID=fmt.Sprintf("%064x",r.subject*10+r.index);f.AssetID=f.ID
			refs=append(refs,reviewReference{f,FaceLabel{Name:fmt.Sprintf("Identity %d",r.subject),Source:"manual"}})
		}
	}
	correct,wrong,unresolved,unknownAccept,unknownRejected,unknownUnprocessed:=0,0,0,0,0,0
	for _,r:=range all {
		if r.subject<=10||r.subject<=30&&r.index<=2 {continue}
		known:=r.subject<=30
		if r.face==nil {if known{unresolved++}else{unknownUnprocessed++};continue}
		f:=*r.face;f.ID=fmt.Sprintf("%064x",r.subject*10+r.index);f.AssetID=f.ID
		name,_:=matchAutomaticFace(f,refs,p)
		if known {
			if name==""{unresolved++}else if name==fmt.Sprintf("Identity %d",r.subject){correct++}else{wrong++}
		}else if name==""{unknownRejected++}else{unknownAccept++}
	}
	if correct+wrong+unresolved!=160||unknownAccept+unknownRejected+unknownUnprocessed!=100{t.Fatal("automatic quality denominator accounting invalid")}
	pass:=detected>=360&&len(refs)==40&&correct>=80&&wrong==0&&unknownAccept==0
	report:=map[string]any{"scope":"separate frozen automatic-name policy screen on previously studied ORL corpus; not general-library acceptance","policy":automaticNamePolicyID,"threshold":p.threshold,"margin":p.margin,"minimum_independent_sources":2,"corpus_exactly_one_face":detected,"enrollments_available":len(refs),"known_total":160,"known_correct":correct,"known_wrong":wrong,"known_unresolved":unresolved,"unknown_total":100,"unknown_false_accept":unknownAccept,"unknown_rejected":unknownRejected,"unknown_unprocessed":unknownUnprocessed,"narrow_automatic_screen_passed":pass,"production_automatic_naming_enabled":automaticNamesValidated}
	b,e:=json.Marshal(report);if e!=nil{t.Fatal(e)};t.Log(string(b))
	if !pass{t.Error("frozen automatic-name policy screen failed; policy left unchanged and automatic naming remains disabled")}
}
