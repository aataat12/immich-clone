package library

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	bolt "go.etcd.io/bbolt"
)

func TestGallerySearchAssignedPeopleTagsVisibilityAndChanges(t *testing.T) {
	s, _ := openTest(t)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, e := s.Put("BeachTrip.png", bytes.NewReader(fixture(t)))
	if e != nil { t.Fatal(e) }
	if e = s.QueueAnalysis(a.ID); e != nil { t.Fatal(e) }
	if _, e = s.ProcessAnalysis(context.Background()); e != nil { t.Fatal(e) }
	p, e := s.OrganizationPage("", "", "")
	if e != nil || len(p.Faces)!=1 { t.Fatal(e, p) }
	if e=s.LabelFaces([]string{p.Faces[0].ID}, "Alice"); e!=nil { t.Fatal(e) }
	e=s.ManualFace(context.Background(), a.ID, []float64{.5,.5,.2,.2}, "Bob")
	if e!=nil { t.Fatal(e) }
	if e=s.SetTags(a.ID, []string{"Résumé trip"}); e!=nil { t.Fatal(e) }
	check:=func(query, visibility string, count uint64) {
		t.Helper(); p,e:=s.SearchGalleryPage("name",visibility,"",query,1)
		if e!=nil || p.Count!=count || uint64(len(p.Assets))!=count { t.Fatalf("%s/%s: %+v %v",query,visibility,p,e) }
	}
	for _, q:=range []string{"alice","BOB","beachtrip","RÉSUMÉ","cat"} { check(q,"library",1) }
	check("nonexistent","library",0)
	if e=s.DismissTag(a.ID,"cat",false); e!=nil {t.Fatal(e)}
	check("cat","library",0)
	if e=s.DismissFace(p.Faces[0].ID,false); e!=nil {t.Fatal(e)}
	check("Alice","library",0)
	check("Bob","library",1)
	if e=s.SetHidden(a.ID,true); e!=nil {t.Fatal(e)}
	check("Bob","library",0); check("Bob","hidden",1); check("résumé","hidden",1)
}

func TestGallerySearchWholeCatalogBoundedPagingAndCursor(t *testing.T) {
	s,_:=openTest(t)
	seedGallery(t,s,10003)
	if e:=s.db.Update(func(tx *bolt.Tx)error{
		for _,i:=range []int{2,5000,9999} {
			if e:=writeJSON(tx.Bucket([]byte(tagsBucket)),fmt.Sprintf("%064x",i),[]string{"far-match"});e!=nil{return e}
		}
		return nil
	});e!=nil {t.Fatal(e)}
	after:="";seen:=map[string]bool{}
	for {
		p,e:=s.SearchGalleryPage("name","library",after,"far-match",1)
		if e!=nil||p.Count!=3||len(p.Assets)!=1 {t.Fatal(p,e)}
		if seen[p.Assets[0].ID] {t.Fatal("duplicate search row")};seen[p.Assets[0].ID]=true
		if p.Next=="" {break};after=p.Next
	}
	if len(seen)!=3 {t.Fatal("incomplete filtered pagination")}
	p,e:=s.SearchGalleryPage("name","library","","far-match",1)
	if e!=nil {t.Fatal(e)}
	if _,e=s.SearchGalleryPage("name","library",p.Next,"different",1); !errors.Is(e,ErrGalleryInput){t.Fatal("query cursor reuse",e)}
	if e=s.SetTags(p.Assets[0].ID,[]string{"corrected"});e!=nil{t.Fatal(e)}
	if _,e=s.SearchGalleryPage("name","library",p.Next,"far-match",1); !errors.Is(e,ErrGalleryChanged){t.Fatal("stale metadata cursor",e)}
	if _,e=s.SearchGalleryPage("name","library","",string([]byte{255}),1);!errors.Is(e,ErrGalleryInput){t.Fatal("invalid UTF8",e)}
}
