package library

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const assetOrganizationBucket = "asset_organization_v3"
const albumsBucket = "albums_v3"
const albumMembersBucket = "album_members_v3"
const maxCurationBytes = 64 << 20
const maxAlbums = 1000
const maxMemberships = 100000
const maxInitialAlbumMembers = 500

var curationBuckets = []string{assetOrganizationBucket, albumsBucket, albumMembersBucket}
var ErrCurationInput = errors.New("invalid organization metadata")
var ErrCurationConflict = errors.New("organization changed; reload before saving")
var ErrCurationLimit = errors.New("organization storage limit reached")
var ErrCurationCapacity = errors.New("insufficient space for a durable organization write")

type AssetOrganization struct {
	AssetID  string    `json:"asset_id"`
	Added    time.Time `json:"added"`
	Revision uint64    `json:"revision"`
	Favorite bool      `json:"favorite"`
	Title    string    `json:"title"`
	Caption  string    `json:"caption"`
	Notes    string    `json:"notes"`
	Token    string    `json:"token,omitempty"`
}
type Album struct {
	Incarnation  string `json:"incarnation,omitempty"`
	ID           string `json:"id"`
	Revision     uint64 `json:"revision"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Cover        string `json:"cover,omitempty"`
	NextPosition uint64 `json:"next_position"`
	Token        string `json:"token,omitempty"`
}
type AlbumMember struct {
	AlbumID  string    `json:"album_id"`
	AssetID  string    `json:"asset_id"`
	Added    time.Time `json:"added"`
	Position uint64    `json:"position"`
}
type AlbumPage struct {
	Album   Album         `json:"album"`
	Members []AlbumMember `json:"members"`
	Count   int           `json:"count"`
	Next    string        `json:"next,omitempty"`
}
type CurationAlbumTarget struct {
	AssetID string    `json:"asset_id"`
	Added   time.Time `json:"added"`
}
type CurationEdit struct {
	Action      string                `json:"action"`
	Expected    string                `json:"expected"`
	AssetID     string                `json:"asset_id"`
	Added       time.Time             `json:"added"`
	AlbumID     string                `json:"album_id"`
	OtherID     string                `json:"other_id"`
	Favorite    bool                  `json:"favorite"`
	Title       string                `json:"title"`
	Caption     string                `json:"caption"`
	Notes       string                `json:"notes"`
	Description string                `json:"description"`
	Members     []CurationAlbumTarget `json:"members,omitempty"`
}

func curationText(s string, max int, multiline bool) bool {
	if len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) && !(multiline && (c == '\n' || c == '\r' || c == '\t')) {
			return false
		}
	}
	return true
}
func curationAsset(tx *bolt.Tx, id string) (Asset, error) {
	var a Asset
	if !validID(id) {
		return a, ErrCurationInput
	}
	b := tx.Bucket([]byte("assets"))
	if b == nil {
		return a, ErrCurationInput
	}
	raw := b.Get([]byte(id))
	if raw == nil {
		return a, os.ErrNotExist
	}
	if len(raw) > 32<<20 {
		return a, ErrCurationInput
	}
	e := json.Unmarshal(raw, &a)
	return a, e
}
func curationRead(tx *bolt.Tx, bucket, key string, out any) error {
	b := tx.Bucket([]byte(bucket))
	if b == nil {
		return os.ErrNotExist
	}
	return readJSON(b, key, out)
}
func curationToken(tx *bolt.Tx, revision uint64, added time.Time) string {
	return string(tx.Bucket([]byte(galleryBucket)).Get([]byte("epoch"))) + ":" + fmt.Sprint(revision) + ":" + added.UTC().Format(time.RFC3339Nano)
}
func assetOrganization(tx *bolt.Tx, id string) (AssetOrganization, error) {
	a, e := curationAsset(tx, id)
	if e != nil {
		return AssetOrganization{}, e
	}
	out := AssetOrganization{AssetID: id, Added: a.Added}
	if e = curationRead(tx, assetOrganizationBucket, id, &out); e != nil && !errors.Is(e, os.ErrNotExist) {
		return out, e
	}
	out.Token = curationToken(tx, out.Revision, a.Added)
	return out, nil
}
func albumEditToken(tx *bolt.Tx, a Album) string {
	return curationToken(tx, a.Revision, time.Time{}) + ":" + a.Incarnation
}
func curationAlbum(tx *bolt.Tx, id string) (Album, error) {
	var a Album
	if !validID(id) {
		return a, ErrCurationInput
	}
	e := curationRead(tx, albumsBucket, id, &a)
	if e == nil {
		a.Token = albumEditToken(tx, a)
	}
	return a, e
}
func (s *Store) AssetOrganization(id string) (AssetOrganization, error) {
	var out AssetOrganization
	e := s.db.View(func(tx *bolt.Tx) error { var e error; out, e = assetOrganization(tx, id); return e })
	return out, e
}
func (s *Store) Albums(after string, limit int) ([]Album, string, error) {
	out := []Album{}
	next := ""
	if (after != "" && !validID(after)) || limit < 1 || limit > 100 {
		return out, next, ErrCurationInput
	}
	e := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(albumsBucket))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		k, v := c.First()
		if after != "" {
			if b.Get([]byte(after)) == nil {
				return ErrCurationConflict
			}
			c.Seek([]byte(after))
			k, v = c.Next()
		}
		for ; k != nil; k, v = c.Next() {
			if len(out) == limit {
				next = out[len(out)-1].ID
				break
			}
			var a Album
			if e := json.Unmarshal(v, &a); e != nil {
				return e
			}
			a.Token = albumEditToken(tx, a)
			out = append(out, a)
		}
		return nil
	})
	return out, next, e
}
func albumMembers(tx *bolt.Tx, id string) ([]AlbumMember, error) {
	out := []AlbumMember{}
	b := tx.Bucket([]byte(albumMembersBucket))
	if b == nil {
		return out, nil
	}
	c := b.Cursor()
	prefix := []byte(id + ":")
	for k, v := c.Seek(prefix); k != nil && strings.HasPrefix(string(k), string(prefix)); k, v = c.Next() {
		var m AlbumMember
		if e := json.Unmarshal(v, &m); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}
func (s *Store) AlbumPage(id, after string, limit int) (AlbumPage, error) {
	out := AlbumPage{Members: []AlbumMember{}}
	if limit < 1 || limit > 100 || len(after) > 512 {
		return out, ErrCurationInput
	}
	e := s.db.View(func(tx *bolt.Tx) error {
		a, e := curationAlbum(tx, id)
		if e != nil {
			return e
		}
		members, e := albumMembers(tx, id)
		if e != nil {
			return e
		}
		start := 0
		if after != "" {
			raw, e := base64.RawURLEncoding.DecodeString(after)
			var c struct {
				Token  string
				Offset int
			}
			if e != nil || json.Unmarshal(raw, &c) != nil || c.Offset < 0 || c.Offset > len(members) {
				return ErrCurationInput
			}
			if c.Token != a.Token {
				return ErrCurationConflict
			}
			start = c.Offset
		}
		end := start + limit
		if end > len(members) {
			end = len(members)
		}
		out.Album = a
		out.Count = len(members)
		out.Members = append(out.Members, members[start:end]...)
		if end < len(members) {
			b, _ := json.Marshal(struct {
				Token  string
				Offset int
			}{a.Token, end})
			out.Next = base64.RawURLEncoding.EncodeToString(b)
		}
		return nil
	})
	return out, e
}

func ensureCuration(tx *bolt.Tx) error {
	meta := tx.Bucket([]byte("meta"))
	v := meta.Get([]byte("schema"))
	if !supportedCatalogSchema(v) {
		return ErrCurationInput
	}
	if string(v) == "3" {
		return nil
	}
	for _, name := range curationBuckets {
		if _, e := tx.CreateBucket([]byte(name)); e != nil {
			return e
		}
	}
	if e := setGalleryNumber(meta, "curation_bytes", 0); e != nil {
		return e
	}
	return meta.Put([]byte("schema"), []byte("3"))
}
func curationWrite(tx *bolt.Tx, bucket, key string, value any) error {
	b := tx.Bucket([]byte(bucket))
	var raw []byte
	var e error
	if value != nil {
		raw, e = json.Marshal(value)
		if e != nil {
			return e
		}
	}
	meta := tx.Bucket([]byte("meta"))
	size := galleryNumber(meta, "curation_bytes")
	old := uint64(len(b.Get([]byte(key))))
	if size < old {
		return ErrCurationInput
	}
	size = size - old + uint64(len(raw))
	if size > maxCurationBytes || len(raw) > 1<<20 || (uint64(len(raw)) > old && tx.Size() > MaxUpload-(4<<20)) {
		return ErrCurationLimit
	}
	if value == nil {
		e = b.Delete([]byte(key))
	} else {
		e = b.Put([]byte(key), raw)
	}
	if e != nil {
		return e
	}
	return setGalleryNumber(meta, "curation_bytes", size)
}

// Every mutation is an explicit operation and one atomic catalog transaction.
// New records, byte accounting and the version gate commit or roll back together.
func (s *Store) EditCuration(in CurationEdit) (any, error) {
	if s.offline != nil {
		return nil, ErrCurationInput
	}
	if len(in.Expected) > 256 || !curationText(in.Title, 256, false) || !curationText(in.Caption, 16<<10, true) || !curationText(in.Notes, 64<<10, true) || !curationText(in.Description, 4096, true) {
		return nil, ErrCurationInput
	}
	if len(in.Members) > maxInitialAlbumMembers || len(in.Members) != 0 && in.Action != "create_album" {
		return nil, ErrCurationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	// As with recoverable deletion, do not start database growth on already
	// exhausted storage: a Windows mapping failure can invalidate the handle.
	if free, e := AvailableBytes(s.root); e != nil {
		return nil, e
	} else if free < 16<<20 {
		return nil, ErrCurationCapacity
	}
	var out any
	e := s.db.Update(func(tx *bolt.Tx) error { var e error; out, e = editCurationTx(tx, in); return e })
	return out, e
}
func editCurationTx(tx *bolt.Tx, in CurationEdit) (any, error) {
	var out any
	e := func() error {
		if in.Action == "set_asset" {
			a, e := assetOrganization(tx, in.AssetID)
			if e != nil {
				return e
			}
			if in.Expected != a.Token || !in.Added.Equal(a.Added) {
				return ErrCurationConflict
			}
			if a.Favorite == in.Favorite && a.Title == in.Title && a.Caption == in.Caption && a.Notes == in.Notes {
				out = a
				return nil
			}
			if a.Revision == ^uint64(0) {
				return ErrCurationLimit
			}
			if e = ensureCuration(tx); e != nil {
				return e
			}
			a.Revision++
			a.Favorite = in.Favorite
			a.Title = in.Title
			a.Caption = in.Caption
			a.Notes = in.Notes
			a.Token = ""
			if e = curationWrite(tx, assetOrganizationBucket, in.AssetID, a); e != nil {
				return e
			}
			a.Token = curationToken(tx, a.Revision, a.Added)
			out = a
			return nil
		}
		if !validID(in.AlbumID) {
			return ErrCurationInput
		}
		a, e := curationAlbum(tx, in.AlbumID)
		if in.Action == "create_album" {
			if strings.TrimSpace(in.Title) == "" || in.Expected != "" {
				return ErrCurationInput
			}
			if e == nil {
				if a.Title != in.Title || a.Description != in.Description || a.Revision != 1 || a.NextPosition != uint64(len(in.Members)) {
					return ErrCurationConflict
				}
				members, err := albumMembers(tx, a.ID)
				if err != nil {
					return err
				}
				if len(members) != len(in.Members) {
					return ErrCurationConflict
				}
				for i, member := range members {
					if member.AssetID != in.Members[i].AssetID || !member.Added.Equal(in.Members[i].Added) {
						return ErrCurationConflict
					}
				}
				out = a
				return nil
			}
			if !errors.Is(e, os.ErrNotExist) {
				return e
			}
			if e = ensureCuration(tx); e != nil {
				return e
			}
			if tx.Bucket([]byte(albumsBucket)).Stats().KeyN >= maxAlbums {
				return ErrCurationLimit
			}
			incarnation, e := newFaceID()
			if e != nil {
				return e
			}
			a = Album{ID: in.AlbumID, Incarnation: incarnation, Revision: 1, Title: in.Title, Description: in.Description}
			if e = writeInitialMembers(tx, &a, in.Members); e != nil {
				return e
			}
			if e = curationWrite(tx, albumsBucket, a.ID, a); e != nil {
				return e
			}
			a.Token = albumEditToken(tx, a)
			out = a
			return nil
		}
		if e != nil {
			return e
		}
		if in.Expected != a.Token {
			return ErrCurationConflict
		}
		if a.Revision == ^uint64(0) {
			return ErrCurationLimit
		}
		members, e := albumMembers(tx, a.ID)
		if e != nil {
			return e
		}
		key := a.ID + ":" + in.AssetID
		changed := true
		switch in.Action {
		case "update_album":
			if strings.TrimSpace(in.Title) == "" {
				return ErrCurationInput
			}
			changed = a.Title != in.Title || a.Description != in.Description
			a.Title = in.Title
			a.Description = in.Description
		case "delete_album":
			for _, m := range members {
				if e = curationWrite(tx, albumMembersBucket, a.ID+":"+m.AssetID, nil); e != nil {
					return e
				}
			}
			out = map[string]bool{"deleted": true}
			return curationWrite(tx, albumsBucket, a.ID, nil)
		case "set_cover":
			if in.AssetID != "" {
				found := false
				for _, m := range members {
					if m.AssetID == in.AssetID {
						found = true
					}
				}
				if !found {
					return ErrCurationInput
				}
			}
			changed = a.Cover != in.AssetID
			a.Cover = in.AssetID
		case "add_member":
			asset, e := curationAsset(tx, in.AssetID)
			if e != nil {
				return e
			}
			if !asset.Added.Equal(in.Added) {
				return ErrCurationConflict
			}
			for _, m := range members {
				if m.AssetID == in.AssetID {
					out = a
					return nil
				}
			}
			if len(members) >= 5000 || tx.Bucket([]byte(albumMembersBucket)).Stats().KeyN >= maxMemberships || a.NextPosition == ^uint64(0) {
				return ErrCurationLimit
			}
			m := AlbumMember{AlbumID: a.ID, AssetID: asset.ID, Added: asset.Added, Position: a.NextPosition}
			a.NextPosition++
			if e = curationWrite(tx, albumMembersBucket, key, m); e != nil {
				return e
			}
		case "remove_member":
			found := false
			for _, m := range members {
				if m.AssetID == in.AssetID {
					found = true
				}
			}
			changed = found
			if found {
				if e = curationWrite(tx, albumMembersBucket, key, nil); e != nil {
					return e
				}
				if a.Cover == in.AssetID {
					a.Cover = ""
				}
			}
		case "move_member":
			i, j := -1, -1
			for n, m := range members {
				if m.AssetID == in.AssetID {
					i = n
				}
				if m.AssetID == in.OtherID {
					j = n
				}
			}
			if i < 0 || j < 0 {
				return ErrCurationInput
			}
			changed = i != j
			if changed {
				moving := members[i]
				if i < j {
					copy(members[i:j], members[i+1:j+1])
				} else {
					copy(members[j+1:i+1], members[j:i])
				}
				members[j] = moving
				for n, m := range members {
					m.Position = uint64(n)
					if e = curationWrite(tx, albumMembersBucket, a.ID+":"+m.AssetID, m); e != nil {
						return e
					}
				}
				a.NextPosition = uint64(len(members))
			}
		default:
			return ErrCurationInput
		}
		if changed {
			a.Revision++
			a.Token = ""
			if e = curationWrite(tx, albumsBucket, a.ID, a); e != nil {
				return e
			}
			a.Token = albumEditToken(tx, a)
		}
		out = a
		return nil
	}()
	return out, e
}

func writeInitialMembers(tx *bolt.Tx, album *Album, targets []CurationAlbumTarget) error {
	if tx.Bucket([]byte(albumMembersBucket)).Stats().KeyN+len(targets) > maxMemberships {
		return ErrCurationLimit
	}
	seen := make(map[string]bool, len(targets))
	for i, target := range targets {
		if seen[target.AssetID] {
			return ErrCurationInput
		}
		seen[target.AssetID] = true
		asset, err := curationAsset(tx, target.AssetID)
		if errors.Is(err, os.ErrNotExist) {
			return ErrCurationConflict
		}
		if err != nil {
			return err
		}
		if asset.Hidden || !asset.Added.Equal(target.Added) {
			return ErrCurationConflict
		}
		member := AlbumMember{AlbumID: album.ID, AssetID: asset.ID, Added: asset.Added, Position: uint64(i)}
		if err := curationWrite(tx, albumMembersBucket, album.ID+":"+asset.ID, member); err != nil {
			return err
		}
	}
	album.NextPosition = uint64(len(targets))
	return nil
}

func deleteAssetCuration(tx *bolt.Tx, id string) error {
	if tx.Bucket([]byte(assetOrganizationBucket)) == nil {
		return nil
	}
	if e := curationWrite(tx, assetOrganizationBucket, id, nil); e != nil {
		return e
	}
	var remove []AlbumMember
	if e := tx.Bucket([]byte(albumMembersBucket)).ForEach(func(k, v []byte) error {
		var m AlbumMember
		if e := json.Unmarshal(v, &m); e != nil {
			return e
		}
		if m.AssetID == id {
			remove = append(remove, m)
		}
		return nil
	}); e != nil {
		return e
	}
	for _, m := range remove {
		a, e := curationAlbum(tx, m.AlbumID)
		if e != nil {
			return e
		}
		if a.Revision == ^uint64(0) {
			return ErrCurationLimit
		}
		a.Revision++
		if a.Cover == id {
			a.Cover = ""
		}
		a.Token = ""
		if e = curationWrite(tx, albumsBucket, a.ID, a); e != nil {
			return e
		}
		if e = curationWrite(tx, albumMembersBucket, a.ID+":"+id, nil); e != nil {
			return e
		}
	}
	return nil
}

func exportCuration(tx *bolt.Tx) (map[string]any, error) {
	out := map[string]any{"version": 1}
	for _, name := range append(append([]string{}, curationBuckets...), organizationBatchesBucket) {
		rows := map[string]json.RawMessage{}
		if b := tx.Bucket([]byte(name)); b != nil {
			if e := b.ForEach(func(k, v []byte) error { rows[string(k)] = append([]byte(nil), v...); return nil }); e != nil {
				return nil, e
			}
		}
		out[name] = rows
	}
	return out, nil
}

func validateCuration(tx *bolt.Tx) error {
	meta := tx.Bucket([]byte("meta"))
	version := ""
	if meta != nil {
		version = string(meta.Get([]byte("schema")))
	}
	if version != "" && version != "1" && version != "2" && version != "3" {
		return errCatalogInput
	}
	if version != "3" {
		for _, name := range append(append([]string{}, curationBuckets...), organizationBatchesBucket) {
			if tx.Bucket([]byte(name)) != nil {
				return errCatalogInput
			}
		}
		return nil
	}
	size, batchErr := validateOrganizationBatches(tx)
	if batchErr != nil {
		return batchErr
	}
	for _, name := range curationBuckets {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return errCatalogInput
		}
		if e := b.ForEach(func(k, v []byte) error {
			if v == nil || len(v) > 1<<20 || !utf8.Valid(v) {
				return errCatalogInput
			}
			size += uint64(len(v))
			return nil
		}); e != nil {
			return e
		}
	}
	if size > maxCurationBytes || len(meta.Get([]byte("curation_bytes"))) != 8 || size != galleryNumber(meta, "curation_bytes") || tx.Bucket([]byte(albumsBucket)).Stats().KeyN > maxAlbums || tx.Bucket([]byte(albumMembersBucket)).Stats().KeyN > maxMemberships {
		return errCatalogInput
	}
	if e := tx.Bucket([]byte(assetOrganizationBucket)).ForEach(func(k, v []byte) error {
		var a AssetOrganization
		if json.Unmarshal(v, &a) != nil || a.AssetID != string(k) || a.Revision == 0 || a.Token != "" || !curationText(a.Title, 256, false) || !curationText(a.Caption, 16<<10, true) || !curationText(a.Notes, 64<<10, true) {
			return errCatalogInput
		}
		asset, e := curationAsset(tx, a.AssetID)
		if e != nil || !asset.Added.Equal(a.Added) {
			return errCatalogInput
		}
		return nil
	}); e != nil {
		return e
	}
	albums := map[string]Album{}
	positions := map[string]map[uint64]bool{}
	covers := map[string]bool{}
	if e := tx.Bucket([]byte(albumsBucket)).ForEach(func(k, v []byte) error {
		var a Album
		if json.Unmarshal(v, &a) != nil || a.ID != string(k) || !validID(a.ID) || (a.Incarnation != "" && !validID(a.Incarnation)) || a.Revision == 0 || a.Token != "" || strings.TrimSpace(a.Title) == "" || !curationText(a.Title, 256, false) || !curationText(a.Description, 4096, true) || (a.Cover != "" && !validID(a.Cover)) {
			return errCatalogInput
		}
		albums[a.ID] = a
		positions[a.ID] = map[uint64]bool{}
		return nil
	}); e != nil {
		return e
	}
	if e := tx.Bucket([]byte(albumMembersBucket)).ForEach(func(k, v []byte) error {
		var m AlbumMember
		if json.Unmarshal(v, &m) != nil || string(k) != m.AlbumID+":"+m.AssetID {
			return errCatalogInput
		}
		a, ok := albums[m.AlbumID]
		if !ok || m.Position >= a.NextPosition || positions[a.ID][m.Position] {
			return errCatalogInput
		}
		asset, e := curationAsset(tx, m.AssetID)
		if e != nil || !asset.Added.Equal(m.Added) {
			return errCatalogInput
		}
		if len(positions[a.ID]) >= 5000 {
			return errCatalogInput
		}
		positions[a.ID][m.Position] = true
		if a.Cover == m.AssetID {
			covers[a.ID] = true
		}
		return nil
	}); e != nil {
		return e
	}
	for _, a := range albums {
		if a.Cover != "" && !covers[a.ID] {
			return errCatalogInput
		}
	}
	return nil
}
